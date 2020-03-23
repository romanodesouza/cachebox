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
	"github.com/romanodesouza/cachebox"
	"github.com/romanodesouza/cachebox/mock/mock_storage"
	"github.com/romanodesouza/cachebox/storage"
)

func TestCacheNS_GetMostRecentTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		keys    []string
		ctx     context.Context
		cachens func(*gomock.Controller) *cachebox.CacheNS
		want    int64
		wantErr error
	}{
		{
			name: "it should skip the call when bypassing",
			ctx:  cachebox.WithBypass(context.Background()),
			keys: []string{"nskey1", "nskey2"},
			cachens: func(_ *gomock.Controller) *cachebox.CacheNS {
				now := time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
				cachebox.Now = func() time.Time { return now }
				cachens := cachebox.NewCacheNS(nil)
				return cachens
			},
			want:    1577840461000000001,
			wantErr: nil,
		},
		{
			name: "it should return the most recent timestamp",
			ctx:  context.Background(),
			keys: []string{"nskey1", "nskey2"},
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_storage.NewMockNamespaceStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2").
					Return([][]byte{
						marshalInt64(1577840451000000001),
						marshalInt64(1577840461000000001),
					}, nil)

				cachens := cachebox.NewCacheNS(store)
				return cachens
			},
			want:    1577840461000000001,
			wantErr: nil,
		},
		{
			name: "it should recompute and set any miss, but also return the most recent timestamp",
			ctx:  context.Background(),
			keys: []string{"nskey1", "nskey2", "nskey3", "nskey4"},
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_storage.NewMockNamespaceStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1", "nskey2", "nskey3", "nskey4").
					Return([][]byte{
						marshalInt64(1577840441000000001),
						marshalInt64(1577840451000000001),
						nil,
						nil,
					}, nil)

				now := time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
				store.EXPECT().Set(gomock.Any(), []storage.Item{
					{Key: "nskey3", Value: marshalInt64(now.UnixNano()), TTL: 24 * time.Hour},
					{Key: "nskey4", Value: marshalInt64(now.UnixNano()), TTL: 24 * time.Hour},
				})

				cachebox.Now = func() time.Time { return now }
				cachens := cachebox.NewCacheNS(store)
				return cachens
			},
			want:    1577840461000000001,
			wantErr: nil,
		},
		{
			name: "it should use user defined ttl",
			keys: []string{"nskey1"},
			ctx:  context.Background(),
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_storage.NewMockNamespaceStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1").
					Return([][]byte{nil}, nil)

				now := time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
				store.EXPECT().Set(gomock.Any(), []storage.Item{
					{Key: "nskey1", Value: marshalInt64(now.UnixNano()), TTL: time.Hour},
				})

				cachebox.Now = func() time.Time { return now }
				cachens := cachebox.NewCacheNS(store, cachebox.WithNamespaceKeyTTL(time.Hour))
				return cachens
			},
			want:    1577840461000000001,
			wantErr: nil,
		},
		{
			name: "it should return storage error on set and current clock time",
			ctx:  context.Background(),
			keys: []string{"nskey1"},
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_storage.NewMockNamespaceStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1").
					Return([][]byte{nil}, nil)

				now := time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
				store.EXPECT().Set(gomock.Any(), []storage.Item{
					{Key: "nskey1", Value: marshalInt64(now.UnixNano()), TTL: 24 * time.Hour},
				}).Return(errors.New("namespacestorage: set error"))

				cachebox.Now = func() time.Time { return now }
				cachens := cachebox.NewCacheNS(store)
				return cachens
			},
			want:    1577840461000000001,
			wantErr: errors.New("namespacestorage: set error"),
		},
		{
			name: "it should return storage error on mget and current clock time",
			ctx:  context.Background(),
			keys: []string{"nskey1"},
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_storage.NewMockNamespaceStorage(ctrl)
				store.EXPECT().MGet(gomock.Any(), "nskey1").
					Return([][]byte{nil}, errors.New("namespacestorage: mget error"))

				now := time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
				cachebox.Now = func() time.Time { return now }
				cachens := cachebox.NewCacheNS(store)
				return cachens
			},
			want:    1577840461000000001,
			wantErr: errors.New("namespacestorage: mget error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cachens := tt.cachens(ctrl)
			timestamp, err := cachens.GetMostRecentTimestamp(tt.ctx, tt.keys...)

			if timestamp != tt.want {
				t.Errorf("got %d; want %d", timestamp, tt.want)
			}

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCacheNS_Delete(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		key     string
		cachens func(ctrl *gomock.Controller) *cachebox.CacheNS
		wantErr error
	}{
		{
			name: "it should skip the call when bypassing",
			ctx:  cachebox.WithBypass(context.Background()),
			key:  "nskey",
			cachens: func(_ *gomock.Controller) *cachebox.CacheNS {
				return cachebox.NewCacheNS(nil)
			},
			wantErr: nil,
		},
		{
			name: "it should return the storage error when it occurs",
			ctx:  context.Background(),
			key:  "nskey",
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_storage.NewMockNamespaceStorage(ctrl)
				store.EXPECT().Delete(gomock.Any(), "nskey").Return(errors.New("namespacestorage: delete error"))

				return cachebox.NewCacheNS(store)
			},
			wantErr: errors.New("namespacestorage: delete error"),
		},
		{
			name: "it should return nil when it succeeds",
			ctx:  context.Background(),
			key:  "nskey",
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_storage.NewMockNamespaceStorage(ctrl)
				store.EXPECT().Delete(gomock.Any(), "nskey").Return(nil)

				return cachebox.NewCacheNS(store)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cachens := tt.cachens(ctrl)
			err := cachens.Delete(tt.ctx, tt.key)

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCacheNS_DeleteMulti(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		keys    []string
		cachens func(ctrl *gomock.Controller) *cachebox.CacheNS
		wantErr error
	}{
		{
			name: "it should skip the call when bypassing",
			ctx:  cachebox.WithBypass(context.Background()),
			keys: []string{"nskey1", "nskey2"},
			cachens: func(_ *gomock.Controller) *cachebox.CacheNS {
				return cachebox.NewCacheNS(nil)
			},
			wantErr: nil,
		},
		{
			name: "it should return the storage error when it occurs",
			ctx:  context.Background(),
			keys: []string{"nskey1", "nskey2"},
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_storage.NewMockNamespaceStorage(ctrl)
				store.EXPECT().Delete(gomock.Any(), "nskey1", "nskey2").
					Return(errors.New("namespacestorage: delete error"))

				return cachebox.NewCacheNS(store)
			},
			wantErr: errors.New("namespacestorage: delete error"),
		},
		{
			name: "it should return nil when it succeeds",
			ctx:  context.Background(),
			keys: []string{"nskey1", "nskey2"},
			cachens: func(ctrl *gomock.Controller) *cachebox.CacheNS {
				store := mock_storage.NewMockNamespaceStorage(ctrl)
				store.EXPECT().Delete(gomock.Any(), "nskey1", "nskey2").Return(nil)

				return cachebox.NewCacheNS(store)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cachens := tt.cachens(ctrl)
			err := cachens.DeleteMulti(tt.ctx, tt.keys)

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
