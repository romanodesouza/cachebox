// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox_test

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/romanodesouza/cachebox"
	"github.com/romanodesouza/cachebox/mock/mock_cachebox"
)

func TestCacheNS_Get(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		key     string
		cachens func(ctrl *gomock.Controller) *cachebox.CacheNS
		want    []byte
		wantErr error
	}{
		{
			name: "it should return the storage error when it occurs",
			ctx:  context.Background(),
			key:  "key",
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2", "cachebox:rk:key").
					Return(nil, errors.New("storage: mget error"))

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			want:    nil,
			wantErr: errors.New("storage: mget error"),
		},
		{
			name: "it should get hit when the item version is newer or equal than the namespaces",
			ctx:  context.Background(),
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2", "cachebox:rk:key").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						marshalInt64(1577840461000000001),
						append(marshalInt64(1577840461000000001), []byte("ok")...),
					}, nil)

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			key:     "key",
			want:    []byte("ok"),
			wantErr: nil,
		},
		{
			name: "it should get miss when the item version is older than the namespaces",
			ctx:  context.Background(),
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2", "cachebox:rk:key").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						marshalInt64(1577840461000000001),
						append(marshalInt64(1577840441000000001), []byte("ok")...),
					}, nil)

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			key:     "key",
			want:    nil,
			wantErr: nil,
		},
		{
			name: "it should get miss when the item is expired",
			ctx:  context.Background(),
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2", "cachebox:rk:key").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						marshalInt64(1577840461000000001),
						nil,
					}, nil)

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			key:     "key",
			want:    nil,
			wantErr: nil,
		},
		{
			name: "it should force a miss in case of bypass, after setting the namespace version",
			ctx:  cachebox.WithBypass(context.Background()),
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2", "cachebox:rk:key").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						marshalInt64(1577840461000000001),
						append(marshalInt64(1577840461000000001), []byte("ok")...),
					}, nil)

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			key:     "key",
			want:    nil,
			wantErr: nil,
		},
		{
			name: "it should force a miss in case of recompute, after setting the namespace version",
			ctx:  cachebox.WithRecompute(context.Background()),
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2", "cachebox:rk:key").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						marshalInt64(1577840461000000001),
						append(marshalInt64(1577840461000000001), []byte("ok")...),
					}, nil)

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			key:     "key",
			want:    nil,
			wantErr: nil,
		},
		{
			name: "it should set the most recent timestamp is case of any namespace miss",
			ctx:  context.Background(),
			key:  "key",
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2", "cachebox:rk:key").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						nil,
						append(marshalInt64(1577840461000000001), []byte("ok")...),
					}, nil)

				cachebox.Now = func() time.Time {
					return time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
				}
				store.EXPECT().Set(gomock.Any(), []cachebox.Item{
					{Key: "nskey2", Value: marshalInt64(cachebox.Now().UnixNano()), TTL: 12 * time.Hour},
				})

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			want:    []byte("ok"),
			wantErr: nil,
		},
		{
			name: "it should use user defined default ttl",
			ctx:  context.Background(),
			key:  "key",
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2", "cachebox:rk:key").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						nil,
						append(marshalInt64(1577840461000000001), []byte("ok")...),
					}, nil)

				cachebox.Now = func() time.Time {
					return time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
				}
				store.EXPECT().Set(gomock.Any(), []cachebox.Item{
					{Key: "nskey2", Value: marshalInt64(cachebox.Now().UnixNano()), TTL: time.Hour},
				})

				cache := cachebox.NewCache(store, cachebox.WithDefaultNamespaceTTL(time.Hour))
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			want:    []byte("ok"),
			wantErr: nil,
		},
		{
			name: "it should return storage error when setting the most recent timestamp is case of any namespace miss",
			ctx:  context.Background(),
			key:  "key",
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2", "cachebox:rk:key").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						nil,
						append(marshalInt64(1577840461000000001), []byte("ok")...),
					}, nil)

				cachebox.Now = func() time.Time {
					return time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
				}
				store.EXPECT().Set(gomock.Any(), []cachebox.Item{
					{Key: "nskey2", Value: marshalInt64(cachebox.Now().UnixNano()), TTL: 12 * time.Hour},
				}).Return(errors.New("storage: set error"))

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			want:    nil,
			wantErr: errors.New("storage: set error"),
		},
		{
			name: "it should use versioned keys on key-based expiration strategy",
			ctx:  context.Background(),
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						marshalInt64(1577840461000000001),
					}, nil)
				store.EXPECT().MGet(gomock.Any(), "cachebox:v1577840461000000001:key").
					Return([][]byte{
						[]byte("ok"),
					}, nil)

				cache := cachebox.NewCache(store, cachebox.WithKeyBasedExpiration())
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			key:     "key",
			want:    []byte("ok"),
			wantErr: nil,
		},
		{
			name: "it should return the storage error when retrieving item by versioned key when it occurs",
			ctx:  context.Background(),
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						marshalInt64(1577840461000000001),
					}, nil)
				store.EXPECT().MGet(gomock.Any(), "cachebox:v1577840461000000001:key").
					Return(nil, errors.New("storage: mget error"))

				cache := cachebox.NewCache(store, cachebox.WithKeyBasedExpiration())
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			key:     "key",
			want:    nil,
			wantErr: errors.New("storage: mget error"),
		},
		{
			name: "it should calculate the namespace version only once",
			ctx:  context.Background(),
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2", "cachebox:rk:warmkey").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						marshalInt64(1577840461000000001),
						append(marshalInt64(1577840461000000001), []byte("warm")...),
					}, nil)
				store.EXPECT().MGet(gomock.Any(), "cachebox:rk:key").Return([][]byte{
					append(marshalInt64(1577840461000000001), []byte("ok")...),
				}, nil)

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")

				// Force the namespace version calculation
				_, _ = cachens.Get(context.Background(), "warmkey")

				return cachens
			},
			key:     "key",
			want:    []byte("ok"),
			wantErr: nil,
		},
		{
			name: "it should calculate the namespace version only once on key-based expiration",
			ctx:  context.Background(),
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						marshalInt64(1577840461000000001),
					}, nil)
				store.EXPECT().MGet(gomock.Any(), "cachebox:v1577840461000000001:warmkey").Return([][]byte{
					[]byte("ok"),
				}, nil)
				store.EXPECT().MGet(gomock.Any(), "cachebox:v1577840461000000001:key").Return([][]byte{
					[]byte("ok"),
				}, nil)

				cache := cachebox.NewCache(store, cachebox.WithKeyBasedExpiration())
				cachens := cache.Namespace("nskey1", "nskey2")

				// Force the namespace version calculation
				_, _ = cachens.Get(context.Background(), "warmkey")

				return cachens
			},
			key:     "key",
			want:    []byte("ok"),
			wantErr: nil,
		},
		{
			name: "it should return the storage error when it occurs (with namespace version previously calculated)",
			ctx:  context.Background(),
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						marshalInt64(1577840461000000001),
					}, nil)
				store.EXPECT().MGet(gomock.Any(), "cachebox:v1577840461000000001:warmkey").Return([][]byte{
					[]byte("ok"),
				}, nil)
				store.EXPECT().MGet(gomock.Any(), "cachebox:v1577840461000000001:key").Return(nil,
					errors.New("storage: mget error"))

				cache := cachebox.NewCache(store, cachebox.WithKeyBasedExpiration())
				cachens := cache.Namespace("nskey1", "nskey2")

				// Force the namespace version calculation
				_, _ = cachens.Get(context.Background(), "warmkey")

				return cachens
			},
			key:     "key",
			want:    nil,
			wantErr: errors.New("storage: mget error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cachens := tt.cachens(ctrl)
			b, err := cachens.Get(tt.ctx, tt.key)

			if diff := cmp.Diff(tt.want, b); diff != "" {
				t.Errorf("unexpected result(-want +got):\n%s", diff)
			}

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCacheNS_Set(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		item    cachebox.Item
		cachens func(ctrl *gomock.Controller) *cachebox.CacheNS
		wantErr error
	}{
		{
			name: "it should skip the call when bypassing",
			ctx:  cachebox.WithBypass(context.Background()),
			item: cachebox.Item{
				Key:   "key",
				Value: []byte("ok"),
				TTL:   time.Minute,
			},
			cachens: func(_ *gomock.Controller) *cachebox.CacheNS {
				return cachebox.NewCacheNS(nil, nil)
			},
			wantErr: nil,
		},
		{
			name: "it should fetch the namespace version if needed",
			ctx:  context.Background(),
			item: cachebox.Item{
				Key:   "key",
				Value: []byte("ok"),
				TTL:   time.Minute,
			},
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2").Return([][]byte{
					marshalInt64(1577840451000000001),
					marshalInt64(1577840461000000001),
				}, nil)
				store.EXPECT().Set(gomock.Any(), cachebox.Item{
					Key:   "cachebox:rk:key",
					Value: append(marshalInt64(1577840461000000001), []byte("ok")...),
					TTL:   time.Minute,
				})

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			wantErr: nil,
		},
		{
			name: "it should use the key-based strategy",
			ctx:  context.Background(),
			item: cachebox.Item{
				Key:   "key",
				Value: []byte("ok"),
				TTL:   time.Minute,
			},
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2").Return([][]byte{
					marshalInt64(1577840451000000001),
					marshalInt64(1577840461000000001),
				}, nil)
				store.EXPECT().Set(gomock.Any(), cachebox.Item{
					Key:   "cachebox:v1577840461000000001:key",
					Value: []byte("ok"),
					TTL:   time.Minute,
				})

				cache := cachebox.NewCache(store, cachebox.WithKeyBasedExpiration())
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			wantErr: nil,
		},
		{
			name: "it should return the storage error when it occurs",
			ctx:  context.Background(),
			item: cachebox.Item{
				Key:   "key",
				Value: []byte("ok"),
				TTL:   time.Minute,
			},
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2").Return(nil, errors.New("storage: mget error"))

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			wantErr: errors.New("storage: mget error"),
		},
		{
			name: "it should return an error when trying to set a timestamp in storage",
			ctx:  context.Background(),
			item: cachebox.Item{
				Key:   "key",
				Value: []byte("ok"),
				TTL:   time.Minute,
			},
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2").Return([][]byte{
					marshalInt64(1577840451000000001),
					nil,
				}, nil)
				store.EXPECT().Set(gomock.Any(), gomock.Any()).Return(errors.New("storage: set error"))

				cache := cachebox.NewCache(store)
				cachens := cache.Namespace("nskey1", "nskey2")
				return cachens
			},
			wantErr: errors.New("storage: set error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cachens := tt.cachens(ctrl)
			err := cachens.Set(tt.ctx, tt.item)

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func marshalInt64(i int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(i))

	return b
}
