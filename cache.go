// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox

import (
	"context"
	"time"
)

// Cache handles a cache storage.
type Cache struct {
	storage    Storage
	nsttl      time.Duration
	recyclable bool
}

// NewCache returns a new Cache instance.
func NewCache(storage Storage, opts ...func(*Cache)) *Cache {
	c := &Cache{
		storage:    storage,
		nsttl:      12 * time.Hour,
		recyclable: true,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithDefaultNamespaceTTL sets the ttl for namespace keys.
//
// Default is 12h.
func WithDefaultNamespaceTTL(ttl time.Duration) func(*Cache) {
	return func(c *Cache) { c.nsttl = ttl }
}

// WithKeyBasedExpiration enables key-based expiration based on namespace version.
//
// Given a key "cachekey" and a namespace "ns" of version "1", the versioned key would be "cachebox:v1:cachekey"
// Once the namespace gets invalidated, the next formed key could be "cachebox:v2:cachekey" and so on.
// Versioning is done using unix nano timestamps.
// By default, this behaviour is disabled in favour of the recyclable keys strategy.
func WithKeyBasedExpiration() func(*Cache) {
	return func(c *Cache) { c.recyclable = false }
}

// Get performs a get call in the cache storage.
//
// In case of recompute or bypass, it returns (nil, nil) to fake a miss and skip the call.
func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	if IsRecompute(ctx) || IsBypass(ctx) {
		return nil, nil
	}

	bb, err := c.storage.MGet(ctx, key)
	if err != nil {
		return nil, err
	}

	return bb[0], nil
}

// GetMulti performs a get multi call in the cache storage.
//
// In case of recompute or bypass, it returns (nil, nil) to fake a miss and skip the call.
func (c *Cache) GetMulti(ctx context.Context, keys []string) ([][]byte, error) {
	if IsRecompute(ctx) || IsBypass(ctx) {
		return nil, nil
	}

	bb, err := c.storage.MGet(ctx, keys...)
	if err != nil {
		return nil, err
	}

	return bb, nil
}

// Set performs a set call in the cache storage.
//
// In case of bypass, it returns nil to skip the call.
func (c *Cache) Set(ctx context.Context, item Item) error {
	if IsBypass(ctx) {
		return nil
	}

	return c.storage.Set(ctx, item)
}

// SetMulti performs a set multi call in the cache storage.
//
// In case of bypass, it returns nil to skip the call.
func (c *Cache) SetMulti(ctx context.Context, items []Item) error {
	if IsBypass(ctx) {
		return nil
	}

	return c.storage.Set(ctx, items...)
}

// Delete performs a delete call in the cache storage.
//
// In case of bypass, it returns nil to skip the call.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if IsBypass(ctx) {
		return nil
	}

	return c.storage.Delete(ctx, key)
}

// DeleteMulti performs a delete multi call in the cache storage.
//
// In case of bypass, it returns nil to skip the call.
func (c *Cache) DeleteMulti(ctx context.Context, keys []string) error {
	if IsBypass(ctx) {
		return nil
	}

	return c.storage.Delete(ctx, keys...)
}

// Namespace a new CacheNS instance to perform cache calls based on a namespace version.
func (c *Cache) Namespace(keys ...string) *CacheNS {
	return NewCacheNS(c, keys)
}

type key struct{ name string }

var recomputeKey = key{"recompute"}

// WithRecompute returns a context with recompute state.
//
// A recompute state bypasses cache reading to force updating the current cache state.
// Use this to precompute values.
func WithRecompute(ctx context.Context) context.Context {
	return context.WithValue(ctx, recomputeKey, struct{}{})
}

// IsRecompute checks whether there is a recompute state.
func IsRecompute(ctx context.Context) bool {
	_, ok := ctx.Value(recomputeKey).(struct{})
	return ok
}

var bypassKey = key{"bypass"}

// WithBypass returns a context with bypass state.
//
// A bypass state bypasses both cache reading and writing.
// Use this to skip the cache layer.
func WithBypass(ctx context.Context) context.Context {
	return context.WithValue(ctx, bypassKey, struct{}{})
}

// IsBypass checks whether there is a bypass state.
func IsBypass(ctx context.Context) bool {
	_, ok := ctx.Value(bypassKey).(struct{})
	return ok
}
