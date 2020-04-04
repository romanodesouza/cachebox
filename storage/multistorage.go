// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package storage

import (
	"context"

	"github.com/romanodesouza/cachebox"
)

var _ cachebox.Storage = (*MultiStorage)(nil)

// MultiStorage implements the cachebox.Storage interface by wrapping a list of storages.
type MultiStorage struct {
	storages []cachebox.Storage
}

// NewMultiStorage returns a new MultiStorage instance.
func NewMultiStorage(storages ...cachebox.Storage) *MultiStorage {
	return &MultiStorage{storages: storages}
}

// MGet performs a get multi call in the underlying cache storages.
//
// Returns early an error whether any of them fail.
func (m *MultiStorage) MGet(ctx context.Context, keys ...string) ([][]byte, error) {
	// Try to fetch everything from the first storage
	bb, err := m.storages[0].MGet(ctx, keys...)
	if err != nil {
		return nil, err
	}

	missIdx := make([]int, 0, len(keys)/2)

	for i, b := range bb {
		if b == nil {
			missIdx = append(missIdx, i)
		}
	}

	if len(missIdx) == 0 {
		return bb, nil
	}

	keymap := make(map[string]int, len(missIdx))
	miss := make([]string, len(missIdx))

	for i, idx := range missIdx {
		key := keys[idx]
		keymap[key] = idx
		miss[i] = key
	}

	for i := 1; i < len(m.storages); i++ {
		storage := m.storages[i]

		res, err := storage.MGet(ctx, miss...)
		if err != nil {
			return nil, err
		}

		newMiss := make([]string, 0, len(miss)/2)

		for i, b := range res {
			key := miss[i]

			if b == nil {
				newMiss = append(newMiss, key)
				continue
			}

			bb[keymap[key]] = b
		}

		miss = newMiss
	}

	return bb, nil
}

// Set performs a set call in all underlying cache storages.
// Returns early an error whether any of them fail.
func (m *MultiStorage) Set(ctx context.Context, items ...cachebox.Item) error {
	for _, storage := range m.storages {
		if err := storage.Set(ctx, items...); err != nil {
			return err
		}
	}

	return nil
}

// Delete performs a delete call in all underlying cache storages.
// Returns early an error whether any of them fail.
func (m *MultiStorage) Delete(ctx context.Context, keys ...string) error {
	for _, storage := range m.storages {
		if err := storage.Delete(ctx, keys...); err != nil {
			return err
		}
	}

	return nil
}
