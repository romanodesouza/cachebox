// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/romanodesouza/cachebox"
	"github.com/romanodesouza/cachebox/mock/mock_cachebox"
)

func TestCache_WithKeyLock(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		keys         []string
		cache        func(ctrl *gomock.Controller) *cachebox.Cache
		debouncedSet func() (string, []byte)
		want         [][]byte
		wantErr      error
	}{
		{
			name: "it should return early when items are found in the storage, so no need to use lock",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "key1", "key2").Return([][]byte{
					[]byte("ok"),
					[]byte("ok"),
				}, nil)

				return cachebox.NewCache(store, cachebox.WithKeyLock())
			},
			want:    [][]byte{[]byte("ok"), []byte("ok")},
			wantErr: nil,
		},
		{
			name: "it should block get calls until set is called",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				// miss
				store.EXPECT().MGet(gomock.Any(), "key1", "key2").Return([][]byte{
					nil,
					[]byte("ok"),
				}, nil)

				// expected set to release the lock
				store.EXPECT().Set(gomock.Any(), cachebox.Item{
					Key:   "key1",
					Value: []byte("debounce"),
					TTL:   time.Minute,
				}).Return(nil)

				// second pending call returns nil to get the value from the set
				store.EXPECT().MGet(gomock.Any(), "key1", "key2").Return([][]byte{
					nil,
					[]byte("ok"),
				}, nil)

				return cachebox.NewCache(store, cachebox.WithKeyLock())
			},
			debouncedSet: func() (string, []byte) {
				<-time.After(2 * time.Second)
				return "key1", []byte("debounce")
			},
			want:    [][]byte{[]byte("debounce"), []byte("ok")},
			wantErr: nil,
		},
		{
			name: "it should block until context times out",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				go func() {
					<-time.After(time.Second)
					cancel()
				}()
				return ctx
			}(),
			keys: []string{"key1", "key2"},
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				// miss
				store.EXPECT().MGet(gomock.Any(), "key1", "key2").Return([][]byte{
					nil,
					[]byte("ok"),
				}, nil)

				// second pending call
				store.EXPECT().MGet(gomock.Any(), "key1", "key2").Return([][]byte{
					nil,
					[]byte("ok"),
				}, nil)

				return cachebox.NewCache(store, cachebox.WithKeyLock())
			},
			debouncedSet: func() (string, []byte) {
				<-time.After(2 * time.Second)
				return "key1", []byte("debounce")
			},
			want:    [][]byte{nil, []byte("ok")},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cache := tt.cache(ctrl)

			if tt.debouncedSet != nil {
				_, _ = cache.GetMulti(tt.ctx, tt.keys)

				go func() {
					key, value := tt.debouncedSet()
					_ = cache.Set(tt.ctx, cachebox.Item{
						Key:   key,
						Value: value,
						TTL:   time.Minute,
					})
				}()
			}

			bb, err := cache.GetMulti(tt.ctx, tt.keys)

			if diff := cmp.Diff(tt.want, bb); diff != "" {
				t.Errorf("unexpected result(-want +got):\n%s", diff)
			}

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}
