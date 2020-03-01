// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:generate mockgen -destination mock/mock_cachebox/mock_storage.go github.com/romanodesouza/cachebox Storage

package cachebox

import (
	"context"
	"time"
)

// Storage is the interface that defines the cache operations on a single or many keys.
type Storage interface {
	Get(ctx context.Context, key string) ([]byte, error)
	GetMulti(ctx context.Context, keys []string) ([][]byte, error)
	Set(ctx context.Context, key string, b []byte, ttl time.Duration) error
	SetMulti(ctx context.Context, kv map[string][]byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteMulti(ctx context.Context, keys []string) error
}
