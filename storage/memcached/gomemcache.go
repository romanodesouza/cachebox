// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package memcached

import (
	"context"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/romanodesouza/cachebox"
)

var _ cachebox.Storage = (*GoMemcache)(nil)

// GoMemcache implements the cachebox.Storage interface by wrapping the gomemcache client.
type GoMemcache struct {
	client *memcache.Client
}

// NewGoMemcache returns a new GoMemcache instance.
func NewGoMemcache(client *memcache.Client) *GoMemcache {
	return &GoMemcache{client: client}
}

// MGet performs a get or multi get call.
func (g *GoMemcache) MGet(ctx context.Context, keys ...string) ([][]byte, error) {
	if len(keys) == 1 {
		item, err := g.client.Get(keys[0])

		switch {
		case err == memcache.ErrCacheMiss:
			return [][]byte{nil}, nil
		case err != nil:
			return nil, err
		}

		return [][]byte{item.Value}, nil
	}

	reply, err := g.client.GetMulti(keys)
	if err != nil {
		return nil, err
	}

	bb := make([][]byte, len(keys))

	for i, key := range keys {
		item, ok := reply[key]
		if !ok {
			bb[i] = nil
			continue
		}

		bb[i] = item.Value
	}

	return bb, nil
}

// Set performs a single or many set calls.
func (g *GoMemcache) Set(ctx context.Context, items ...cachebox.Item) error {
	for _, item := range items {
		err := g.client.Set(&memcache.Item{
			Key:        item.Key,
			Value:      item.Value,
			Expiration: int32(item.TTL / time.Second),
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// Delete performs a single or many delete calls.
func (g *GoMemcache) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		err := g.client.Delete(key)

		if err != nil {
			return err
		}
	}

	return nil
}
