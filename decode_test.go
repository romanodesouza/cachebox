// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/romanodesouza/cachebox"
)

type MsgPackDecoder string

func (m *MsgPackDecoder) UnmarshalMsg(b []byte) ([]byte, error) {
	*m = MsgPackDecoder(string(b))
	return b, nil
}

var ErrMsgPackDecode = errors.New("msgpack decode error")

type MsgPackDecoderFailer struct{}

func (*MsgPackDecoderFailer) UnmarshalMsg(b []byte) ([]byte, error) {
	return nil, ErrMsgPackDecode
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		b       []byte
		v       interface{}
		want    []byte
		wantErr error
	}{
		{
			name:    "it should assign a byte slice when v is also a byte slice",
			b:       []byte("data"),
			v:       new([]byte),
			want:    []byte("data"),
			wantErr: nil,
		},
		{
			name:    "it should accept custom cachebox.MsgUnmarshaler interface",
			b:       []byte("data"),
			v:       new(MsgPackDecoder),
			want:    []byte("data"),
			wantErr: nil,
		},
		{
			name:    "it should passthrough cachebox.MsgUnmarshaler errors",
			b:       []byte("corrupted data"),
			v:       new(MsgPackDecoderFailer),
			want:    nil,
			wantErr: ErrMsgPackDecode,
		},
		{
			name:    "it should decode using json as fallback",
			b:       []byte(`"data"`),
			v:       new(string),
			want:    []byte("data"),
			wantErr: nil,
		},
		{
			name:    "it should return ErrUnmarshalMiss if b is nil",
			b:       nil,
			v:       []byte{},
			wantErr: cachebox.ErrUnmarshalMiss,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := cachebox.Unmarshal(tt.b, tt.v)

			var got []byte
			switch t := tt.v.(type) {
			case *[]byte:
				got = *t
			case *MsgPackDecoder:
				got = []byte(string(*t))
			case *MsgPackDecoderFailer:
				got = nil
			case *int:
				got = []byte(fmt.Sprintf("%d", *t))
			case *string:
				got = []byte(*t)
			}

			if string(tt.want) != string(got) {
				t.Errorf("want %s, got %s", string(tt.want), string(got))
			}
			if tt.wantErr != err {
				t.Errorf("wantErr %v, got %v", tt.wantErr, err)
			}
		})
	}
}
