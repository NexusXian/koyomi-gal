package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	searchCacheKeyPrefix = "classification:search:"
	pageCacheKeyPrefix   = "classification:page:"
	searchCacheTTL       = 24 * time.Hour
	pageCacheTTL         = 12 * time.Hour
)

// Cache wraps Redis for agent tool caching. Nil-safe: when no Redis client is
// configured the agent still works without caching.
type Cache struct {
	client *redis.Client
}

func NewCache(client *redis.Client) *Cache {
	return &Cache{client: client}
}

func (c *Cache) getJSON(ctx context.Context, key string, target any) bool {
	if c == nil || c.client == nil {
		return false
	}
	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return false
	}
	return true
}

func (c *Cache) setJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	if c == nil || c.client == nil {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.client.Set(ctx, key, raw, ttl).Err()
}
