package database

import (
	"context"
	"embed"
	"linkshortener/internal/types"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	clickmigrations "github.com/golang-migrate/migrate/v4/database/clickhouse"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/oschwald/geoip2-golang"
)

//go:embed migrations/clickhouse/*.sql
var migrationsClickHouseFS embed.FS

type Analytics struct {
	db           *sqlx.DB
	clicksBuffer chan types.ClickData
	geo          *geoip2.Reader
	workerCancel context.CancelFunc
	workerWG     sync.WaitGroup
	startOnce    sync.Once
	closeOnce    sync.Once
}

func ConnectClickHouse(ctx context.Context, addr, user, pass, dbName string) (*Analytics, error) {
	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: dbName,
			Username: user,
			Password: pass,
		},
		DialTimeout: time.Second * 30,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	conn := sqlx.NewDb(db, "clickhouse")
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := conn.PingContext(pingCtx); err != nil {
		slog.Error("failed to ping database", "err", err)
		return nil, err
	}

	geodatabase, err := geoip2.Open("GeoLite2-City.mmdb")
	if err != nil {
		return nil, err
	}

	a := &Analytics{
		db:           conn,
		clicksBuffer: make(chan types.ClickData, 1000),
		geo:          geodatabase,
	}

	if err := a.runMigrations(); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *Analytics) Start(ctx context.Context) {
	a.startOnce.Do(func() {
		workerCtx, cancel := context.WithCancel(ctx)
		a.workerCancel = cancel
		a.workerWG.Add(1)
		go a.worker(workerCtx)
	})
}

func (a *Analytics) runMigrations() error {
	d, err := iofs.New(migrationsClickHouseFS, "migrations/clickhouse")
	if err != nil {
		return err
	}

	driver, err := clickmigrations.WithInstance(a.db.DB, &clickmigrations.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs", d,
		"clickhouse", driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	slog.Info("ClickHouse migrations applied successfully")
	return nil
}

func (a *Analytics) worker(ctx context.Context) {
	defer a.workerWG.Done()

	var buffer []types.ClickData
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if len(buffer) > 0 {
				flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := a.recordClicks(flushCtx, buffer); err != nil {
					slog.Warn("RecordClicks flush on shutdown failed", "error", err, "count", len(buffer))
				}
				cancel()
			}
			return
		case data := <-a.clicksBuffer:
			buffer = append(buffer, data)
			if len(buffer) >= 100 {
				flushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				err := a.recordClicks(flushCtx, buffer)
				cancel()
				if err != nil {
					slog.Warn("RecordClicks error", "error", err)
					continue
				}
				buffer = nil
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				flushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				err := a.recordClicks(flushCtx, buffer)
				cancel()
				if err != nil {
					slog.Warn("RecordClicks error", "error", err)
					continue
				}
				buffer = nil
			}
		}
	}
}

func (a *Analytics) Close() error {
	var closeErr error

	a.closeOnce.Do(func() {
		if a.workerCancel != nil {
			a.workerCancel()
		}
		a.workerWG.Wait()

		if a.geo != nil {
			if err := a.geo.Close(); err != nil {
				closeErr = err
				return
			}
		}
		closeErr = a.db.Close()
	})

	return closeErr
}

func (a *Analytics) recordClicks(ctx context.Context, clicks []types.ClickData) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO clicks (user_id, short_code, country, city, user_agent, referer) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, data := range clicks {
		city := "Unknown"
		country := "Unknown"
		ip := net.ParseIP(data.IP)
		if ip != nil {
			record, err := a.geo.City(ip)
			if err == nil {
				if name, ok := record.City.Names["en"]; ok {
					city = name
				}
				if name, ok := record.Country.Names["en"]; ok {
					country = name
				}
			}
		}
		_, err = stmt.ExecContext(ctx, data.UserId, data.ShortCode, country, city, data.UserAgent, data.Referer)
		if err != nil {
			slog.Error("Failed to exec insert for click", "error", err, "ip", data.IP)
			continue
		}
	}
	return tx.Commit()
}

func (a *Analytics) PushClick(data types.ClickData) {
	if a.workerCancel == nil {
		slog.Warn("Analytics worker is not started, dropping click data", "link", data.ShortCode)
		return
	}

	select {
	case a.clicksBuffer <- data:
	default:
		slog.Warn("Analytics buffer full, dropping click data", "link", data.ShortCode)
	}
}

func (a *Analytics) GetAllAnalytic(ctx context.Context, userId int64) ([]types.Analytic, error) {
	var clicks []types.Analytic
	query := `SELECT * FROM clicks WHERE user_id = $1`

	err := a.db.SelectContext(ctx, &clicks, query, userId)
	if err != nil {
		return nil, err
	}

	return clicks, nil
}

func (a *Analytics) GetAnalyticByCode(ctx context.Context, code string) ([]types.Analytic, error) {
	var clicks []types.Analytic
	query := `SELECT * FROM clicks WHERE short_code = $1`
	err := a.db.SelectContext(ctx, &clicks, query, code)

	if err != nil {
		return nil, err
	}

	return clicks, nil
}
