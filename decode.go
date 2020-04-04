// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox

import (
	"encoding/json"
	"errors"
)

// MsgUnmarshaler is the interface that unmarshals an item in the MessagePack format.
type MsgUnmarshaler interface {
	UnmarshalMsg([]byte) ([]byte, error)
}

// ErrUnmarshalMiss represents an error when trying to unmarshal a miss.
var ErrUnmarshalMiss = errors.New("cachebox: can't unmarshal miss")

// Unmarshal decodes a byte slice.
func Unmarshal(b []byte, v interface{}) error {
	// Can't decode miss.
	if b == nil {
		return ErrUnmarshalMiss
	}

	// If it's a []byte, just assign it
	if target, ok := v.(*[]byte); ok {
		*target = b
		return nil
	}

	// Custom MsgUnmarshaler interface
	if i, ok := v.(MsgUnmarshaler); ok {
		_, err := i.UnmarshalMsg(b)
		return err
	}

	// Fallbacks to JSON unmarshaling
	return json.Unmarshal(b, v)
}
