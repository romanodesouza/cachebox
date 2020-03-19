// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox

import (
	"context"

	"github.com/romanodesouza/cachebox/storage"
)

// Cache handles a cache storage.
type Cache struct {
	storage storage.Storage
}

// NewCache returns a new Cache instance.
func NewCache(storage storage.Storage, opts ...func(*Cache)) *Cache {
	c := &Cache{storage: storage}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Get performs a get call in the cache storage.
//
// In case of refresh or bypass, it returns (nil, nil) to fake a miss and skip the call.
func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	if IsRefresh(ctx) || IsBypass(ctx) {
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
// In case of refresh or bypass, it returns (nil, nil) to fake a miss and skip the call.
func (c *Cache) GetMulti(ctx context.Context, keys []string) ([][]byte, error) {
	if IsRefresh(ctx) || IsBypass(ctx) {
		return nil, nil
	}

	bb, err := c.storage.MGet(ctx, keys...)
	if err != nil {
		return nil, err
	}

	return bb, nil
}

// Item represents an item to get inserted in the cache storage.
type Item storage.Item

// Set performs a set call in the cache storage.
//
// In case of bypass, it returns nil to skip the call.
func (c *Cache) Set(ctx context.Context, item Item) error {
	if IsBypass(ctx) {
		return nil
	}

	return c.storage.Set(ctx, storage.Item(item))
}

// SetMulti performs a set multi call in the cache storage.
//
// In case of bypass, it returns nil to skip the call.
func (c *Cache) SetMulti(ctx context.Context, items []Item) error {
	if IsBypass(ctx) {
		return nil
	}

	sItems := make([]storage.Item, len(items))
	for k, item := range items {
		sItems[k] = storage.Item(item)
	}

	return c.storage.Set(ctx, sItems...)
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
