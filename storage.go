// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//nolint: lll
//go:generate mockgen -destination mock/mock_cachebox/mock_storage.go github.com/romanodesouza/cachebox Storage

package cachebox

import (
	"context"
	"time"
)

// Storage is the interface that defines the cache operations on a single or many keys.
type Storage interface {
	MGet(ctx context.Context, keys ...string) ([][]byte, error)
	Set(ctx context.Context, items ...Item) error
	Delete(ctx context.Context, keys ...string) error
}

// Item represents an item to get inserted in the cache storage.
type Item struct {
	Key   string
	Value []byte
	TTL   time.Duration
}

// StorageHooks represents hooks to get executed after or before storage functions.
type StorageHooks struct {
	AfterMGet func(ctx context.Context, key string, b []byte) ([]byte, error)
	BeforeSet func(ctx context.Context, item Item) (Item, error)
}

// StorageWrapper holds a storage interface, wrapping hooks over it.
type StorageWrapper struct {
	Storage

	afterMGet []func(ctx context.Context, key string, b []byte) ([]byte, error)
	beforeSet []func(ctx context.Context, item Item) (Item, error)
}

// NewStorageWrapper returns a new StorageWrapper instance.
func NewStorageWrapper(storage Storage, hooks ...StorageHooks) *StorageWrapper {
	var w StorageWrapper

	if sw, ok := storage.(*StorageWrapper); ok {
		w = *sw
	} else {
		w.Storage = storage
	}

	for _, h := range hooks {
		if h.AfterMGet != nil {
			w.afterMGet = append(w.afterMGet, h.AfterMGet)
		}

		if h.BeforeSet != nil {
			w.beforeSet = append(w.beforeSet, h.BeforeSet)
		}
	}

	return &w
}

// MGet performs a get multi call in the storage with hooks assigned.
func (w *StorageWrapper) MGet(ctx context.Context, keys ...string) ([][]byte, error) {
	bb, err := w.Storage.MGet(ctx, keys...)
	if err != nil {
		return nil, err
	}

	if len(w.afterMGet) > 0 {
		for i := range bb {
			for _, hook := range w.afterMGet {
				var err error

				bb[i], err = hook(ctx, keys[i], bb[i])
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return bb, nil
}

// Set performs a set call in the cache storage.
func (w *StorageWrapper) Set(ctx context.Context, items ...Item) error {
	if len(w.beforeSet) > 0 {
		for i := range items {
			for _, hook := range w.beforeSet {
				var err error

				items[i], err = hook(ctx, items[i])
				if err != nil {
					return err
				}
			}
		}
	}

	return w.Storage.Set(ctx, items...)
}