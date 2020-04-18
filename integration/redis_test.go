// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build integration

package integration

import (
	"os"
	"testing"

	"github.com/gomodule/redigo/redis"
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
