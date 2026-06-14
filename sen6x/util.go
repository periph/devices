// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"bytes"
)

// clen returns the index of the first NULL byte in n or len(n) if n contains no NULL byte.
func clen(n []byte) int {
	if i := bytes.IndexByte(n, 0); i != -1 {
		return i
	}

	return len(n)
}

func ptr[T uint16 | int16 | float32](n T) *T {
	return &n
}
