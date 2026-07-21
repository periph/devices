// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"testing"
)

func TestDevReadNumberConcentrationValues(t *testing.T) {
	cmd := []byte{0x03, 0x16}

	cases := []writeAndReadTestCase[*PMNumberConcentrations]{
		{
			name:  "all values set",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x00, 0x0c, 0xfc, // PM0.5
				0x00, 0x0f, 0xaf, // PM1.0
				0x00, 0x0f, 0xaf, // PM2.5
				0x00, 0x0f, 0xaf, // PM4.0
				0x00, 0x0f, 0xaf, // PM10.0
			},
			want: &PMNumberConcentrations{
				PM05: ptr(uint16(12)),
				PM1:  ptr(uint16(15)),
				PM25: ptr(uint16(15)),
				PM4:  ptr(uint16(15)),
				PM10: ptr(uint16(15)),
			},
		},
		{
			name:  "all values unknown",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0xff, 0xff, 0xac, // PM0.5
				0xff, 0xff, 0xac, // PM1.0
				0xff, 0xff, 0xac, // PM2.5
				0xff, 0xff, 0xac, // PM4.0
				0xff, 0xff, 0xac, // PM10.0
			},
			want: &PMNumberConcentrations{},
		},
		{
			name:  "bad crc",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x00, 0x0c, 0xfc, // PM0.5
				0x00, 0x0f, 0xaf, // PM1.0
				0x00, 0x0f, 0xff, // PM2.5 with incorrect CRC (should be 0xaf)
				0x00, 0x0f, 0xaf, // PM4.0
				0x00, 0x0f, 0xaf, // PM10.0
			},
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).ReadNumberConcentrationValues)
}
