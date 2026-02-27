package cache

import (
	"context"
	"linkshortener/internal/types"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *redis.Client
}

func ConnectRedis(url, password string) (*Cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     url,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Cache{rdb: rdb}, nil
}

func (c *Cache) Get(ctx context.Context, shortLine string) (string, error) {
	return c.rdb.Get(ctx, shortLine).Result()
}

func (c *Cache) Set(ctx context.Context, linkPair types.LinkPair, expiration time.Duration) error {
	return c.rdb.Set(ctx, linkPair.ShortLink, linkPair.OriginalLink, expiration).Err()
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

func (c *Cache) Close() error {
	return c.rdb.Close()
}
