// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/romanodesouza/cachebox/mock/mock_cachebox"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/romanodesouza/cachebox"
)

func TestCache_Get(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		key     string
		cache   func(ctrl *gomock.Controller) *cachebox.Cache
		want    []byte
		wantErr error
	}{
		{
			name: "it should skip the call when recomputing",
			ctx:  cachebox.WithBypass(context.Background(), cachebox.BypassReading),
			key:  "key",
			cache: func(_ *gomock.Controller) *cachebox.Cache {
				return cachebox.NewCache(nil)
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "it should skip the call when bypassing",
			ctx:  cachebox.WithBypass(context.Background(), cachebox.BypassReadWriting),
			key:  "key",
			cache: func(_ *gomock.Controller) *cachebox.Cache {
				return cachebox.NewCache(nil)
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "it should return the storage error when it occurs",
			ctx:  context.Background(),
			key:  "key",
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "key").Return(nil, errors.New("storage: mget error"))

				return cachebox.NewCache(store)
			},
			want:    nil,
			wantErr: errors.New("storage: mget error"),
		},
		{
			name: "it should return the storage bytes when it succeeds",
			ctx:  context.Background(),
			key:  "key",
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "key").Return([][]byte{[]byte("ok")}, nil)

				return cachebox.NewCache(store)
			},
			want:    []byte("ok"),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cache := tt.cache(ctrl)
			b, err := cache.Get(tt.ctx, tt.key)

			if diff := cmp.Diff(tt.want, b); diff != "" {
				t.Errorf("unexpected result(-want +got):\n%s", diff)
			}

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCache_GetMulti(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		keys    []string
		cache   func(ctrl *gomock.Controller) *cachebox.Cache
		want    [][]byte
		wantErr error
	}{
		{
			name: "it should skip the call when recomputing",
			ctx:  cachebox.WithBypass(context.Background(), cachebox.BypassReading),
			keys: []string{"key1", "key2"},
			cache: func(_ *gomock.Controller) *cachebox.Cache {
				return cachebox.NewCache(nil)
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "it should skip the call when bypassing",
			ctx:  cachebox.WithBypass(context.Background(), cachebox.BypassReadWriting),
			keys: []string{"key1", "key2"},
			cache: func(_ *gomock.Controller) *cachebox.Cache {
				return cachebox.NewCache(nil)
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "it should return the storage error when it occurs",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), []string{"key1", "key2"}).
					Return(nil, errors.New("storage: get multi error"))

				return cachebox.NewCache(store)
			},
			want:    nil,
			wantErr: errors.New("storage: get multi error"),
		},
		{
			name: "it should return the storage bytes when it succeeds",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), []string{"key1", "key2"}).
					Return([][]byte{[]byte("ok"), []byte("ok")}, nil)

				return cachebox.NewCache(store)
			},
			want:    [][]byte{[]byte("ok"), []byte("ok")},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cache := tt.cache(ctrl)
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

func TestCache_Set(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		item    cachebox.Item
		cache   func(ctrl *gomock.Controller) *cachebox.Cache
		wantErr error
	}{
		{
			name: "it should skip the call when bypassing",
			ctx:  cachebox.WithBypass(context.Background(), cachebox.BypassReadWriting),
			item: cachebox.Item{
				Key:   "key",
				Value: []byte("ok"),
				TTL:   time.Minute,
			},
			cache: func(_ *gomock.Controller) *cachebox.Cache {
				return cachebox.NewCache(nil)
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
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().Set(gomock.Any(), cachebox.Item{
					Key:   "key",
					Value: []byte("ok"),
					TTL:   time.Minute,
				}).Return(errors.New("storage: set error"))

				return cachebox.NewCache(store)
			},
			wantErr: errors.New("storage: set error"),
		},
		{
			name: "it should return nil when it succeeds",
			ctx:  context.Background(),
			item: cachebox.Item{
				Key:   "key",
				Value: []byte("ok"),
				TTL:   time.Minute,
			},
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().Set(gomock.Any(), cachebox.Item{
					Key:   "key",
					Value: []byte("ok"),
					TTL:   time.Minute,
				}).Return(nil)

				return cachebox.NewCache(store)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cache := tt.cache(ctrl)
			err := cache.Set(tt.ctx, tt.item)

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCache_SetMulti(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		items   []cachebox.Item
		cache   func(ctrl *gomock.Controller) *cachebox.Cache
		wantErr error
	}{
		{
			name: "it should skip the call when bypassing",
			ctx:  cachebox.WithBypass(context.Background(), cachebox.BypassReadWriting),
			items: []cachebox.Item{
				{
					Key:   "key1",
					Value: []byte("ok"),
					TTL:   time.Minute,
				},
				{
					Key:   "key2",
					Value: []byte("ok"),
					TTL:   time.Minute,
				},
			},
			cache: func(_ *gomock.Controller) *cachebox.Cache {
				return cachebox.NewCache(nil)
			},
			wantErr: nil,
		},
		{
			name: "it should return the storage error when it occurs",
			ctx:  context.Background(),
			items: []cachebox.Item{
				{
					Key:   "key1",
					Value: []byte("ok"),
					TTL:   time.Minute,
				},
				{
					Key:   "key2",
					Value: []byte("ok"),
					TTL:   time.Minute,
				},
			},
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().Set(gomock.Any(), []cachebox.Item{
					{
						Key:   "key1",
						Value: []byte("ok"),
						TTL:   time.Minute,
					},
					{
						Key:   "key2",
						Value: []byte("ok"),
						TTL:   time.Minute,
					},
				}).Return(errors.New("storage: set error"))

				return cachebox.NewCache(store)
			},
			wantErr: errors.New("storage: set error"),
		},
		{
			name: "it should return nil when it succeeds",
			ctx:  context.Background(),
			items: []cachebox.Item{
				{
					Key:   "key1",
					Value: []byte("ok"),
					TTL:   time.Minute,
				},
				{
					Key:   "key2",
					Value: []byte("ok"),
					TTL:   time.Minute,
				},
			},
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().Set(gomock.Any(), []cachebox.Item{
					{
						Key:   "key1",
						Value: []byte("ok"),
						TTL:   time.Minute,
					},
					{
						Key:   "key2",
						Value: []byte("ok"),
						TTL:   time.Minute,
					},
				}).Return(nil)

				return cachebox.NewCache(store)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cache := tt.cache(ctrl)
			err := cache.SetMulti(tt.ctx, tt.items)

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCache_Delete(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		key     string
		cache   func(ctrl *gomock.Controller) *cachebox.Cache
		wantErr error
	}{
		{
			name: "it should skip the call when bypassing",
			ctx:  cachebox.WithBypass(context.Background(), cachebox.BypassReadWriting),
			key:  "key",
			cache: func(_ *gomock.Controller) *cachebox.Cache {
				return cachebox.NewCache(nil)
			},
			wantErr: nil,
		},
		{
			name: "it should return the storage error when it occurs",
			ctx:  context.Background(),
			key:  "key",
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().Delete(gomock.Any(), "key").Return(errors.New("storage: delete error"))

				return cachebox.NewCache(store)
			},
			wantErr: errors.New("storage: delete error"),
		},
		{
			name: "it should return nil when it succeeds",
			ctx:  context.Background(),
			key:  "key",
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().Delete(gomock.Any(), "key").Return(nil)

				return cachebox.NewCache(store)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cache := tt.cache(ctrl)
			err := cache.Delete(tt.ctx, tt.key)

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCache_DeleteMulti(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		keys    []string
		cache   func(ctrl *gomock.Controller) *cachebox.Cache
		wantErr error
	}{
		{
			name: "it should skip the call when bypassing",
			ctx:  cachebox.WithBypass(context.Background(), cachebox.BypassReadWriting),
			keys: []string{"key1", "key2"},
			cache: func(_ *gomock.Controller) *cachebox.Cache {
				return cachebox.NewCache(nil)
			},
			wantErr: nil,
		},
		{
			name: "it should return the storage error when it occurs",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().Delete(gomock.Any(), "key1", "key2").Return(errors.New("storage: delete error"))

				return cachebox.NewCache(store)
			},
			wantErr: errors.New("storage: delete error"),
		},
		{
			name: "it should return nil when it succeeds",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			cache: func(ctrl *gomock.Controller) *cachebox.Cache {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().Delete(gomock.Any(), "key1", "key2").Return(nil)

				return cachebox.NewCache(store)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cache := tt.cache(ctrl)
			err := cache.DeleteMulti(tt.ctx, tt.keys)

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}
