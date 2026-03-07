package database

import (
	"context"
	"errors"
	customerrs "linkshortener/internal/customErrs"
	"linkshortener/internal/types"
	"log/slog"
	"time"
)

//go:generate mockgen -destination=mock_analytics_test.go -package=database . Analytics
//go:generate mockgen -destination=mock_cache_test.go -package=database . Cache
//go:generate mockgen -destination=mock_users_repo_test.go -package=database . UsersRepo
//go:generate mockgen -destination=mock_links_repo_test.go -package=database . LinksRepo
//go:generate mockgen -destination=mock_sql_test.go -package=database . SQL

type Analytics interface {
	Start(ctx context.Context)
	PushClick(data types.ClickData)
	GetAllAnalytic(ctx context.Context, userId int64) ([]types.Analytic, error)
	GetAnalyticByCode(ctx context.Context, code string, userId int64) ([]types.Analytic, error)
	Close() error
}

type Cache interface {
	Set(ctx context.Context, shortCode string, cache *types.LinkCache, expiration time.Duration) error
	Get(ctx context.Context, shortCode string) (*types.LinkCache, error)
	Update(ctx context.Context, shortCode string, cache *types.LinkCache, expiration time.Duration) error
	Delete(ctx context.Context, shortCode string) error
	Close() error
}

type UsersRepo interface {
	CreateUser(ctx context.Context, telegramID int64) error
	GetUserIDByTelegramID(ctx context.Context, telegramID int64) (int64, error)
}

type LinksRepo interface {
	CreateLink(ctx context.Context, userID int64, originalLink string) (int64, error)
	SetShortCode(ctx context.Context, id int64, shortCode string) error
	GetLink(ctx context.Context, shortCode string) (*types.LinkCache, error)
	GetAllLinksByUser(ctx context.Context, userId int64) ([]types.LinkData, error)
	UpdateLink(ctx context.Context, userId int64, shortCode, newLink string) error
	DeleteLinkByCode(ctx context.Context, userId int64, shortCode string) error
	DeleteLinkById(ctx context.Context, userId, linkId int64) error
	DeleteAllLinksByUser(ctx context.Context, userId int64) error
}

type SQL interface {
	UsersRepo
	LinksRepo
	Close() error
}

type Database struct {
	analytics Analytics
	cache     Cache
	sql       SQL
}

func CreateDatabase(ctx context.Context, analytics Analytics, sql SQL, cache Cache) *Database {
	analytics.Start(ctx)
	return &Database{
		analytics: analytics,
		sql:       sql,
		cache:     cache,
	}
}

func (d *Database) CreateUser(ctx context.Context, telegramID int64) error {
	return d.sql.CreateUser(ctx, telegramID)
}

func (d *Database) CreateLink(ctx context.Context, userID int64, originalLink string) (int64, error) {
	return d.sql.CreateLink(ctx, userID, originalLink)
}

func (d *Database) SetShortCode(ctx context.Context, id int64, shortCode string) error {
	return d.sql.SetShortCode(ctx, id, shortCode)
}

func (d *Database) UpdateLink(ctx context.Context, userId int64, shortCode, newLink string) error {
	if err := d.sql.UpdateLink(ctx, userId, shortCode, newLink); err != nil {
		return err
	}
	return d.cache.Update(ctx, shortCode, &types.LinkCache{OriginalLink: newLink, UserID: userId}, 10*time.Minute)
}

func (d *Database) DeleteLinkByCode(ctx context.Context, userId int64, shortCode string) error {
	if err := d.sql.DeleteLinkByCode(ctx, userId, shortCode); err != nil {
		return err
	}
	if err := d.cache.Delete(ctx, shortCode); err != nil {
		return err
	}
	return nil
}

func (d *Database) GetLinkCacheByCode(ctx context.Context, shortCode string) (*types.LinkCache, error) {
	var err error
	var linkCache *types.LinkCache
	linkCache, err = d.cache.Get(ctx, shortCode)

	if err == nil {
		return linkCache, nil
	}

	if !errors.Is(err, customerrs.ErrNoFound) {
		slog.Warn("Cache error", "error", err)
	}

	linkCache, err = d.sql.GetLink(ctx, shortCode)

	if err != nil {
		slog.Error("Database error", "error", err)
		return nil, err
	}

	if err = d.cache.Set(ctx, shortCode, linkCache, 10*time.Minute); err != nil {
		slog.Warn("Failed to warm up cache", "error", err)
		return linkCache, err
	}

	return linkCache, nil
}

func (d *Database) GetUserIDByTelegramID(ctx context.Context, telegramID int64) (int64, error) {
	return d.sql.GetUserIDByTelegramID(ctx, telegramID)
}

func (d *Database) GetAllLinksByUser(ctx context.Context, userId int64) ([]types.LinkData, error) {
	return d.sql.GetAllLinksByUser(ctx, userId)
}

func (d *Database) DeleteLinkById(ctx context.Context, userId, linkId int64) error {
	return d.sql.DeleteLinkById(ctx, userId, linkId)
}

func (d *Database) DeleteAllLinksByUser(ctx context.Context, userId int64) error {
	linksData, err := d.sql.GetAllLinksByUser(ctx, userId)
	if err != nil {
		return err
	}
	err = d.sql.DeleteAllLinksByUser(ctx, userId)
	if err != nil {
		return nil
	}
	for _, data := range linksData {
		if err = d.cache.Delete(ctx, data.ShortCode); err != nil {
			return err
		}
	}
	return nil
}

func (d *Database) Close() error {
	if err := d.analytics.Close(); err != nil {
		return err
	}
	if err := d.cache.Close(); err != nil {
		return err
	}
	if err := d.sql.Close(); err != nil {
		return err
	}
	return nil
}

func (d *Database) PushClick(data types.ClickData) {
	d.analytics.PushClick(data)
}

func (d *Database) GetAllAnalytic(ctx context.Context, userId int64) ([]types.Analytic, error) {
	return d.analytics.GetAllAnalytic(ctx, userId)
}

func (d *Database) GetAnalyticByCode(ctx context.Context, code string, userId int64) ([]types.Analytic, error) {
	return d.analytics.GetAnalyticByCode(ctx, code, userId)
}
