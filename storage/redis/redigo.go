// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package redis

import (
	"context"

	"github.com/gomodule/redigo/redis"
	"github.com/romanodesouza/cachebox"
)

var _ cachebox.Storage = (*Redigo)(nil)

// Redigo implements the cachebox.Storage interface by wrapping a redigo redis Pool.
type Redigo struct {
	pool *redis.Pool
}

// NewRedigo returns a new Redigo instance.
func NewRedigo(pool *redis.Pool) *Redigo {
	return &Redigo{pool: pool}
}

// MGet performs a get or a multi get call.
func (r *Redigo) MGet(ctx context.Context, keys ...string) ([][]byte, error) {
	conn, err := r.pool.GetContext(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close() //nolint:errcheck

	if len(keys) == 1 {
		b, err := redis.Bytes(conn.Do("GET", keys[0]))

		switch {
		case err == redis.ErrNil:
			return [][]byte{nil}, nil
		case err != nil:
			return nil, err
		}

		return [][]byte{b}, nil
	}

	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	return redis.ByteSlices(conn.Do("MGET", args...))
}

// Set performs a single or many set calls.
func (r *Redigo) Set(ctx context.Context, items ...cachebox.Item) error {
	conn, err := r.pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck

	for _, item := range items {
		if err := conn.Send("SETEX", item.Key, int32(item.TTL.Seconds()), item.Value); err != nil {
			return err
		}
	}

	return conn.Flush()
}

// Delete performs a single or many delete calls.
func (r *Redigo) Delete(ctx context.Context, keys ...string) error {
	conn, err := r.pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck

	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	_, err = conn.Do("DEL", args...)

	return err
}
