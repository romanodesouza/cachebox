// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/romanodesouza/cachebox"
)

type MsgPackEncoder struct{}

func (*MsgPackEncoder) MarshalMsg(b []byte) ([]byte, error) {
	return []byte("msgpack"), nil
}

var (
	ErrMsgPackEncode = errors.New("msgpack encode error")
)

type MsgPackEncoderFailer struct{}

func (*MsgPackEncoderFailer) MarshalMsg(b []byte) ([]byte, error) {
	return nil, ErrMsgPackEncode
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name    string
		v       interface{}
		want    []byte
		wantErr error
	}{
		{
			name:    "it should passthrough already serialized data",
			v:       []byte("data"),
			want:    []byte("data"),
			wantErr: nil,
		},
		{
			name:    "it should accept custom cachebox.MsgMarshaler interface",
			v:       new(MsgPackEncoder),
			want:    []byte("msgpack"),
			wantErr: nil,
		},
		{
			name:    "it should passthrough custom cachebox.MsgMarshaler errors",
			v:       new(MsgPackEncoderFailer),
			want:    nil,
			wantErr: ErrMsgPackEncode,
		},
		{
			name:    "it should encode any other data type using json as fallback",
			v:       "encode me",
			want:    []byte(`"encode me"`),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := cachebox.Marshal(tt.v)

			if diff := cmp.Diff(string(tt.want), string(got)); diff != "" {
				t.Errorf("unexpected result(-want +got):\n%s", diff)
			}
			if tt.wantErr != gotErr {
				t.Errorf("want %v, got %v", tt.wantErr, gotErr)
			}
		})
	}
}
