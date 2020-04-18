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

	"github.com/gomodule/redigo/redis"
	"github.com/romanodesouza/cachebox"
	storageredis "github.com/romanodesouza/cachebox/storage/redis"
)

func TestRedigo(t *testing.T) {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", os.Getenv("REDIS_HOST"))
		},
	}

	conn := pool.Get()
	defer conn.Close() //nolint:errcheck

	if _, err := conn.Do("FLUSHALL"); err != nil {
		t.Fatalf("could not clean up redis %v", err)
	}

	store := storageredis.NewRedigo(pool)
	run(t, store)
}

func BenchmarkRedigo(b *testing.B) {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", os.Getenv("REDIS_HOST"))
		},
	}

	store := storageredis.NewRedigo(pool)
	cache := cachebox.NewCache(store)
	ctx := context.Background()

	key := "key"
	ok := []byte("ok")

	b.ResetTimer()

	b.Run("redigo:get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			conn, _ := pool.GetContext(ctx)
			_, _ = redis.Bytes(conn.Do("GET", key))
			conn.Close()
		}
	})

	b.Run("cachebox:get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = cache.Get(ctx, key)
		}
	})

	b.Run("redigo:set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			conn, _ := pool.GetContext(ctx)
			_, _ = redis.Bytes(conn.Do("SETEX", key, int32(60), ok))
			conn.Close()
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

	b.Run("redigo:getmulti", func(b *testing.B) {
		var keys []interface{}
		for i := 0; i < b.N; i++ {
			keys = append(keys, fmt.Sprintf("key_%d", i))
		}
		conn, _ := pool.GetContext(ctx)
		_, _ = redis.ByteSlices(conn.Do("MGET", keys...))
		conn.Close()
	})

	b.Run("cachebox:getmulti", func(b *testing.B) {
		var keys []string
		for i := 0; i < b.N; i++ {
			keys = append(keys, fmt.Sprintf("key_%d", i))
		}
		_, _ = cache.GetMulti(ctx, keys)
	})

	b.Run("redigo:setmulti", func(b *testing.B) {
		conn, _ := pool.GetContext(ctx)
		for i := 0; i < b.N; i++ {
			_ = conn.Send("SETEX", key, int32(60), ok)
		}
		_ = conn.Flush()
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

	b.Run("redigo:delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			conn, _ := pool.GetContext(ctx)
			_, _ = conn.Do("DEL", key)
			conn.Close()
		}
	})

	b.Run("cachebox:delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cache.Delete(ctx, key)
		}
	})

	b.Run("redigo:deletemulti", func(b *testing.B) {
		var keys []interface{}
		for i := 0; i < b.N; i++ {
			keys = append(keys, fmt.Sprintf("key_%d", i))
		}
		conn, _ := pool.GetContext(ctx)
		_, _ = conn.Do("DEL", keys...)
		conn.Close()
	})

	b.Run("cachebox:deletemulti", func(b *testing.B) {
		var keys []string
		for i := 0; i < b.N; i++ {
			keys = append(keys, fmt.Sprintf("key_%d", i))
		}
		_ = cache.DeleteMulti(ctx, keys)
	})
}
