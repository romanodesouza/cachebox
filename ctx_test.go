// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox_test

import (
	"context"
	"testing"

	"github.com/romanodesouza/cachebox"
)

func TestIsRefresh(t *testing.T) {
	tests := []struct {
		name string
		want bool
		ctx  context.Context
	}{
		{
			name: "it should return false when it doesn't have the refresh state",
			want: false,
			ctx:  context.Background(),
		},
		{
			name: "it should return true when it does have the refresh state",
			want: true,
			ctx:  cachebox.WithRefresh(context.Background()),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := cachebox.IsRefresh(tt.ctx)

			if tt.want != got {
				t.Errorf("want %t, got %t", tt.want, got)
			}
		})
	}
}

func TestIsBypass(t *testing.T) {
	tests := []struct {
		name string
		want bool
		ctx  context.Context
	}{
		{
			name: "it should return false when it doesn't have the bypass state",
			want: false,
			ctx:  context.Background(),
		},
		{
			name: "it should return true when it does have the bypass state",
			want: true,
			ctx:  cachebox.WithBypass(context.Background()),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := cachebox.IsBypass(tt.ctx)

			if tt.want != got {
				t.Errorf("want %t, got %t", tt.want, got)
			}
		})
	}
}
