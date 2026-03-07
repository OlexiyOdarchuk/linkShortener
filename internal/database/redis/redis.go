package redis

import (
	"context"
	"encoding/json"
	"errors"
	customerrs "linkshortener/internal/customErrs"
	"linkshortener/internal/types"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	rdb *redis.Client
}

const codePrefix = "code:"

func Connect(url, password string) (*Redis, error) {
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

	return &Redis{rdb: rdb}, nil
}

func (c *Redis) Get(ctx context.Context, shortCode string) (*types.LinkCache, error) {
	val, err := c.rdb.Get(ctx, codePrefix+shortCode).Result()

	if errors.Is(err, redis.Nil) {
		return nil, customerrs.ErrNoFound
	}

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

func (c *Redis) Set(ctx context.Context, shortCode string, cache *types.LinkCache, expiration time.Duration) error {
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, codePrefix+shortCode, data, expiration).Err()
}

func (c *Redis) Delete(ctx context.Context, shortCode string) error {
	return c.rdb.Del(ctx, codePrefix+shortCode).Err()
}

func (c *Redis) Update(ctx context.Context, shortCode string, cache *types.LinkCache, expiration time.Duration) error {
	_, err := c.Get(ctx, shortCode)
	if err != nil {
		return err
	}
	err = c.Delete(ctx, codePrefix+shortCode)
	if err != nil {
		return err
	}
	return c.Set(ctx, codePrefix+shortCode, cache, expiration)
}

func (c *Redis) Close() error {
	return c.rdb.Close()
}
