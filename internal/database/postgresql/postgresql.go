package postgresql

import (
	"context"
	"embed"
	"errors"
	"linkshortener/internal/types"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsPostgreSQLFS embed.FS

type PostgreSQL struct {
	db *sqlx.DB
}

func Connect(ctx context.Context, url string) (*PostgreSQL, error) {
	db, err := sqlx.ConnectContext(ctx, "postgres", url)
	if err != nil {
		return nil, err
	}

	pg := &PostgreSQL{db: db}

	if err := pg.runMigrations(); err != nil {
		return nil, err
	}

	return pg, nil
}

func (db *PostgreSQL) runMigrations() error {
	d, err := iofs.New(migrationsPostgreSQLFS, "migrations/postgresql")
	if err != nil {
		return err
	}

	driver, err := postgres.WithInstance(db.db.DB, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs", d,
		"postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	slog.Info("Database migrations applied successfully")
	return nil
}

func (db *PostgreSQL) Close() error {
	return db.db.Close()
}

func (db *PostgreSQL) CreateUser(ctx context.Context, telegramID int64) error {
	query := `
		INSERT INTO users (telegram_id) VALUES ($1)
		ON CONFLICT (telegram_id) DO UPDATE SET telegram_id = EXCLUDED.telegram_id;`
	_, err := db.db.ExecContext(ctx, query, telegramID)
	return err
}

func (db *PostgreSQL) GetUserIDByTelegramID(ctx context.Context, telegramID int64) (int64, error) {
	var id int64
	err := db.db.GetContext(ctx, &id, "SELECT id FROM users WHERE telegram_id = $1", telegramID)
	return id, err
}

func (db *PostgreSQL) GetAllLinksByUser(ctx context.Context, userId int64) ([]types.LinkData, error) {
	query := `SELECT * FROM links WHERE user_id = $1`
	var links []types.LinkData
	err := db.db.SelectContext(ctx, &links, query, userId)
	return links, err
}
func (db *PostgreSQL) DeleteAllLinksByUser(ctx context.Context, userId int64) error {
	query := `DELETE FROM links WHERE user_id = $1`
	_, err := db.db.ExecContext(ctx, query, userId)
	return err
}

func (db *PostgreSQL) CreateLink(ctx context.Context, userID int64, originalLink string) (int64, error) {
	var id int64
	query := `INSERT INTO links (user_id, original_link, short_code) VALUES ($1, $2, '') RETURNING id`
	err := db.db.QueryRowContext(ctx, query, userID, originalLink).Scan(&id)
	return id, err
}

func (db *PostgreSQL) SetShortCode(ctx context.Context, id int64, shortCode string) error {
	query := `UPDATE links SET short_code = $1 WHERE id = $2`
	_, err := db.db.ExecContext(ctx, query, shortCode, id)
	return err
}

func (db *PostgreSQL) UpdateLink(ctx context.Context, userId int64, shortCode, newLink string) error {
	query := `UPDATE links SET original_link = $1, updated_at = CURRENT_TIMESTAMP WHERE user_id = $2 AND short_code = $3`
	_, err := db.db.ExecContext(ctx, query, newLink, userId, shortCode)
	return err
}

func (db *PostgreSQL) DeleteLinkByCode(ctx context.Context, userId int64, shortCode string) error {
	query := `DELETE FROM links WHERE user_id = $1 AND short_code = $2`
	_, err := db.db.ExecContext(ctx, query, userId, shortCode)
	return err
}

func (db *PostgreSQL) DeleteLinkById(ctx context.Context, userId, linkId int64) error {
	query := `DELETE FROM links WHERE user_id = $1 AND id = $2`
	_, err := db.db.ExecContext(ctx, query, userId, linkId)
	return err
}

func (db *PostgreSQL) GetLink(ctx context.Context, shortCode string) (*types.LinkCache, error) {
	query := `SELECT original_link, user_id FROM links WHERE short_code = $1`
	var linkCache types.LinkCache
	err := db.db.GetContext(ctx, &linkCache, query, shortCode)
	if err != nil {
		return nil, err
	}
	return &linkCache, err
}
