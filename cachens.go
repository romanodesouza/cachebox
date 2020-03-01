// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/romanodesouza/cachebox/storage"
)

//nolint:golint
var Now = func() time.Time { return time.Now().UTC() }

// CacheNS handles a namespace storage.
//
// Default ttl of keys is 24h.
type CacheNS struct {
	storage storage.NamespaceStorage
	ttl     time.Duration
}

// NewCacheNS returns a new CacheNS instance.
func NewCacheNS(storage storage.NamespaceStorage, opts ...func(*CacheNS)) *CacheNS {
	cachens := &CacheNS{
		storage: storage,
		ttl:     24 * time.Hour,
	}

	for _, opt := range opts {
		opt(cachens)
	}

	return cachens
}

// WithNamespaceKeyTTL sets the ttl for namespace keys.
//
// Default is 24h.
func WithNamespaceKeyTTL(ttl time.Duration) func(*CacheNS) {
	return func(c *CacheNS) { c.ttl = ttl }
}

// GetMostRecentTimestamp returns the most recent timestamp amongst the given keys.
//
// To avoid clashing on invalidations at the same time, it uses nano precision.
// In case of bypass, it returns the current clock time and nil to skip the call.
func (c *CacheNS) GetMostRecentTimestamp(ctx context.Context, keys ...string) (int64, error) {
	// Cached now to avoid unnecessary syscalls
	var now time.Time

	getNow := func() time.Time {
		if now.IsZero() {
			now = Now()
		}

		return now
	}

	if IsBypass(ctx) {
		return getNow().UnixNano(), nil
	}

	bb, err := c.storage.MGet(ctx, keys...)
	if err != nil {
		return getNow().UnixNano(), err
	}

	var (
		mostRecentTimestamp int64 = 0
		setItems            []storage.Item
	)

	for i, b := range bb {
		var timestamp int64

		if b == nil {
			timestamp = getNow().UnixNano()

			setItems = append(setItems, storage.Item{
				Key:   keys[i],
				Value: marshalInt64(timestamp),
				TTL:   c.ttl,
			})
		} else {
			timestamp = unmarshalInt64(b)
		}

		if timestamp > mostRecentTimestamp {
			mostRecentTimestamp = timestamp
		}
	}

	if len(setItems) > 0 {
		if err := c.storage.Set(ctx, setItems...); err != nil {
			return getNow().UnixNano(), err
		}
	}

	return mostRecentTimestamp, nil
}

// Delete performs a delete call in the cache storage.
//
// In case of bypass, it returns nil to skip the call.
func (c *CacheNS) Delete(ctx context.Context, key string) error {
	if IsBypass(ctx) {
		return nil
	}

	return c.storage.Delete(ctx, key)
}

// DeleteMulti performs a delete multi call in the cache storage.
//
// In case of bypass, it returns nil to skip the call.
func (c *CacheNS) DeleteMulti(ctx context.Context, keys []string) error {
	if IsBypass(ctx) {
		return nil
	}

	return c.storage.Delete(ctx, keys...)
}

func marshalInt64(i int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(i))

	return b
}

func unmarshalInt64(b []byte) int64 {
	return int64(binary.LittleEndian.Uint64(b))
}
