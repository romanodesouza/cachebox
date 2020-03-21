// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package storage_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/romanodesouza/cachebox/mock/mock_storage"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/romanodesouza/cachebox/storage"
)

func TestMultiStorage_MGet(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		keys         []string
		multistorage func(ctrl *gomock.Controller) *storage.MultiStorage
		want         [][]byte
		wantErr      error
	}{
		{
			name: "it should try first to return everything from the first storage",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			multistorage: func(ctrl *gomock.Controller) *storage.MultiStorage {
				store1 := mock_storage.NewMockStorage(ctrl)
				store1.EXPECT().MGet(gomock.Any(), gomock.Any()).Return([][]byte{[]byte("ok"), []byte("ok")}, nil)
				store2 := mock_storage.NewMockStorage(ctrl)

				return storage.NewMultiStorage(store1, store2)
			},
			want:    [][]byte{[]byte("ok"), []byte("ok")},
			wantErr: nil,
		},
		{
			name: "it should return early in case of error",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			multistorage: func(ctrl *gomock.Controller) *storage.MultiStorage {
				store1 := mock_storage.NewMockStorage(ctrl)
				store1.EXPECT().MGet(gomock.Any(), gomock.Any()).Return(nil, errors.New("store1: mget error"))
				store2 := mock_storage.NewMockStorage(ctrl)

				return storage.NewMultiStorage(store1, store2)
			},
			want:    nil,
			wantErr: errors.New("store1: mget error"),
		},
		{
			name: "it should try all storages to fetch the data",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			multistorage: func(ctrl *gomock.Controller) *storage.MultiStorage {
				store1 := mock_storage.NewMockStorage(ctrl)
				store1.EXPECT().MGet(gomock.Any(), "key1", "key2").Return([][]byte{[]byte("ok"), nil}, nil)
				store2 := mock_storage.NewMockStorage(ctrl)
				store2.EXPECT().MGet(gomock.Any(), "key2").Return([][]byte{[]byte("ok")}, nil)

				return storage.NewMultiStorage(store1, store2)
			},
			want:    [][]byte{[]byte("ok"), []byte("ok")},
			wantErr: nil,
		},
		{
			name: "it should keep nil for not found items",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			multistorage: func(ctrl *gomock.Controller) *storage.MultiStorage {
				store1 := mock_storage.NewMockStorage(ctrl)
				store1.EXPECT().MGet(gomock.Any(), "key1", "key2").Return([][]byte{[]byte("ok"), nil}, nil)
				store2 := mock_storage.NewMockStorage(ctrl)
				store2.EXPECT().MGet(gomock.Any(), "key2").Return([][]byte{nil}, nil)

				return storage.NewMultiStorage(store1, store2)
			},
			want:    [][]byte{[]byte("ok"), nil},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ms := tt.multistorage(ctrl)
			bb, err := ms.MGet(tt.ctx, tt.keys...)

			if diff := cmp.Diff(tt.want, bb); diff != "" {
				t.Errorf("unexpected result(-want +got):\n%s", diff)
			}

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultiStorage_Set(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		items        []storage.Item
		multistorage func(ctrl *gomock.Controller) *storage.MultiStorage
		wantErr      error
	}{
		{
			name:  "it should set in all storages",
			ctx:   context.Background(),
			items: []storage.Item{{Key: "key1"}, {Key: "key2"}},
			multistorage: func(ctrl *gomock.Controller) *storage.MultiStorage {
				store1 := mock_storage.NewMockStorage(ctrl)
				store1.EXPECT().Set(gomock.Any(), storage.Item{Key: "key1"}, storage.Item{Key: "key2"}).Return(nil)
				store2 := mock_storage.NewMockStorage(ctrl)
				store2.EXPECT().Set(gomock.Any(), storage.Item{Key: "key1"}, storage.Item{Key: "key2"}).Return(nil)

				return storage.NewMultiStorage(store1, store2)
			},
			wantErr: nil,
		},
		{
			name:  "it should return early in case of error",
			ctx:   context.Background(),
			items: []storage.Item{{Key: "key1"}, {Key: "key2"}},
			multistorage: func(ctrl *gomock.Controller) *storage.MultiStorage {
				store1 := mock_storage.NewMockStorage(ctrl)
				store1.EXPECT().Set(gomock.Any(), gomock.Any()).Return(errors.New("store1: set error"))
				store2 := mock_storage.NewMockStorage(ctrl)

				return storage.NewMultiStorage(store1, store2)
			},
			wantErr: errors.New("store1: set error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ms := tt.multistorage(ctrl)
			err := ms.Set(tt.ctx, tt.items...)

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultiStorage_Delete(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		keys         []string
		multistorage func(ctrl *gomock.Controller) *storage.MultiStorage
		wantErr      error
	}{
		{
			name: "it should delete from all storages",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			multistorage: func(ctrl *gomock.Controller) *storage.MultiStorage {
				store1 := mock_storage.NewMockStorage(ctrl)
				store1.EXPECT().Delete(gomock.Any(), "key1", "key2").Return(nil)
				store2 := mock_storage.NewMockStorage(ctrl)
				store2.EXPECT().Delete(gomock.Any(), "key1", "key2").Return(nil)

				return storage.NewMultiStorage(store1, store2)
			},
			wantErr: nil,
		},
		{
			name: "it should return early in case of error",
			ctx:  context.Background(),
			keys: []string{"key1", "key2"},
			multistorage: func(ctrl *gomock.Controller) *storage.MultiStorage {
				store1 := mock_storage.NewMockStorage(ctrl)
				store1.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errors.New("store1: delete error"))
				store2 := mock_storage.NewMockStorage(ctrl)

				return storage.NewMultiStorage(store1, store2)
			},
			wantErr: errors.New("store1: delete error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ms := tt.multistorage(ctrl)
			err := ms.Delete(tt.ctx, tt.keys...)

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}
