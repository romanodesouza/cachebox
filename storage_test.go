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

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/romanodesouza/cachebox"
	"github.com/romanodesouza/cachebox/mock/mock_cachebox"
)

func TestNewStorageWrapper(t *testing.T) {
	t.Run("it should append hooks on same instance to reuse loops", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		store := mock_cachebox.NewMockStorage(ctrl)
		store.EXPECT().MGet(gomock.Any(), gomock.Any()).Return([][]byte{[]byte("ok")}, nil)

		wrap := cachebox.NewStorageWrapper(store, cachebox.StorageHooks{
			AfterMGet: func(ctx context.Context, key string, b []byte) ([]byte, error) {
				return append(b, []byte("first")...), nil
			},
		})

		wrap = cachebox.NewStorageWrapper(wrap, cachebox.StorageHooks{
			AfterMGet: func(ctx context.Context, key string, b []byte) ([]byte, error) {
				return append(b, []byte("second")...), nil
			},
		})

		ctx := context.Background()
		bb, _ := wrap.MGet(ctx, "key1")

		want := [][]byte{[]byte("okfirstsecond")}
		if diff := cmp.Diff(want, bb); diff != "" {
			t.Errorf("unexpected result(-want +got):\n%s", diff)
		}
	})
}

func TestNewStorageWrapper_MGet(t *testing.T) {
	tests := []struct {
		name    string
		storage func(ctrl *gomock.Controller) cachebox.Storage
		hooks   []cachebox.StorageHooks
		keys    []string
		want    [][]byte
		wantErr error
	}{
		{
			name: "it should return early the storage error",
			storage: func(ctrl *gomock.Controller) cachebox.Storage {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), gomock.Any()).Return(nil, errors.New("storage: mget error"))
				return store
			},
			hooks: []cachebox.StorageHooks{{
				AfterMGet: func(ctx context.Context, key string, b []byte) ([]byte, error) {
					return b, nil
				},
			}},
			keys:    []string{"key1", "key2"},
			want:    nil,
			wantErr: errors.New("storage: mget error"),
		},
		{
			name: "it should return hook error",
			storage: func(ctrl *gomock.Controller) cachebox.Storage {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), gomock.Any()).Return([][]byte{[]byte("ok"), []byte("ok")}, nil)
				return store
			},
			hooks: []cachebox.StorageHooks{{
				AfterMGet: func(ctx context.Context, key string, b []byte) ([]byte, error) {
					return nil, errors.New("hooks: after mget error")
				},
			}},
			keys:    []string{"key1", "key2"},
			want:    nil,
			wantErr: errors.New("hooks: after mget error"),
		},
		{
			name: "it should transform the bytes after calling storage mget",
			storage: func(ctrl *gomock.Controller) cachebox.Storage {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), gomock.Any()).Return(
					[][]byte{
						[]byte("transform"),
						[]byte("me"),
					},
					nil,
				)
				return store
			},
			hooks: []cachebox.StorageHooks{{
				AfterMGet: func(ctx context.Context, key string, b []byte) ([]byte, error) {
					return append(b, []byte(" transformed")...), nil
				},
			}},
			keys: []string{"key1", "key2"},
			want: [][]byte{
				[]byte("transform transformed"),
				[]byte("me transformed"),
			},
			wantErr: nil,
		},
		{
			name: "it should run all after mget hooks in sequence",
			storage: func(ctrl *gomock.Controller) cachebox.Storage {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), gomock.Any()).Return(
					[][]byte{
						[]byte("transform"),
						[]byte("me"),
					},
					nil,
				)
				return store
			},
			hooks: []cachebox.StorageHooks{
				{
					AfterMGet: func(ctx context.Context, key string, b []byte) ([]byte, error) {
						return append(b, []byte("1")...), nil
					},
				},
				{
					AfterMGet: func(ctx context.Context, key string, b []byte) ([]byte, error) {
						return append(b, []byte("2")...), nil
					},
				},
			},
			keys: []string{"key1", "key2"},
			want: [][]byte{
				[]byte("transform12"),
				[]byte("me12"),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			wrap := cachebox.NewStorageWrapper(tt.storage(ctrl), tt.hooks...)
			bb, err := wrap.MGet(ctx, tt.keys...)

			if diff := cmp.Diff(tt.want, bb); diff != "" {
				t.Errorf("unexpected result(-want +got):\n%s", diff)
			}

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func TestStorageWrapper_Set(t *testing.T) {
	tests := []struct {
		name    string
		storage func(ctrl *gomock.Controller) cachebox.Storage
		hooks   []cachebox.StorageHooks
		items   []cachebox.Item
		wantErr error
	}{
		{
			name: "it should return early the storage error",
			storage: func(ctrl *gomock.Controller) cachebox.Storage {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().Set(gomock.Any(), gomock.Any()).Return(errors.New("storage: set error"))
				return store
			},
			hooks: []cachebox.StorageHooks{{
				BeforeSet: func(ctx context.Context, item cachebox.Item) (cachebox.Item, error) {
					return item, nil
				},
			}},
			wantErr: errors.New("storage: set error"),
		},
		{
			name: "it should return hook error",
			storage: func(ctrl *gomock.Controller) cachebox.Storage {
				store := mock_cachebox.NewMockStorage(ctrl)
				return store
			},
			hooks: []cachebox.StorageHooks{{
				BeforeSet: func(ctx context.Context, item cachebox.Item) (cachebox.Item, error) {
					return item, errors.New("hooks: before set error")
				},
			}},
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
			wantErr: errors.New("hooks: before set error"),
		},
		{
			name: "it should transform the bytes before calling storage set",
			storage: func(ctrl *gomock.Controller) cachebox.Storage {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().Set(gomock.Any(), []cachebox.Item{
					{
						Key:   "key1",
						Value: []byte("transform transformed"),
						TTL:   time.Minute,
					},
					{
						Key:   "key2",
						Value: []byte("me transformed"),
						TTL:   time.Minute,
					},
				}).Return(nil)
				return store
			},
			hooks: []cachebox.StorageHooks{{
				BeforeSet: func(ctx context.Context, item cachebox.Item) (cachebox.Item, error) {
					item.Value = append(item.Value, []byte(" transformed")...)
					return item, nil
				},
			}},
			items: []cachebox.Item{
				{
					Key:   "key1",
					Value: []byte("transform"),
					TTL:   time.Minute,
				},
				{
					Key:   "key2",
					Value: []byte("me"),
					TTL:   time.Minute,
				},
			},
			wantErr: nil,
		},
		{
			name: "it should run all before set hooks in sequence",
			storage: func(ctrl *gomock.Controller) cachebox.Storage {
				store := mock_cachebox.NewMockStorage(ctrl)
				store.EXPECT().Set(gomock.Any(), []cachebox.Item{
					{
						Key:   "key1",
						Value: []byte("transform12"),
						TTL:   time.Minute,
					},
					{
						Key:   "key2",
						Value: []byte("me12"),
						TTL:   time.Minute,
					},
				}).Return(nil)
				return store
			},
			hooks: []cachebox.StorageHooks{
				{
					BeforeSet: func(ctx context.Context, item cachebox.Item) (cachebox.Item, error) {
						item.Value = append(item.Value, []byte("1")...)
						return item, nil
					},
				},
				{
					BeforeSet: func(ctx context.Context, item cachebox.Item) (cachebox.Item, error) {
						item.Value = append(item.Value, []byte("2")...)
						return item, nil
					},
				},
			},
			items: []cachebox.Item{
				{
					Key:   "key1",
					Value: []byte("transform"),
					TTL:   time.Minute,
				},
				{
					Key:   "key2",
					Value: []byte("me"),
					TTL:   time.Minute,
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			wrap := cachebox.NewStorageWrapper(tt.storage(ctrl), tt.hooks...)
			err := wrap.Set(ctx, tt.items...)

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}
