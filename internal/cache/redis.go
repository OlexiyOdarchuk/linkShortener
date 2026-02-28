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

type LinkCache struct {
	OriginalURL string `json:"url"`
	UserID      int64  `json:"uid"`
}

const linkPrefix = "link:"

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

func (c *Cache) Get(ctx context.Context, shortLink string) (LinkCache, error) {
	val, err := c.rdb.Get(ctx, linkPrefix+shortLink).Result()
	if err == redis.Nil {
		return LinkCache{}, err
	}
	if err != nil {
		return LinkCache{}, err
	}
	var data LinkCache
	err = json.Unmarshal([]byte(val), &data)
	if err != nil {
		return LinkCache{}, err
	}
	return data, nil
}

func (c *Cache) Set(ctx context.Context, linkPair types.LinkPair, userId int64, expiration time.Duration) error {
	data, err := json.Marshal(&LinkCache{linkPair.OriginalLink, userId})
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, linkPrefix+linkPair.ShortLink, data, expiration).Err()
}

func (c *Cache) Delete(ctx context.Context, shortLink string) error {
	return c.rdb.Del(ctx, linkPrefix+shortLink).Err()
}

func (c *Cache) Close() error {
	return c.rdb.Close()
}
