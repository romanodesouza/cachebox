// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package storage

import (
	"context"
)

// Hooks represents hooks to get executed after or before storage functions.
type Hooks struct {
	AfterMGet func(ctx context.Context, key string, b []byte) ([]byte, error)
	BeforeSet func(ctx context.Context, item Item) (Item, error)
}

// HooksWrap holds a storage interface, wrapping hooks over it.
type HooksWrap struct {
	Storage

	afterMGet []func(ctx context.Context, key string, b []byte) ([]byte, error)
	beforeSet []func(ctx context.Context, item Item) (Item, error)
}

// NewHooksWrap returns a new HooksWrap instance.
func NewHooksWrap(storage Storage, hooks ...Hooks) *HooksWrap {
	var hw HooksWrap

	if h, ok := storage.(*HooksWrap); ok {
		hw = *h
	} else {
		hw.Storage = storage
	}

	for _, h := range hooks {
		if h.AfterMGet != nil {
			hw.afterMGet = append(hw.afterMGet, h.AfterMGet)
		}

		if h.BeforeSet != nil {
			hw.beforeSet = append(hw.beforeSet, h.BeforeSet)
		}
	}

	return &hw
}

// MGet performs a get multi call in the storage with hooks assigned.
func (h *HooksWrap) MGet(ctx context.Context, keys ...string) ([][]byte, error) {
	bb, err := h.Storage.MGet(ctx, keys...)
	if err != nil {
		return nil, err
	}

	if len(h.afterMGet) > 0 {
		for i := range bb {
			for _, hook := range h.afterMGet {
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
func (h *HooksWrap) Set(ctx context.Context, items ...Item) error {
	if len(h.beforeSet) > 0 {
		for i := range items {
			for _, hook := range h.beforeSet {
				var err error
				items[i], err = hook(ctx, items[i])

				if err != nil {
					return err
				}
			}
		}
	}

	return h.Storage.Set(ctx, items...)
}
