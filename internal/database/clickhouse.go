package database

import (
	"database/sql"
	"embed"
	"log/slog"
	"net"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	clickmigrations "github.com/golang-migrate/migrate/v4/database/clickhouse"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/oschwald/geoip2-golang"
)

//go:embed migrations/clickhouse/*.sql
var migrationsClickHouseFS embed.FS

type Analytics struct {
	db           *sql.DB
	clicksBuffer chan ClickData
	geo          *geoip2.Reader
}

type ClickData struct {
	UserId    int64  `json:"user_id"`
	ShortLink string `json:"short_link"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Referer   string `json:"referer"`
}

func ConnectClickHouse(addr, user, pass, dbName string) (*Analytics, error) {
	conn := clickhouse.OpenDB(&clickhouse.Options{
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
	if err := conn.Ping(); err != nil {
		return nil, err
	}

	geodatabase, err := geoip2.Open("GeoLite2-City.mmdb")
	if err != nil {
		return nil, err
	}

	a := &Analytics{
		db:           conn,
		clicksBuffer: make(chan ClickData, 1000),
		geo:          geodatabase,
	}

	if err := a.runMigrations(); err != nil {
		return nil, err
	}

	go a.worker()
	return a, nil
}

func (a *Analytics) runMigrations() error {
	d, err := iofs.New(migrationsClickHouseFS, "migrations/clickhouse")
	if err != nil {
		return err
	}

	driver, err := clickmigrations.WithInstance(a.db, &clickmigrations.Config{})
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

func (a *Analytics) worker() {
	var buffer []ClickData
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case data := <-a.clicksBuffer:
			buffer = append(buffer, data)
			if len(buffer) >= 100 {
				err := a.recordClicks(buffer)
				if err != nil {
					slog.Warn("RecordClicks error", "error", err)
				}
				buffer = nil
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				err := a.recordClicks(buffer)
				if err != nil {
					slog.Warn("RecordClicks error", "error", err)
				}
				buffer = nil
			}
		}
	}
}

func (a *Analytics) Close() error {
	if a.geo != nil {
		err := a.geo.Close()
		if err != nil {
			return err
		}
	}
	return a.db.Close()
}

func (a *Analytics) recordClicks(clicks []ClickData) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO clicks (user_id, short_link, country, city, user_agent, referer) VALUES (?, ?, ?, ?, ?, ?)")
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
		_, err = stmt.Exec(data.UserId, data.ShortLink, country, city, data.UserAgent, data.Referer)
		if err != nil {
			slog.Error("failed to exec insert for click", "error", err, "ip", data.IP)
			continue
		}
	}
	return tx.Commit()
}

func (a *Analytics) PushClick(data ClickData) {
	select {
	case a.clicksBuffer <- data:
	default:
		slog.Warn("Analytics buffer full, dropping click data", "link", data.ShortLink)
	}
}
