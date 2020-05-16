// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox

import "time"

var GzipData = gzipData
var GunzipData = gunzipData

var NewStorageWrapper = newStorageWrapper

// Not an export but a little trick to not expose the now var.
func SetNowFn(fn func() time.Time) {
	now = fn
}
