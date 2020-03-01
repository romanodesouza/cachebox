// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox

import "context"

type key struct{ name string }

var refreshKey = key{"refresh"}

// WithRefresh returns a context with refresh state.
// A refresh state bypasses cache reading to force updating the current cache state.
// Use this to precompute values.
func WithRefresh(ctx context.Context) context.Context {
	return context.WithValue(ctx, refreshKey, struct{}{})
}

// IsRefresh checks whether there is a refresh state.
func IsRefresh(ctx context.Context) bool {
	_, ok := ctx.Value(refreshKey).(struct{})
	return ok
}

var bypassKey = key{"bypass"}

// WithBypass returns a context with bypass state.
// A bypass state bypasses both cache reading and writing.
// Use this to skip the cache layer.
func WithBypass(ctx context.Context) context.Context {
	return context.WithValue(ctx, bypassKey, struct{}{})
}

// IsBypass checks whether there is a bypass state.
func IsBypass(ctx context.Context) bool {
	_, ok := ctx.Value(bypassKey).(struct{})
	return ok
}
