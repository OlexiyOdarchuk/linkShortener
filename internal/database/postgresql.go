package database

import (
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
var migrationsFS embed.FS

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

func (db *Database) CreateUser(userId string) error {
	// TODO
	return nil
}
func (db *Database) GetAllLinksByUser(userId string) ([]types.LinkPair, error) {
	// TODO
	return nil, nil
}
func (db *Database) DeleteAllLinksByUser(userId string) error {
	// TODO
	return nil
}

func (db *Database) CreateShortLink(userId, originalLink string) (types.LinkPair, error) {
	// TODO
	return types.LinkPair{OriginalLink: "", ShortLink: ""}, nil
}

func (db *Database) UpdateLink(userId string, oldPair types.LinkPair, newLink string) (types.LinkPair, error) {
	// TODO
	return types.LinkPair{ShortLink: oldPair.ShortLink, OriginalLink: newLink}, nil
}

func (db *Database) DeleteLink(userId string, shortLink string) error {
	// TODO
	return nil
}

func (db *Database) GetLink(shortLink string) (string, error) {
	// TODO
	return "", nil
}
