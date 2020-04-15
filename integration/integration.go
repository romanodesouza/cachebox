// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/romanodesouza/cachebox"
)

func run(t *testing.T, store cachebox.Storage) {
	cache := cachebox.NewCache(store)
	ctx := context.Background()

	t.Run("it should get miss in the first fresh get call", func(t *testing.T) {
		b, err := cache.Get(ctx, "key")

		if fmt.Sprintf("%v", err) != "<nil>" {
			t.Errorf("got %v; want <nil>", err)
		}

		if !bytes.Equal(b, nil) {
			t.Errorf("got %v; want <nil>", b)
		}
	})

	t.Run("it should set an item", func(t *testing.T) {
		err := cache.Set(ctx, cachebox.Item{
			Key:   "key",
			Value: []byte("ok"),
			TTL:   time.Minute,
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("it should retrieve an item", func(t *testing.T) {
		b, err := cache.Get(ctx, "key")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if want := []byte("ok"); !bytes.Equal(b, want) {
			t.Errorf("got %v; want %v", b, want)
		}
	})

	t.Run("it should set many items", func(t *testing.T) {
		err := cache.SetMulti(ctx, []cachebox.Item{
			{
				Key:   "key1",
				Value: []byte("ok1"),
				TTL:   time.Minute,
			},
			{
				Key:   "key2",
				Value: []byte("ok2"),
				TTL:   time.Minute,
			},
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("it should retrieve many items", func(t *testing.T) {
		bb, err := cache.GetMulti(ctx, []string{"key1", "key2"})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(bb) != 2 {
			t.Errorf("unexpected number of items: %d", len(bb))
		}

		if want := []byte("ok1"); !bytes.Equal(bb[0], want) {
			t.Errorf("got %v; want %v", bb[0], want)
		}

		if want := []byte("ok2"); !bytes.Equal(bb[1], want) {
			t.Errorf("got %v; want %v", bb[1], want)
		}
	})

	t.Run("it should delete many items", func(t *testing.T) {
		err := cache.DeleteMulti(ctx, []string{"key1", "key2"})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		bb, err := cache.GetMulti(ctx, []string{"key1", "key2"})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if want := []byte(nil); !bytes.Equal(bb[0], want) {
			t.Errorf("got %v; want %v", bb[0], want)
		}

		if want := []byte(nil); !bytes.Equal(bb[1], want) {
			t.Errorf("got %v; want %v", bb[1], want)
		}
	})

	t.Run("it should invalidate namespaced items", func(t *testing.T) {
		testNamespace(t, cache)
	})

	t.Run("it should invalidate namespaced items on key-based expiration strategy", func(t *testing.T) {
		testNamespace(t, cachebox.NewCache(store, cachebox.WithKeyBasedExpiration()))
	})
}

func testNamespace(t *testing.T, cache *cachebox.Cache) {
	var b []byte
	var err error

	ctx := context.Background()
	ns := cache.Namespace("nskey1", "nskey2")

	err = ns.Set(ctx, cachebox.Item{
		Key:   "key1",
		Value: []byte("ok1"),
		TTL:   time.Minute,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = ns.Set(ctx, cachebox.Item{
		Key:   "key2",
		Value: []byte("ok2"),
		TTL:   time.Minute,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	ns2 := cache.Namespace("nskey2")
	err = ns2.Set(ctx, cachebox.Item{
		Key:   "key3",
		Value: []byte("ok3"),
		TTL:   time.Minute,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Invalidate nskey1
	err = cache.Delete(ctx, "nskey1")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Next namespaced call should get this items invalidated
	ns = cache.Namespace("nskey1", "nskey2")

	for _, key := range []string{"key1", "key2"} {
		b, err = ns.Get(ctx, key)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if b != nil {
			t.Errorf("got %v; want <nil>", b)
		}
	}

	// nskey2 items should remain valid
	ns = cache.Namespace("nskey2")
	b, err = ns.Get(ctx, "key3")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if want := []byte("ok3"); !bytes.Equal(b, want) {
		t.Errorf("got %v; want %v", b, want)
	}
}
