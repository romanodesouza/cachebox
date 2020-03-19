// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/romanodesouza/cachebox"
	"github.com/romanodesouza/cachebox/mock/mock_storage"
	"github.com/romanodesouza/cachebox/storage"
)

func TestGzipData(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		level   int
		wantErr error
	}{
		{
			name:    "it should gzip at default level of compression",
			input:   []byte("repeat repeat"),
			level:   gzip.DefaultCompression,
			wantErr: nil,
		},
		{
			name:    "it should gzip at best speed level of compression",
			input:   []byte("repeat repeat"),
			level:   gzip.BestSpeed,
			wantErr: nil,
		},
		{
			name:    "it should not accept an unknown level of compression",
			input:   []byte("repeat repeat"),
			level:   -5,
			wantErr: errors.New("gzip: invalid compression level: -5"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b, err := cachebox.GzipData(tt.input, tt.level)

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}

			if err == nil && !isGzipped(b) {
				t.Errorf("%v is not gzipped", b)
			}
		})
	}
}

func TestGunzipData(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    []byte
		wantErr error
	}{
		{
			name: "it should gunzip at default level of compression",
			input: func() []byte {
				b, _ := cachebox.GzipData([]byte("repeat repeat"), gzip.DefaultCompression)
				return b
			}(),
			want:    []byte("repeat repeat"),
			wantErr: nil,
		},
		{
			name: "it should gunzip at best speed level of compression",
			input: func() []byte {
				b, _ := cachebox.GzipData([]byte("repeat repeat"), gzip.BestSpeed)
				return b
			}(),
			want:    []byte("repeat repeat"),
			wantErr: nil,
		},
		{
			name:    "it should return error for invalid gzip bytes",
			input:   nil,
			wantErr: io.EOF,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b, err := cachebox.GunzipData(tt.input)

			if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wantErr) {
				t.Errorf("got %v; want %v", err, tt.wantErr)
			}

			if err == nil && !bytes.Equal(b, tt.want) {
				t.Errorf("got %v; want %v", b, tt.want)
			}
		})
	}
}

func isGzipped(b []byte) bool {
	return len(b) >= 2 && b[0] == 31 && b[1] == 139
}

func TestCache_WithGzipCompression(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		tests := []struct {
			name    string
			ctx     context.Context
			keys    []string
			cache   func(ctrl *gomock.Controller) *cachebox.Cache
			want    [][]byte
			wantErr error
		}{
			{
				name: "it should return the bytes as is when gzip is enabled and there were stored values",
				ctx:  context.Background(),
				keys: []string{"key1"},
				cache: func(ctrl *gomock.Controller) *cachebox.Cache {
					value := []byte("not gzipped yet")
					store := mock_storage.NewMockStorage(ctrl)
					store.EXPECT().MGet(gomock.Any(), gomock.Any()).Return([][]byte{value}, nil)

					return cachebox.NewCache(store, cachebox.WithGzipCompression(gzip.DefaultCompression))
				},
				want:    [][]byte{[]byte("not gzipped yet")},
				wantErr: nil,
			},
			{
				name: "it should gunzip compressed value",
				ctx:  context.Background(),
				keys: []string{"key1"},
				cache: func(ctrl *gomock.Controller) *cachebox.Cache {
					value := []byte("gzipped now")
					gzipped, _ := cachebox.GzipData(value, gzip.DefaultCompression)
					store := mock_storage.NewMockStorage(ctrl)
					store.EXPECT().MGet(gomock.Any(), gomock.Any()).Return([][]byte{gzipped}, nil)

					return cachebox.NewCache(store, cachebox.WithGzipCompression(gzip.DefaultCompression))
				},
				want:    [][]byte{[]byte("gzipped now")},
				wantErr: nil,
			},
			{
				name: "it should return nil on miss",
				ctx:  context.Background(),
				keys: []string{"key1"},
				cache: func(ctrl *gomock.Controller) *cachebox.Cache {
					store := mock_storage.NewMockStorage(ctrl)
					store.EXPECT().MGet(gomock.Any(), gomock.Any()).Return([][]byte{nil}, nil)

					return cachebox.NewCache(store, cachebox.WithGzipCompression(gzip.DefaultCompression))
				},
				want:    [][]byte{nil},
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
	})

	t.Run("set", func(t *testing.T) {
		tests := []struct {
			name    string
			ctx     context.Context
			item    cachebox.Item
			cache   func(ctrl *gomock.Controller) *cachebox.Cache
			wantErr error
		}{
			{
				name: "it should gzip before storing the item",
				ctx:  context.Background(),
				cache: func(ctrl *gomock.Controller) *cachebox.Cache {
					value := []byte("repeat repeat")
					gzipped, _ := cachebox.GzipData(value, gzip.DefaultCompression)
					store := mock_storage.NewMockStorage(ctrl)
					store.EXPECT().Set(gomock.Any(), storage.Item{
						Key:   "key1",
						Value: gzipped,
						TTL:   time.Minute,
					}).Return(nil)

					return cachebox.NewCache(store, cachebox.WithGzipCompression(gzip.DefaultCompression))
				},
				item: cachebox.Item{
					Key:   "key1",
					Value: []byte("repeat repeat"),
					TTL:   time.Minute,
				},
				wantErr: nil,
			},
			{
				name: "it should not gzip nil value",
				ctx:  context.Background(),
				cache: func(ctrl *gomock.Controller) *cachebox.Cache {
					store := mock_storage.NewMockStorage(ctrl)
					store.EXPECT().Set(gomock.Any(), storage.Item{
						Key:   "key1",
						Value: nil,
						TTL:   time.Minute,
					}).Return(nil)

					return cachebox.NewCache(store, cachebox.WithGzipCompression(gzip.DefaultCompression))
				},
				item: cachebox.Item{
					Key:   "key1",
					Value: nil,
					TTL:   time.Minute,
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
	})
}
