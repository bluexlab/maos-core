package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/dgraph-io/ristretto"
	"golang.org/x/sync/singleflight"
)

type ApiTokenCache struct {
	cache   *ristretto.Cache
	group   singleflight.Group
	fetcher TokenFetcher
	ttl     time.Duration
}

// NewApiTokenCache returns a token micro-caching implementation that:
//
// Briefly caches API tokens to:
// 1. Prevent request bursts to the token service
// 2. Mitigate the thundering herd problem
// 3. Improve response times for repeated requests
//
// Short cache duration balances performance gains with token invalidation needs.
// Non-existent tokens are also briefly cached (with empty content) to prevent
// database stampedes from misconfiguration or malicious attacks.
func NewApiTokenCache(fetcher TokenFetcher, ttl time.Duration) *ApiTokenCache {
	return &ApiTokenCache{
		cache:   createCache(),
		group:   singleflight.Group{},
		fetcher: fetcher,
		ttl:     ttl,
	}
}

func (c *ApiTokenCache) GetToken(ctx context.Context, apiToken string) *Token {
	value, found := c.cache.Get(apiToken)
	if found {
		if value == nil {
			return nil
		}
		return value.(*Token)
	}

	// singleflight to prevent thundering herd problem
	fetched, err, _ := c.group.Do(apiToken, func() (interface{}, error) {
		token, err := c.fetcher(ctx, apiToken)
		if err != nil {
			return nil, err
		}
		if token == nil {
			// Non-existent tokens are also briefly cached (with empty content)
			slog.Warn("api token not found", "token", apiToken)
			c.cache.SetWithTTL(apiToken, nil, 1, c.ttl)
			return nil, nil
		}

		c.cache.SetWithTTL(apiToken, token, 1, c.ttl)
		return token, nil
	})

	if err != nil {
		slog.Error("cannot fetching api token", "token", apiToken, "error", err)
		return nil
	}
	if fetched == nil {
		return nil
	}
	return fetched.(*Token)
}

func (c *ApiTokenCache) Wait() {
	c.cache.Wait()
}

func createCache() *ristretto.Cache {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e4,     // number of keys to track frequency of (10000).
		MaxCost:     1 << 20, // maximum cost of cache (1M).
		BufferItems: 8,       // number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}
	return cache
}
