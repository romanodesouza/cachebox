// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// nolint: lll
//go:generate mockgen -destination ../mock/mock_storage/mock_storage.go github.com/romanodesouza/cachebox/storage Storage

package storage

import (
	"context"
	"time"
)

// Storage is the interface that defines the cache operations on a single or many keys.
type Storage interface {
	Get(ctx context.Context, key string) ([]byte, error)
	MGet(ctx context.Context, keys []string) ([][]byte, error)
	Set(ctx context.Context, items ...Item) error
	Delete(ctx context.Context, keys ...string) error
}

// Item represents an item to get inserted in the cache storage.
type Item struct {
	Key   string
	Value []byte
	TTL   time.Duration
}
