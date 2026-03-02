package service

import (
	"context"
	"database/sql"
	"errors"
	"linkshortener/internal/cache"
	"linkshortener/internal/database"
	"linkshortener/internal/types"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const alphabet = "0123456789qwertyuiopasdfghjklzxcvbnmMNBVCXZLKJHGFDQASWERTYUIOP"

var ErrInvalidCharacter = errors.New("invalid character")
var ErrCodeIsBusy = errors.New("code is busy")

type Shortener struct {
	database *database.Database
	cache    *cache.Cache
}

func NewShortener(database *database.Database, cache *cache.Cache) *Shortener {
	return &Shortener{database: database, cache: cache}
}

func (s *Shortener) CreateNewShortLink(ctx context.Context, originalLink string, userId int64) (string, error) {
	linkId, err := s.database.CreateLink(ctx, userId, originalLink)
	if err != nil {
		return "", err
	}
	shortCode := s.base62Encode(linkId)

	for {
		hasLink, err := s.database.GetLink(ctx, shortCode)
		if hasLink == nil && errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			return "", err
		}
		err = s.database.DeleteLinkById(ctx, userId, linkId)
		if err != nil {
			return "", err
		}
		linkId, err = s.database.CreateLink(ctx, userId, originalLink)
		if err != nil {
			return "", err
		}
		shortCode = s.base62Encode(linkId)
	}

	if err := s.database.SetShortCode(ctx, linkId, shortCode); err != nil {
		return "", err
	}
	if err := s.cache.Set(ctx, shortCode, &types.LinkCache{OriginalLink: originalLink, UserID: userId}, 10*time.Minute); err != nil {
		return "", err
	}
	return shortCode, nil
}

func (s *Shortener) CreateNewCustomShortLink(ctx context.Context, originalLink, shortCode string, userId int64) error {
	hasCode, err := s.database.GetLink(ctx, shortCode)
	if hasCode != nil && err == nil {
		return ErrCodeIsBusy
	}

	linkId, err := s.database.CreateLink(ctx, userId, originalLink)
	if err != nil {
		return err
	}

	if err := s.database.SetShortCode(ctx, linkId, shortCode); err != nil {
		return err
	}
	if err := s.cache.Set(ctx, shortCode, &types.LinkCache{OriginalLink: originalLink, UserID: userId}, 10*time.Minute); err != nil {
		return err
	}

	return nil
}

func (s *Shortener) DeleteLink(ctx context.Context, userId int64, shortCode string) error {
	if err := s.database.DeleteLinkByCode(ctx, userId, shortCode); err != nil {
		return err
	}
	if err := s.cache.Delete(ctx, shortCode); err != nil {
		return err
	}
	return nil
}

func (s *Shortener) UpdateLink(ctx context.Context, userId int64, shortCode, newLink string) error {
	if err := s.database.UpdateLink(ctx, userId, shortCode, newLink); err != nil {
		return err
	}
	return s.cache.Update(ctx, shortCode, &types.LinkCache{OriginalLink: newLink, UserID: userId}, 10*time.Minute)
}

func (s *Shortener) GetLinkCacheByCode(ctx context.Context, shortCode string) (*types.LinkCache, error) {
	var err error
	var linkCache *types.LinkCache
	linkCache, err = s.cache.Get(ctx, shortCode)

	if err == nil {
		return linkCache, nil
	}

	if !errors.Is(err, redis.Nil) {
		slog.Warn("Redis error", "error", err)
	}

	linkCache, err = s.database.GetLink(ctx, shortCode)

	if err != nil {
		slog.Error("Database error", "error", err)
		return nil, err
	}

	if err = s.cache.Set(ctx, shortCode, linkCache, 10*time.Minute); err != nil {
		slog.Warn("Failed to warm up cache", "error", err)
		return linkCache, err
	}

	return linkCache, nil
}

func (s *Shortener) base62Encode(linkId int64) string {
	if linkId == 0 {
		return string(alphabet[0])
	}

	res := make([]byte, 0, 12)

	for linkId > 0 {
		res = append(res, alphabet[linkId%62])
		linkId /= 62
	}
	slices.Reverse(res)
	return string(res)
}

func (s *Shortener) base62Decode(shortCode string) (int64, error) {
	var res int64

	for _, char := range shortCode {
		index := strings.IndexRune(alphabet, char)

		if index == -1 {
			return 0, ErrInvalidCharacter
		}

		res = res*62 + int64(index)
	}

	return res, nil
}

func (s *Shortener) IsValidShortCode(code string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-]{1,20}$`, code)
	return matched
}
