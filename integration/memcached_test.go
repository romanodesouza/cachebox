// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/romanodesouza/cachebox"
	"github.com/romanodesouza/cachebox/storage/memcached"
)

func TestGoMemcache(t *testing.T) {
	client := memcache.New(os.Getenv("MEMCACHED_HOST"))
	if err := client.DeleteAll(); err != nil {
		t.Fatalf("could not clean up memcached %v", err)
	}

	store := memcached.NewGoMemcache(client)
	run(t, store)
}

func BenchmarkGoMemcache(b *testing.B) {
	client := memcache.New(os.Getenv("MEMCACHED_HOST"))
	store := memcached.NewGoMemcache(client)
	cache := cachebox.NewCache(store)
	ctx := context.Background()

	key := "key"
	ok := []byte("ok")

	b.ResetTimer()

	b.Run("gomemcache:get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = client.Get(key)
		}
	})

	b.Run("cachebox:get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = cache.Get(ctx, key)
		}
	})

	b.Run("gomemcache:set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = client.Set(&memcache.Item{
				Key:        key,
				Value:      ok,
				Expiration: 60,
			})
		}
	})

	b.Run("cachebox:set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cache.Set(ctx, cachebox.Item{
				Key:   key,
				Value: ok,
				TTL:   time.Minute,
			})
		}
	})

	b.Run("gomemcache:getmulti", func(b *testing.B) {
		var keys []string
		for i := 0; i < b.N; i++ {
			keys = append(keys, fmt.Sprintf("key_%d", i))
		}
		_, _ = client.GetMulti(keys)
	})

	b.Run("cachebox:getmulti", func(b *testing.B) {
		var keys []string
		for i := 0; i < b.N; i++ {
			keys = append(keys, fmt.Sprintf("key_%d", i))
		}
		_, _ = cache.GetMulti(ctx, keys)
	})

	b.Run("gomemcache:setmulti", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = client.Set(&memcache.Item{
				Key:        fmt.Sprintf("key_%d", i),
				Value:      ok,
				Expiration: 60,
			})
		}
	})

	b.Run("cachebox:setmulti", func(b *testing.B) {
		var items []cachebox.Item
		for i := 0; i < b.N; i++ {
			items = append(items, cachebox.Item{
				Key:   fmt.Sprintf("key_%d", i),
				Value: ok,
				TTL:   time.Minute,
			})
		}
		_ = cache.SetMulti(ctx, items)
	})

	b.Run("gomemcache:delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = client.Delete(key)
		}
	})

	b.Run("cachebox:delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cache.Delete(ctx, key)
		}
	})

	b.Run("gomemcache:deletemulti", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = client.Delete(fmt.Sprintf("key_%d", i))
		}
	})

	b.Run("cachebox:deletemulti", func(b *testing.B) {
		var keys []string
		for i := 0; i < b.N; i++ {
			keys = append(keys, fmt.Sprintf("key_%d", i))
		}
		_ = cache.DeleteMulti(ctx, keys)
	})
}
