# cachebox

[![Godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/romanodesouza/cachebox) [![Build Status](https://travis-ci.org/romanodesouza/cachebox.svg?branch=master)](https://travis-ci.org/romanodesouza/cachebox) [![Go Report Card](https://goreportcard.com/badge/github.com/romanodesouza/cachebox?a=1)](https://goreportcard.com/report/github.com/romanodesouza/cachebox) [![codecov](https://codecov.io/gh/romanodesouza/cachebox/branch/master/graph/badge.svg)](https://codecov.io/gh/romanodesouza/cachebox) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/romanodesouza/cachebox/master/LICENSE)

A caching library to handle group and individual caches.

> There are only two hard things in Computer Science: cache invalidation and naming things.

cachebox implements [namespace versioning](https://github.com/memcached/memcached/wiki/ProgrammingTricks#namespacing) based on timestamps with nano precision over [recyclable keys](https://github.com/rails/rails/pull/29092) to make it easier to invalidate groups of keys without polluting the keyspace.

## install

```
go get github.com/romanodesouza/cachebox
```

## usage
```go
package main

import (
	"context"
	"os"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/romanodesouza/cachebox"
	"github.com/romanodesouza/cachebox/storage/memcached"
)

func main() {
	client := memcache.New(os.Getenv("MEMCACHED_HOST"))
	store := memcached.NewGoMemcache(client)
	cache := cachebox.NewCache(store)
	ctx := context.Background()

	// Get
	reply, err := cache.Get(ctx, key)
	reply, err := cache.GetMulti(ctx, keys)

	// Set
	err := cache.Set(ctx, cachebox.Item{
		Key: "key",
		Value: []byte("ok"),
		TTL: time.Hour,
	})
	err := cache.SetMulti(ctx, []cachebox.Item{
		{
			Key: "key1",
			Value: []byte("ok1"),
			TTL: time.Hour,
		},
		{
			Key: "key2",
			Value: []byte("ok2"),
			TTL: time.Hour,
		},
	})

	// Delete
	err := cache.Delete(ctx, key)
	err := cache.DeleteMulti(ctx, keys)

	// Namespacing
	ns := cache.Namespace("ns:key1", "ns:key2")
	reply, err := ns.Get(ctx, key)
	err := ns.Set(ctx, cachebox.Item{
		Key: "key",
		Value: []byte("ok"),
		TTL: time.Hour,
	})

	// Serialization
	b, err := cachebox.Marshal(i)

	// Deserialization
	err := cachebox.Unmarshal(b, &i)

	// Cache miss check
	err := cachebox.Unmarshal(b, &i)
	if err == cachebox.ErrMiss {
		// ...
	}
}
```

## storage
Built-in support for:
- [memcached](https://github.com/romanodesouza/cachebox/tree/master/storage/memcached)
- [redis](https://github.com/romanodesouza/cachebox/tree/master/storage/redis)

You can provide your own by implementing the Storage interface:
```go
type Storage interface {
	MGet(ctx context.Context, keys ...string) ([][]byte, error)
	Set(ctx context.Context, items ...Item) error
	Delete(ctx context.Context, keys ...string) error
}
```

### multi storage support
```go
store := storage.NewMultiStorage(memcached.NewGoMemcache(client), redis.NewRedigo(pool))
// Will try to fetch keys from memcached first
cache := cachebox.NewCache(store)
```

## bypass
You can bypass only reading or both read/writing.

```go
// Skip all get calls, useful to cache recomputed data
ctx := cachebox.WithBypass(parent, cachebox.BypassReading)

// Skip everything, useful to debug underlying layers
ctx := cachebox.WithBypass(parent, cachebox.BypassReadWriting)
```

## stampede prevention
Avoid a high overload when a key expires and many concurrent calls try to recompute it at the same time using i/o contention with pessimistic lock so when a key expires, only the first call recomputes it while the others await for it or until the context times out.

Read more about cache stampede on [Wikipedia](https://en.wikipedia.org/wiki/Cache_stampede).

```go
cache := cachebox.NewCache(store, cachebox.WithKeyLock())
```

## msgp compatibility
If you use [msgp](https://github.com/tinylib/msgp) to serialize/deserialize items, cachebox can reuse their interfaces.
```go
cachebox.Marshal(i) // uses msgp as long i implements its interface
cachebox.Unmarshal(b, &i) // uses msgp as long *i implements its interface
```

## gzip
Too big values? Enable gzip compression.
```go
cache := cachebox.NewCache(store, cachebox.WithGzipCompression(level))
```

## key-based versioning
Ok, cool, but I still prefer key-based versioning so I can visualize better my keyspace.

```go
cache := cachebox.NewCache(store, cachebox.WithKeyBasedExpiration())
```
Now you will be able to see namespaced keys with the `cachebox:v[timestamp]:` prefix.

## example

```go
type CacheRepository struct {
	cache *cachebox.Cache
	logger *myapp.Loggger
	repo Repository
}

func (c *CacheRepository) FindAll(ctx context.Context) ([]*Entity, error) {
	ids, err := c.FindIDs(ctx)
	if err != nil {
		return nil, err
	}

	return c.FindByIDs(ctx, ids)
}

// FindIDs finds entity ids.
//
// Group caching retrieves a key within one or more namespaces.
// If the namespaces version is newer than key's version, it's a cache miss.
func (c *CacheRepository) FindIDs(ctx context.Context) ([]int64, error) {
	nskeys := []string{"ns:users"}
	if includeInactive {
		nskeys = append(nskeys, "ns:inactiveusers")
	}

	ns := c.cache.Namespace(nskeys...)
	key := "users"

	reply, err := ns.Get(ctx, key)
	if err != nil {
		c.logger.Error(fmt.Errorf("repository/user: could not retrieve ids from cache: %w", err))
	}

	var ids []int64
	err = cachebox.Unmarshal(reply, &ids)

	// Hit
	if err == nil {
		return ids, nil
	}

	// Miss
	c.logger.Error(fmt.Errorf("repository/user: could not deserialize ids: %w", err))


	// Try to fetch ids from next repository layer
	ids, err = c.repo.FindIDs(ctx)
	if err != nil {
		return nil, err
	}

	var b []byte
	b, err = cachebox.Marshal(ids)
	if err != nil {
		c.logger.Error(fmt.Errorf("repository/user: could not serialize ids: %w", err))
		return ids, nil
	}

	err = ns.Set(ctx, cachebox.Item{
		Key: key,
		Value: b,
		TTL: time.Hour,
	})

	if err != nil {
		c.logger.Error(fmt.Errof("repository/user: could not cache ids: %w", err))
	}

	return ids, nil
}

// FindByIDs finds entities by ids.
//
// Individual caching consists in retrieving many items (from database for example) and caching
// them one by one individually, this is effective when you have a high number of shared items.
func (c *CacheRepository) FindByIDs(ctx context.Context, ids []int64) ([]*Entity, error) {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("prefix_%d", id)
	}

	reply, err := c.cache.GetMulti(ctx, keys)
	if err != nil {
		c.logger.Error(fmt.Errorf("repository/user: could not retrieve entities from cache: %w", err))
	}

	entities := make([]*Entity, len(keys))
	idx := make(map[int64]int)

	for i, b := range reply {
		err := cachebox.Unmarshal(b, entities[i])
		
		if err != nil {
			idx[ids[i]] = i
			c.logger.Error(fmt.Errorf("repository/user: could not deserialize item from cache: %w", err))
		}
	}

	// Hit
	if len(idx) == 0 {
		return entities, nil
	}

	// Miss
	missingIDs := make([]int64, 0, len(idx))
	for id := range idx {
		missingIDs = append(missingIDs, id)
	}

	// Try to fetch missing items from next repository layer
	found, err := c.repo.FindByIDs(ctx, missingIDs)
	if err != nil {
		return nil, err
	}

	items := make([]cachebox.Item, 0, len(found))

	for _, entity := range found {
		i := idx[entity.ID]

		// Place the found object in the list
		entities[i] = entity

		b, err := cachebox.Marshal(entity)
		if err != nil {
			c.logger.Error(fmt.Errorf("repository/user: could not serialize entity: %w", err))
			continue
		}

		items = append(items, cachebox.Item{
			Key: keys[i],
			Value: b,
			TTL: time.Hour,
		})
	}

	if err := c.cache.SetMulti(ctx, items); err != nil {
		c.logger.Error(fmt.Errorf("repository/user: could not cache entities: %w", err))
	}

	return entities, nil
}
```

## invalidation

```go
// Invalidate a namespace key to invalidate all related groups of keys
cache.Delete(ctx, "ns:key1")

// When invalidating an individual item, also invalidate the namespaces it belongs to
cache.DeleteMulti(ctx, "user_1", "ns:users", "ns:inactiveusers")

// You could even recompute the individual cache item before invalidating the namespaces
ctx := cachebox.WithBypass(parent, cachebox.BypassReading)
_, _ = FindByIDs(ctx, []int64{1})
cache.DeleteMulti(ctx, "ns:users", "ns:inactiveusers")
```

## benchmarks

cachebox adds almost no overhead over raw storage clients.

```
goos: linux
goarch: amd64
pkg: github.com/romanodesouza/cachebox/integration
BenchmarkGoMemcache/gomemcache:get-4         	   10000	    109752 ns/op	     208 B/op	       9 allocs/op
BenchmarkGoMemcache/cachebox:get-4           	   10000	    109818 ns/op	     256 B/op	      11 allocs/op
BenchmarkGoMemcache/gomemcache:set-4         	   10000	    103729 ns/op	     112 B/op	       5 allocs/op
BenchmarkGoMemcache/cachebox:set-4           	   10000	    104124 ns/op	     160 B/op	       6 allocs/op
BenchmarkGoMemcache/gomemcache:getmulti-4    	 1000000	      1585 ns/op	     222 B/op	       2 allocs/op
BenchmarkGoMemcache/cachebox:getmulti-4      	 1233427	      2302 ns/op	     225 B/op	       2 allocs/op
BenchmarkGoMemcache/gomemcache:setmulti-4    	    5626	    204885 ns/op	     128 B/op	       7 allocs/op
BenchmarkGoMemcache/cachebox:setmulti-4      	    4981	    245714 ns/op	     366 B/op	       7 allocs/op
BenchmarkGoMemcache/gomemcache:delete-4      	    6922	    189450 ns/op	      16 B/op	       1 allocs/op
BenchmarkGoMemcache/cachebox:delete-4        	    6546	    187937 ns/op	      32 B/op	       2 allocs/op
BenchmarkGoMemcache/gomemcache:deletemulti-4 	    6109	    184948 ns/op	      32 B/op	       3 allocs/op
BenchmarkGoMemcache/cachebox:deletemulti-4   	 2760835	      1006 ns/op	     122 B/op	       2 allocs/op
BenchmarkRedigo/redigo:get-4                 	    1015	   1170355 ns/op	   10016 B/op	      42 allocs/op
BenchmarkRedigo/cachebox:get-4               	     903	   1183235 ns/op	   10063 B/op	      44 allocs/op
BenchmarkRedigo/redigo:set-4                 	     870	   1221171 ns/op	   10162 B/op	      44 allocs/op
BenchmarkRedigo/cachebox:set-4               	     914	   1217707 ns/op	   10257 B/op	      47 allocs/op
BenchmarkRedigo/redigo:getmulti-4            	  963810	      1216 ns/op	     171 B/op	       3 allocs/op
BenchmarkRedigo/cachebox:getmulti-4          	 1047338	      1065 ns/op	     180 B/op	       3 allocs/op
BenchmarkRedigo/redigo:setmulti-4            	  573801	      2225 ns/op	     208 B/op	       5 allocs/op
BenchmarkRedigo/cachebox:setmulti-4          	  196122	      7917 ns/op	     518 B/op	       8 allocs/op
BenchmarkRedigo/redigo:delete-4              	    1141	   1027026 ns/op	    9976 B/op	      40 allocs/op
BenchmarkRedigo/cachebox:delete-4            	     889	   1225767 ns/op	    9992 B/op	      41 allocs/op
BenchmarkRedigo/redigo:deletemulti-4         	 3459343	       639 ns/op	     137 B/op	       3 allocs/op
BenchmarkRedigo/cachebox:deletemulti-4       	 2788777	       752 ns/op	     153 B/op	       3 allocs/op
```