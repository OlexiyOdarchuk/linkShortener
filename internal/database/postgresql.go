package database

import (
	"embed"
	"errors"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//type IDatabase interface {
//	CreateUser(userId string) error
//	GetAllLinksByUser(userId string) ([]types.LinkPair, error)
//	DeleteAllLinksByUser(userId string) error
//
//	CreateShortLink(userId, originalLink string) (types.LinkPair, error)
//	UpdateLink(userId string, shortLink types.ShortLink, oldLink, newLink types.OriginalLink) (types.LinkPair, error)
//	DeleteLink(userId string, shortLink string) error
//	GetLink(shortLink types.ShortLink) (types.LinkPair, error)
//}

type Database struct {
	db *sqlx.DB
}

func ConnectPostgres(url string) (*Database, error) {
	db, err := sqlx.Connect("postgres", url)
	if err != nil {
		return nil, err
	}

	pg := &Database{db: db}

	if err := pg.RunMigrations(); err != nil {
		return nil, err
	}

	return pg, nil
}

func (db *Database) RunMigrations() error {
	d, err := iofs.New(migrationsFS, "migrations")
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
