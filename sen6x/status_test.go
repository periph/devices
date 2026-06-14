// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var sen69CAllErrsSet uint32 = 1<<21 | // FanSpeedErr
	1<<4 | // FanErr
	1<<11 | // PMSensorErr
	1<<6 | // RHTSensorErr
	1<<12 | // CO2SensorErr (SEN69C uses bit 12)
	1<<7 | // GasSensorErr
	1<<10 // HCHOSensorErr

func TestDevReadDeviceStatus(t *testing.T) {
	cmd := []byte{0xd2, 0x06}

	cases := []writeAndReadTestCase[*Status]{
		{
			name:  "all unset, SEN66",
			model: SEN66,
			tx:    cmd,
			rx:    []byte{0x00, 0x00, 0x81, 0x00, 0x00, 0x81},
			want:  &Status{},
		},
		{
			name:  "all set, SEN69C",
			model: SEN69C,
			tx:    cmd,
			rx: packWordsWithCRC(
				[]uint16{uint16(sen69CAllErrsSet >> 16), uint16(sen69CAllErrsSet)}),
			want: &Status{
				FanSpeedErr:   true,
				FanErr:        true,
				PMSensorErr:   true,
				RHTSensorErr:  true,
				CO2SensorErr:  true,
				GasSensorErr:  true,
				HCHOSensorErr: true,
			},
		},
		{
			// writeAndRead will fail because no response is set.
			name:      "read error",
			model:     SEN69C,
			tx:        cmd,
			wantErr:   true,
			dontPanic: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).ReadDeviceStatus)
}

func TestDevReadAndClearDeviceStatus(t *testing.T) {
	cmd := []byte{0xd2, 0x10}

	cases := []writeAndReadTestCase[*Status]{
		{
			name:  "all unset, SEN66",
			model: SEN66,
			tx:    cmd,
			rx:    []byte{0x00, 0x00, 0x81, 0x00, 0x00, 0x81},
			want:  &Status{},
		},
		{
			name:  "all set, SEN69C",
			model: SEN69C,
			tx:    cmd,
			rx: packWordsWithCRC(
				[]uint16{uint16(sen69CAllErrsSet >> 16), uint16(sen69CAllErrsSet)}),
			want: &Status{
				FanSpeedErr:   true,
				FanErr:        true,
				PMSensorErr:   true,
				RHTSensorErr:  true,
				CO2SensorErr:  true,
				GasSensorErr:  true,
				HCHOSensorErr: true,
			},
		},
		{
			// writeAndRead will fail because no response is set.
			name:      "read error",
			model:     SEN69C,
			tx:        cmd,
			wantErr:   true,
			dontPanic: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).ReadAndClearDeviceStatus)
}

func TestRegisterBitBool(t *testing.T) {
	cases := []struct {
		name     string
		register uint32
		bit      uint
		want     bool
	}{
		{
			name:     "bit 0 set",
			register: 0x00000001,
			bit:      0,
			want:     true,
		},
		{
			name:     "bit 0 not set",
			register: 0x00000000,
			bit:      0,
			want:     false,
		},
		{
			name:     "bit 31 set",
			register: 0x80000000,
			bit:      31,
			want:     true,
		},
		{
			name:     "bit 31 not set",
			register: 0x7FFFFFFF,
			bit:      31,
			want:     false,
		},
		{
			name:     "bit 21 set",
			register: 1 << 21,
			bit:      21,
			want:     true,
		},
		{
			name:     "bit 21 not set",
			register: 0xf3000a7d,
			bit:      21,
			want:     false,
		},
		{
			name:     "adjacent bits don't bleed",
			register: 0b1101,
			bit:      1,
			want:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := registerBitBool(tc.register, tc.bit)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStatusFromRegister(t *testing.T) {
	cases := []struct {
		name     string
		model    Model
		register uint32
		want     *Status
	}{
		{
			name:     "no errors, SEN66",
			model:    SEN66,
			register: 0x00000000,
			want:     &Status{},
		},
		{
			name:     "fan speed error, bit 21",
			model:    SEN66,
			register: 1 << 21,
			want:     &Status{FanSpeedErr: true},
		},
		{
			name:     "fan error, bit 4",
			model:    SEN66,
			register: 1 << 4,
			want:     &Status{FanErr: true},
		},
		{
			name:     "PM sensor error, bit 11",
			model:    SEN66,
			register: 1 << 11,
			want:     &Status{PMSensorErr: true},
		},
		{
			name:     "RHT sensor error, bit 6",
			model:    SEN66,
			register: 1 << 6,
			want:     &Status{RHTSensorErr: true},
		},
		{
			// SEN66 CO2 error is bit 9.
			name:     "CO2 sensor error SEN66, bit 9",
			model:    SEN66,
			register: 1 << 9,
			want:     &Status{CO2SensorErr: true},
		},
		{
			// SEN63C/SEN69C CO2 error is bit 12.
			name:     "CO2 sensor error SEN63C, bit 12",
			model:    SEN63C,
			register: 1 << 12,
			want:     &Status{CO2SensorErr: true},
		},
		{
			// SEN66 CO2 error is bit 9; bit 12 should not trigger CO2SensorErr.
			name:     "SEN66 ignores bit 12 for CO2",
			model:    SEN66,
			register: 1 << 12,
			want:     &Status{},
		},
		{
			// SEN63C CO2 error is bit 12; bit 9 should not trigger CO2SensorErr.
			name:     "SEN63C ignores bit 9 for CO2",
			model:    SEN63C,
			register: 1 << 9,
			want:     &Status{},
		},
		{
			name:     "gas sensor error SEN66, bit 7",
			model:    SEN66,
			register: 1 << 7,
			want:     &Status{GasSensorErr: true},
		},
		{
			// SEN62 has no gas sensor; bit 7 should not set GasSensorErr.
			name:     "SEN62 ignores gas sensor bit",
			model:    SEN62,
			register: 1 << 7,
			want:     &Status{},
		},
		{
			name:     "HCHO sensor error SEN68, bit 10",
			model:    SEN68,
			register: 1 << 10,
			want:     &Status{HCHOSensorErr: true},
		},
		{
			// SEN66 has no HCHO sensor; bit 10 should not set HCHOSensorErr.
			name:     "SEN66 ignores HCHO sensor bit",
			model:    SEN66,
			register: 1 << 10,
			want:     &Status{},
		},
		{
			name:     "all errors set, SEN69C",
			model:    SEN69C,
			register: sen69CAllErrsSet,
			want: &Status{
				FanSpeedErr:   true,
				FanErr:        true,
				PMSensorErr:   true,
				RHTSensorErr:  true,
				CO2SensorErr:  true,
				GasSensorErr:  true,
				HCHOSensorErr: true,
			},
		},
		{
			// SEN62 only has fan, PM, and RHT errors.
			name:  "all applicable errors set, SEN62",
			model: SEN62,
			register: 1<<21 | // FanSpeedErr
				1<<4 | // FanErr
				1<<11 | // PMSensorErr
				1<<6, // RHTSensorErr
			want: &Status{
				FanSpeedErr:  true,
				FanErr:       true,
				PMSensorErr:  true,
				RHTSensorErr: true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &Dev{model: tc.model}
			got := d.statusFromRegister(tc.register)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStatusAnyErr(t *testing.T) {
	cases := []struct {
		name   string
		status Status
		want   bool
	}{
		{
			name:   "no errors",
			status: Status{},
			want:   false,
		},
		{
			name:   "fan speed error only",
			status: Status{FanSpeedErr: true},
			want:   true,
		},
		{
			name:   "fan error only",
			status: Status{FanErr: true},
			want:   true,
		},
		{
			name:   "PM sensor error only",
			status: Status{PMSensorErr: true},
			want:   true,
		},
		{
			name:   "RHT sensor error only",
			status: Status{RHTSensorErr: true},
			want:   true,
		},
		{
			name:   "CO2 sensor error only",
			status: Status{CO2SensorErr: true},
			want:   true,
		},
		{
			name:   "gas sensor error only",
			status: Status{GasSensorErr: true},
			want:   true,
		},
		{
			name:   "HCHO sensor error only",
			status: Status{HCHOSensorErr: true},
			want:   true,
		},
		{
			name: "all errors",
			status: Status{
				FanSpeedErr:   true,
				FanErr:        true,
				PMSensorErr:   true,
				RHTSensorErr:  true,
				CO2SensorErr:  true,
				GasSensorErr:  true,
				HCHOSensorErr: true,
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.status.AnyErr()
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
