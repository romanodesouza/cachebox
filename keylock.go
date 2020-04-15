// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox

import (
	"context"
	"sync"
	"sync/atomic"
)

// WithKeyLock enables a pessimistic lock when retrieving a value from multiple calls to avoid cache stampede.
//
// The first get call which receives a miss returns to the caller and the following get calls are blocked until
// set is called or context times out.
func WithKeyLock() func(*Cache) {
	ct := &contention{
		items: make(map[string]*item),
	}

	return func(c *Cache) {
		c.storage = NewStorageWrapper(c.storage, StorageHooks{
			AfterSet:  ct.AfterSet,
			AfterMGet: ct.AfterMGet,
		})
	}
}

type item struct {
	pending int32
	b       []byte
	done    chan struct{}
}

func (i *item) incrPending() { atomic.AddInt32(&i.pending, 1) }

func (i *item) decrPending() { atomic.AddInt32(&i.pending, -1) }

func (i *item) totPending() int32 { return atomic.AddInt32(&i.pending, 0) }

// contention represents a thread-safe structure to fetch items with i/o contention.
type contention struct {
	sync.Mutex
	items map[string]*item
}

func (c *contention) AfterMGet(ctx context.Context, key string, b []byte) ([]byte, error) {
	// Returns early in case of hit
	if b != nil {
		return b, nil
	}

	c.Lock()
	i, ok := c.items[key]

	if !ok {
		c.items[key] = &item{
			done: make(chan struct{}),
		}
		c.Unlock()

		return nil, nil
	}

	c.Unlock()
	i.incrPending()

	select {
	case <-i.done:
	case <-ctx.Done():
	}

	i.decrPending()

	// Deletes the item after all pending blocks have received it
	if i.totPending() == 0 {
		c.delete(key)
	}

	return i.b, nil
}

func (c *contention) AfterSet(_ context.Context, item Item) error {
	c.Lock()
	i, ok := c.items[item.Key]

	if ok {
		i.b = item.Value
		close(i.done)
	}
	c.Unlock()

	// Ensure the itme deletion in case of no following gets
	if i.totPending() == 0 {
		c.delete(item.Key)
	}

	return nil
}

func (c *contention) delete(key string) {
	c.Lock()
	delete(c.items, key)
	c.Unlock()
}
