// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build integration

package integration

import (
	"os"
	"testing"

	"github.com/bradfitz/gomemcache/memcache"
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
