// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import "testing"

func TestClen(t *testing.T) {
	cases := []struct {
		name string
		b    []byte
		want int
	}{
		{
			name: "nil",
			b:    nil,
			want: 0,
		},
		{
			name: "empty",
			b:    []byte{},
			want: 0,
		},
		{
			name: "null byte at end",
			b:    []byte{0xaa, 0xbb, 0xcc, 0x0},
			want: 3,
		},
		{
			name: "null byte in middle",
			b:    []byte{0xaa, 0xbb, 0x00, 0xcc},
			want: 2,
		},
		{
			name: "null byte at start",
			b:    []byte{0x00, 0xaa, 0xbb, 0xcc},
			want: 0,
		},
		{
			name: "multiple null bytes",
			b:    []byte{0xaa, 0x00, 0x00, 0xbb},
			want: 1,
		},
		{
			name: "no null byte",
			b:    []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee},
			want: 5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := clen(tc.b)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
