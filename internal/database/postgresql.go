package database

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

//go:embed migrations/postgresql/*.sql
var migrationsPostgreSQLFS embed.FS

type Database struct {
	db *sqlx.DB
}

func ConnectPostgres(ctx context.Context, url string) (*Database, error) {
	db, err := sqlx.ConnectContext(ctx, "postgres", url)
	if err != nil {
		return nil, err
	}

	pg := &Database{db: db}

	if err := pg.runMigrations(); err != nil {
		return nil, err
	}

	return pg, nil
}

func (db *Database) runMigrations() error {
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

func (db *Database) Close() error {
	return db.db.Close()
}

func (db *Database) CreateUser(ctx context.Context, userId string) error {
	// TODO
	return nil
}
func (db *Database) GetAllLinksByUser(ctx context.Context, userId string) ([]types.LinkPair, error) {
	// TODO
	return nil, nil
}
func (db *Database) DeleteAllLinksByUser(ctx context.Context, userId string) error {
	// TODO
	return nil
}

func (db *Database) CreateShortLink(ctx context.Context, userId, originalLink string) (types.LinkPair, error) {
	// TODO
	return types.LinkPair{OriginalLink: "", ShortLink: ""}, nil
}

func (db *Database) UpdateLink(ctx context.Context, userId string, oldPair types.LinkPair, newLink string) (types.LinkPair, error) {
	// TODO
	return types.LinkPair{ShortLink: oldPair.ShortLink, OriginalLink: newLink}, nil
}

func (db *Database) DeleteLink(ctx context.Context, userId string, shortLink string) error {
	// TODO
	return nil
}

func (db *Database) GetLink(ctx context.Context, shortLink string) (string, error) {
	// TODO
	return "", nil
}
