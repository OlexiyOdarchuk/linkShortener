package service

import (
	"context"
	"errors"
	"linkshortener/internal/cache"
	"linkshortener/internal/database"
	"linkshortener/internal/types"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const alphabet = "0123456789qwertyuiopasdfghjklzxcvbnmMNBVCXZLKJHGFDQASWERTYUIOP"

var ErrInvalidCharacter = errors.New("invalid character")

type Shortener struct {
	database *database.Database
	cache    *cache.Cache
}

func NewShortener(database *database.Database, cache *cache.Cache) *Shortener {
	return &Shortener{database: database, cache: cache}
}

func (s *Shortener) CreateNewShortLink(originalLink string, telegramID int64) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.database.CreateUser(ctx, telegramID); err != nil {
		return "", err
	}

	userId, err := s.database.GetUserIDByTelegramID(ctx, telegramID)
	if err != nil {
		return "", err
	}
	linkId, err := s.database.CreateLink(ctx, userId, originalLink)
	if err != nil {
		return "", err
	}
	code := s.base62Encode(linkId)
	if err := s.database.SetShortCode(ctx, linkId, code); err != nil {
		return "", err
	}
	if err := s.cache.Set(ctx, code, &types.LinkCache{OriginalLink: originalLink, UserID: userId}, 10*time.Minute); err != nil {
		return "", err
	}
	return code, nil
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
