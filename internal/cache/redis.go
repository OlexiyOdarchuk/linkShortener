package cache

import (
	"context"
	"encoding/json"
	"linkshortener/internal/types"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *redis.Client
}

const codePrefix = "code:"

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

func (c *Cache) Get(ctx context.Context, shortCode string) (*types.LinkCache, error) {
	val, err := c.rdb.Get(ctx, codePrefix+shortCode).Result()

	if err != nil {
		return nil, err
	}

	var data *types.LinkCache
	err = json.Unmarshal([]byte(val), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Cache) Set(ctx context.Context, shortCode string, cache *types.LinkCache, expiration time.Duration) error {
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, codePrefix+shortCode, data, expiration).Err()
}

func (c *Cache) Delete(ctx context.Context, shortCode string) error {
	return c.rdb.Del(ctx, codePrefix+shortCode).Err()
}

func (c *Cache) Close() error {
	return c.rdb.Close()
}
