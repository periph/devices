// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package common

import "testing"

func TestCRC8(t *testing.T) {
	var tests = []struct {
		bytes  []byte
		result byte
	}{
		{bytes: []byte{0xbe, 0xef}, result: 0x92},
		{bytes: []byte{0x01, 0xa4}, result: 0x4d},
		{bytes: []byte{0xab, 0xcd}, result: 0x6f},
	}
	for _, test := range tests {
		res := CRC8(test.bytes)
		if res != test.result {
			t.Errorf("CRC8(%#v)!=0x%d received 0x%d", test.bytes, test.result, res)
		}
	}
}
