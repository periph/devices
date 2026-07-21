// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import "testing"

func TestDevPerformForcedCO2Recalibration(t *testing.T) {
	cases := []writeAndReadTestCase[int16]{
		{
			name:  "success, min correction",
			model: SEN66,
			tx: []byte{
				0x67, 0x7, // Command
				0x0, 0xaf, 0x53, // Ref CO2 ppm
			},
			rx:   []byte{0x00, 0x00, 0x81},
			want: -32768,
		},
		{
			name:  "success, max correction",
			model: SEN66,
			tx: []byte{
				0x67, 0x7, // Command
				0x0, 0xaf, 0x53, // Ref CO2 ppm
			},
			rx:   []byte{0xff, 0xfe, 0x9d},
			want: 32766,
		},
		{
			name:  "calibration failed",
			model: SEN66,
			tx: []byte{
				0x67, 0x7, // Command
				0x0, 0xaf, 0x53, // Ref CO2 ppm
			},
			rx:      []byte{0xff, 0xff, 0xac},
			wantErr: true,
		},
		{
			// writeAndRead will fail because no response is set.
			name:  "read error",
			model: SEN66,
			tx: []byte{
				0x67, 0x7, // Command
				0x0, 0xaf, 0x53, // Ref CO2 ppm
			},
			wantErr:   true,
			dontPanic: true,
		},
		{
			// This fails before sending any data, so no tx or rx set.
			name:    "model without CO2 capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, func(d *Dev) (int16, error) {
		return d.PerformForcedCO2Recalibration(175)
	})
}

func TestDevPerformCO2SensorFactoryReset(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx:    []byte{0x67, 0x54},
		},
		{
			// This fails before sending any data, so no tx set.
			name:    "model without CO2 capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteTests(t, cases, (*Dev).PerformCO2SensorFactoryReset)
}

func TestDevGetCO2SensorAutomaticSelfCalibration(t *testing.T) {
	cmd := []byte{0x67, 0x11}

	cases := []writeAndReadTestCase[bool]{
		{
			name:  "enabled",
			model: SEN66,
			tx:    cmd,
			rx:    []byte{0x00, 0x01, 0xb0},
			want:  true,
		},
		{
			name:  "disabled",
			model: SEN66,
			tx:    cmd,
			rx:    []byte{0x00, 0x00, 0x81},
			want:  false,
		},
		{
			// writeAndRead will fail because no response is set.
			name:      "read error",
			model:     SEN66,
			tx:        cmd,
			wantErr:   true,
			dontPanic: true,
		},
		{
			// This fails before sending any data, so no tx or rx set.
			name:    "model without CO2 capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).GetCO2SensorAutomaticSelfCalibration)
}

func TestDevSetCO2SensorAutomaticSelfCalibration(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx: []byte{
				0x67, 0x11, // Command
				0x00, 0x01, 0xb0, // Auto self calibration boolean
			},
		},
		{
			// This fails before sending any data, so no tx set.
			name:    "model without CO2 capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteTests(t, cases, func(d *Dev) error {
		return d.SetCO2SensorAutomaticSelfCalibration(true)
	})
}

func TestDevGetAmbientPressure(t *testing.T) {
	cmd := []byte{0x67, 0x20}

	cases := []writeAndReadTestCase[uint16]{
		{
			name:  "success",
			model: SEN66,
			tx:    cmd,
			rx:    []byte{0x03, 0xf5, 0xdb},
			want:  1013,
		},
		{
			// writeAndRead will fail because no response is set.
			name:      "read error",
			model:     SEN66,
			tx:        cmd,
			wantErr:   true,
			dontPanic: true,
		},
		{
			// This fails before sending any data, so no tx or rx set.
			name:    "model without CO2 capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).GetAmbientPressure)
}

func TestDevSetAmbientPressure(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx: []byte{
				0x67, 0x20, // Command
				0x03, 0x23, 0x79, // Pressure
			},
		},
		{
			// This fails before sending any data, so no tx set.
			name:    "model without CO2 capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteTests(t, cases, func(d *Dev) error {
		return d.SetAmbientPressure(803)
	})
}

func TestDevGetSensorAltitude(t *testing.T) {
	cmd := []byte{0x67, 0x36}

	cases := []writeAndReadTestCase[uint16]{
		{
			name:  "success",
			model: SEN66,
			tx:    cmd,
			rx:    []byte{0x07, 0x7c, 0xaa},
			want:  1916,
		},
		{
			// writeAndRead will fail because no response is set.
			name:      "read error",
			model:     SEN66,
			tx:        cmd,
			wantErr:   true,
			dontPanic: true,
		},
		{
			// This fails before sending any data, so no tx or rx set.
			name:    "model without CO2 capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).GetSensorAltitude)
}

func TestDevSetSensorAltitude(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx: []byte{
				0x67, 0x36, // Command
				0x07, 0x7c, 0xaa, // Altitude
			},
		},
		{
			// This fails before sending any data, so no tx set.
			name:    "model without CO2 capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteTests(t, cases, func(d *Dev) error {
		return d.SetSensorAltitude(1916)
	})
}
