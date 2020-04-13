// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox

import "context"

type bypass int

const (
	// BypassReading represents a bypass state which skip only reading calls. It should be used to recompute values.
	BypassReading bypass = iota
	// BypassReadWriting represents a bypass state which skip all calls.
	BypassReadWriting
)

var bypassKey = struct{}{}

// WithBypass returns a new context within a bypass state.
//
// It is possible to bypass just reading or read and writing.
func WithBypass(parent context.Context, bypass bypass) context.Context {
	return context.WithValue(parent, bypassKey, bypass)
}

func bypassFromContext(ctx context.Context) bypass {
	v := ctx.Value(bypassKey)
	if v == nil {
		return bypass(-1)
	}

	return v.(bypass)
}
