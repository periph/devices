// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"testing"
)

func TestDevSetTemperatureOffsetParameters(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx: []byte{
				0x60, 0xb2, // Command
				0x01, 0x90, 0x4c, // Offset
				0x75, 0x30, 0x08, // Slope
				0x00, 0x0a, 0x5a, // Time constant
				0x00, 0x01, 0xb0, // Slot
			},
		},
	}

	params := TemperatureOffsetParameters{
		Offset:       2,
		Slope:        3,
		TimeConstant: 10,
		Slot:         1,
	}

	runWriteTests(t, cases, func(d *Dev) error {
		return d.SetTemperatureOffsetParameters(params)
	})
}

func TestDevSetTemperatureAccelerationParameters(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx: []byte{
				0x61, 0x00, // Command
				0x00, 0x0a, 0x5a, // K
				0x00, 0x14, 0x06, // P
				0x00, 0x1e, 0xdd, // T1
				0x00, 0x28, 0xbe, // T2
			},
		},
	}

	params := TemperatureAccelerationParameters{
		K:  1,
		P:  2,
		T1: 3,
		T2: 4,
	}

	runWriteTests(t, cases, func(d *Dev) error {
		return d.SetTemperatureAccelerationParameters(params)
	})
}

func TestDevActivateSHTHeater(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx:    []byte{0x67, 0x65},
		},
	}

	runWriteTests(t, cases, (*Dev).ActivateSHTHeater)
}

func TestDevGetSHTHeaterMeasurements(t *testing.T) {
	cmd := []byte{0x67, 0x90}

	cases := []writeAndReadTestCase[*SHTHeaterMeasurements]{
		{
			name:  "all values set",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x09, 0xf0, 0xc0, // RH
				0x33, 0xbb, 0x4b, // Temp
			},
			want: &SHTHeaterMeasurements{
				RH:   ptr(float32(25.44)),
				Temp: ptr(float32(66.215)),
			},
		},
		{
			name:  "all values unknown",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x7f, 0xff, 0x8f, // RH
				0x7f, 0xff, 0x8f, // Temp
			},
			want: &SHTHeaterMeasurements{},
		},
		{
			name:  "bad crc",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x09, 0xf0, 0xff, // RH with incorrect CRC (should be 0xc0)
				0x33, 0xbb, 0x4b, // Temp
			},
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).GetSHTHeaterMeasurements)
}
