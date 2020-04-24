// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"
)

var now = func() time.Time { return time.Now().UTC() }

// CacheNS handles namespaced cache calls.
type CacheNS struct {
	cache     *Cache
	nskeys    []string
	nsversion int64
}

// NewCacheNS returns a new CacheNS instance.
func NewCacheNS(c *Cache, nskeys []string) *CacheNS {
	return &CacheNS{
		cache:  c,
		nskeys: nskeys,
	}
}

// Get performs a get call in the cache storage, checking the namespace version.
//
// On recyclable strategy, compares the namespace version with the given key to confirm a cache hit or miss.
//
// On key-based strategy, prefixes the key with the namespace version.
func (c *CacheNS) Get(ctx context.Context, key string) ([]byte, error) {
	var b []byte

	if c.nsversion == 0 {
		keys := c.nskeys

		// Execute a single MGet on recyclable strategy
		if c.cache.recyclable {
			keys = append(keys, buildRecyclableKey(key))
		}

		bb, err := c.cache.storage.MGet(ctx, keys...)
		if err != nil {
			return nil, err
		}

		ts, err := c.mostRecentTimestamp(ctx, c.nskeys, bb)
		if err != nil {
			return nil, err
		}

		c.nsversion = ts

		// Execute an extra MGet on key-based expiration strategy
		if !c.cache.recyclable {
			bb, err = c.cache.storage.MGet(ctx, buildVersionedKey(key, c.nsversion))
			if err != nil {
				return nil, err
			}
		}

		b = bb[len(bb)-1]
	} else {
		if c.cache.recyclable {
			key = buildRecyclableKey(key)
		} else {
			key = buildVersionedKey(key, c.nsversion)
		}

		bb, err := c.cache.storage.MGet(ctx, key)
		if err != nil {
			return nil, err
		}

		b = bb[0]
	}

	if bpc := bypassFromContext(ctx); bpc == BypassReading || bpc == BypassReadWriting {
		return nil, nil
	}

	// Miss
	if b == nil {
		return nil, nil
	}

	if c.cache.recyclable {
		var version int64
		version, b = splitVersion(b)

		// Miss
		if c.nsversion > version {
			return nil, nil
		}
	}

	// Hit
	return b, nil
}

// Set performs a set call in the cache storage, handling the namespace version.
//
// On recyclable strategy, prepends 8 bytes with the encoded namespace version in the item value as its cache version.
//
// On key-based strategy, prefixes the item key with the namespace version.
func (c *CacheNS) Set(ctx context.Context, item Item) error {
	if bypassFromContext(ctx) == BypassReadWriting {
		return nil
	}

	if c.nsversion == 0 {
		bb, err := c.cache.storage.MGet(ctx, c.nskeys...)
		if err != nil {
			return err
		}

		ts, err := c.mostRecentTimestamp(ctx, c.nskeys, bb)
		if err != nil {
			return err
		}

		c.nsversion = ts
	}

	if c.cache.recyclable {
		item.Key = buildRecyclableKey(item.Key)
		item.Value = append(marshalInt64(c.nsversion), item.Value...)
	} else {
		item.Key = buildVersionedKey(item.Key, c.nsversion)
	}

	return c.cache.storage.Set(ctx, item)
}

func (c *CacheNS) mostRecentTimestamp(ctx context.Context, keys []string, bb [][]byte) (int64, error) {
	var mostRecentTimestamp int64
	var items []Item

	for i, key := range keys {
		var timestamp int64

		if bb[i] == nil {
			timestamp = now().UnixNano()

			items = append(items, Item{
				Key:   key,
				Value: marshalInt64(timestamp),
				TTL:   c.cache.nsttl,
			})
		} else {
			timestamp = unmarshalInt64(bb[i])
		}

		if timestamp > mostRecentTimestamp {
			mostRecentTimestamp = timestamp
		}
	}

	if len(items) > 0 {
		if err := c.cache.storage.Set(ctx, items...); err != nil {
			return -1, err
		}
	}

	return mostRecentTimestamp, nil
}

func buildRecyclableKey(key string) string {
	return fmt.Sprintf("cachebox:recyc:%s", key)
}

func buildVersionedKey(key string, version int64) string {
	return fmt.Sprintf("cachebox:v%d:%s", version, key)
}

func splitVersion(b []byte) (int64, []byte) {
	version := unmarshalInt64(b[:8])
	return version, b[8:]
}

func marshalInt64(i int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(i))

	return b
}

func unmarshalInt64(b []byte) int64 {
	return int64(binary.LittleEndian.Uint64(b))
}
