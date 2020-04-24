// Copyright 2020 Romano de Souza. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cachebox

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
)

// WithGzipCompression enables gzip compression of key values.
func WithGzipCompression(level int) func(c *Cache) {
	return func(c *Cache) {
		c.storage = NewStorageWrapper(c.storage, StorageHooks{
			BeforeSet: gzipCompress(level),
			AfterMGet: gzipUncompress(),
		})
	}
}

func gzipCompress(level int) func(context.Context, Item) (Item, error) {
	return func(_ context.Context, item Item) (Item, error) {
		if item.Value == nil {
			return item, nil
		}

		var err error
		item.Value, err = gzipData(item.Value, level)

		return item, err
	}
}

func gzipUncompress() func(context.Context, string, []byte) ([]byte, error) {
	return func(_ context.Context, _ string, b []byte) ([]byte, error) {
		if b == nil {
			return b, nil
		}

		return gunzipData(b)
	}
}

func gzipData(b []byte, level int) ([]byte, error) {
	var buf bytes.Buffer

	w, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}

	_, err = w.Write(b)
	_ = w.Close()

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func gunzipData(b []byte) ([]byte, error) {
	br := bytes.NewBuffer(b)
	r, err := gzip.NewReader(br)

	switch {
	case err == gzip.ErrHeader:
		return b, nil
	case err != nil:
		return nil, err
	}

	defer r.Close() //nolint:errcheck

	bw := new(bytes.Buffer)

	if _, err := io.Copy(bw, r); err != nil {
		return nil, err
	}

	return bw.Bytes(), nil
}
