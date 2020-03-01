// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox

import "encoding/json"

// MsgMarshaler is the interface that marshals an item in the MessagePack format.
type MsgMarshaler interface {
	MarshalMsg(b []byte) ([]byte, error)
}

// Marshal encodes an item.
func Marshal(v interface{}) ([]byte, error) {
	// If it's already a byte slice, return it
	if b, ok := v.([]byte); ok {
		return b, nil
	}

	// Custom MsgMarhsaler interface
	if i, ok := v.(MsgMarshaler); ok {
		return i.MarshalMsg(nil)
	}

	// Fallbacks to JSON marshaling
	return json.Marshal(v)
}
