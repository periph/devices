// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package tic

import (
	"errors"
	"testing"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/conn/v3/physic"
)

func TestNewI2C(t *testing.T) {
	for _, test := range []struct {
		name      string
		variant   Variant
		ops       []i2ctest.IO
		expectErr bool
	}{
		{
			name:    "success",
			variant: TicT500,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x49}},
				{Addr: I2CAddr, R: []byte{0x00}},
			},
			expectErr: false,
		},
		{
			name:      "invalid variant",
			variant:   Variant("periph"),
			expectErr: true,
		},
		{
			name:    "connection failure",
			variant: TicT500,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x49}},
			},
			expectErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			_, err := NewI2C(&b, test.variant, I2CAddr)
			if test.expectErr && err == nil {
				t.Fatalf("expected error, got: %v", err)
			} else if !test.expectErr && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetTargetPosition(t *testing.T) {
	for _, test := range []struct {
		name      string
		ops       []i2ctest.IO
		want      int32
		expectErr error
	}{
		{
			name: "success",
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x09}},
				{Addr: I2CAddr, R: []byte{byte(PlanningModeTargetPosition)}},

				{Addr: I2CAddr, W: []byte{0xA1, 0x0A}},
				{Addr: I2CAddr, R: []byte{0xEE, 0xDB, 0xEA, 0x0D}},
			},
			want:      0xDEADBEE,
			expectErr: nil,
		},
		{
			name: "incorrect planning mode",
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x09}},
				{Addr: I2CAddr, R: []byte{byte(PlanningModeTargetVelocity)}},

				{Addr: I2CAddr, W: []byte{0xA1, 0x0A}},
				{Addr: I2CAddr, R: []byte{0xEE, 0xDB, 0xEA, 0x0D}},
			},
			expectErr: ErrIncorrectPlanningMode,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			dev := Dev{
				c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
				variant: TicT825,
			}

			got, err := dev.GetTargetPosition()
			if !errors.Is(err, test.expectErr) {
				t.Fatalf("expected error: %v, got: %v", test.expectErr, err)
			}
			if got != test.want {
				t.Fatalf("wanted: %d, got: %d", test.want, got)
			}
		})
	}
}

func TestSetStepMode(t *testing.T) {
	for _, test := range []struct {
		name      string
		variant   Variant
		mode      StepMode
		ops       []i2ctest.IO
		expectErr error
	}{
		{
			name:    "success",
			variant: TicT825,
			mode:    StepModeFull,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0x94, 0x00}},
			},
			expectErr: nil,
		},
		{
			name:    "invalid step mode",
			variant: TicT825,
			mode:    StepMode(0xFF),
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0x94, 0xFF}},
			},
			expectErr: ErrInvalidSetting,
		},
		{
			name:    "unsupported variant",
			variant: TicT500,
			mode:    StepModeMicrostep256,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0x94, 0x09}},
			},
			expectErr: ErrUnsupportedVariant,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			dev := Dev{
				c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
				variant: test.variant,
			}

			err := dev.SetStepMode(test.mode)
			if !errors.Is(err, test.expectErr) {
				t.Fatalf("expected error: %v, got: %v", test.expectErr, err)
			}
		})
	}
}

func TestSetDecayMode(t *testing.T) {
	for _, test := range []struct {
		name      string
		variant   Variant
		mode      DecayMode
		ops       []i2ctest.IO
		expectErr error
	}{
		{
			name:    "success",
			variant: TicT825,
			mode:    DecayModeMixed,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0x92, 0x00}},
			},
			expectErr: nil,
		},
		{
			name:    "invalid decay mode",
			variant: TicT825,
			mode:    DecayMode(0xFF),
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0x94, 0xFF}},
			},
			expectErr: ErrInvalidSetting,
		},
		{
			name:    "unsupported decay mode",
			variant: TicT825,
			mode:    DecayModeMixed75,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0x94, 0x04}},
			},
			expectErr: ErrUnsupportedVariant,
		},
		{
			name:    "unsupported variant",
			variant: TicT500,
			mode:    DecayModeMixed,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0x94, 0x00}},
			},
			expectErr: ErrUnsupportedVariant,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			dev := Dev{
				c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
				variant: test.variant,
			}

			err := dev.SetDecayMode(test.mode)
			if !errors.Is(err, test.expectErr) {
				t.Fatalf("expected error: %v, got: %v", test.expectErr, err)
			}
		})
	}
}

func TestGetCurrentLimit(t *testing.T) {
	for _, test := range []struct {
		name    string
		variant Variant
		ops     []i2ctest.IO
		want    physic.ElectricCurrent
	}{
		{
			name:    "T500 success",
			variant: TicT500,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x4A}},
				{Addr: I2CAddr, R: []byte{0x09}},
			},
			want: 1092 * physic.MilliAmpere,
		},
		{
			name:    "T249 success",
			variant: TicT249,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x4A}},
				{Addr: I2CAddr, R: []byte{0x0A}},
			},
			want: 400 * physic.MilliAmpere,
		},
		{
			name:    "36v4 success",
			variant: Tic36v4,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x4A}},
				{Addr: I2CAddr, R: []byte{0x0A}},
			},
			want: 716 * physic.MilliAmpere,
		},
		{
			name:    "T825 success",
			variant: TicT825,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x4A}},
				{Addr: I2CAddr, R: []byte{0x0A}},
			},
			want: 320 * physic.MilliAmpere,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			dev := Dev{
				c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
				variant: test.variant,
			}

			got, err := dev.GetCurrentLimit()
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("wanted: %v, got: %v", test.want, got)
			}
		})
	}
}

func TestSetCurrentLimit(t *testing.T) {
	for _, test := range []struct {
		name    string
		variant Variant
		limit   physic.ElectricCurrent
		ops     []i2ctest.IO
		want    int32
	}{
		{
			name:    "T500 success",
			variant: TicT500,
			limit:   500 * physic.MilliAmpere,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0x91, 0x04}},
			},
		},
		{
			name:    "36v4 success",
			variant: Tic36v4,
			limit:   500 * physic.MilliAmpere,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0x91, 0x06}},
			},
		},
		{
			name:    "36v4 lower limit",
			variant: Tic36v4,
			limit:   10 * physic.NanoAmpere,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0x91, 0x00}},
			},
		},
		{
			name:    "36v4 upper limit",
			variant: Tic36v4,
			limit:   10 * physic.Ampere,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0x91, 0x7F}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			dev := Dev{
				c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
				variant: test.variant,
			}

			err := dev.SetCurrentLimit(test.limit)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetEnergized(t *testing.T) {
	for _, test := range []struct {
		name string
		ops  []i2ctest.IO
		want bool
	}{
		{
			name: "device energized",
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x01}},
				{Addr: I2CAddr, R: []byte{0x01}},
			},
			want: true,
		},
		{
			name: "device not energized",
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x01}},
				{Addr: I2CAddr, R: []byte{0x00}},
			},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			dev := Dev{
				c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
				variant: Tic36v4,
			}

			got, err := dev.IsEnergized()
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("wanted: %t, got: %t", test.want, got)
			}
		})
	}
}

func TestGetErrorsOccurred(t *testing.T) {
	const want uint32 = 0xAABBCCDD

	b := i2ctest.Playback{
		Ops: []i2ctest.IO{
			{Addr: I2CAddr, W: []byte{0xA2, 0x04}},
			{Addr: I2CAddr, R: []byte{0xDD, 0xCC, 0xBB, 0xAA}},
		},
		DontPanic: true,
	}
	defer b.Close()

	dev := Dev{
		c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
		variant: Tic36v4,
	}

	got, err := dev.GetErrorsOccurred()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("wanted: %d, got: %d", want, got)
	}
}

func TestGetPinState(t *testing.T) {
	for _, test := range []struct {
		name      string
		pin       Pin
		ops       []i2ctest.IO
		want      PinState
		expectErr error
	}{
		{
			name: "success",
			pin:  PinRX,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x48}},
				{Addr: I2CAddr, R: []byte{0x80}},
			},
			want:      PinStateOutputLow,
			expectErr: nil,
		},
		{
			name: "invalid pin",
			pin:  Pin(0xFF),
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0x48}},
			},
			expectErr: ErrInvalidSetting,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			dev := Dev{
				c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
				variant: Tic36v4,
			}

			got, err := dev.GetPinState(test.pin)
			if !errors.Is(err, test.expectErr) {
				t.Fatalf("expected error: %v, got: %v", test.expectErr, err)
			}
			if got != test.want {
				t.Fatalf("wanted: %d, got: %d", test.want, got)
			}
		})
	}
}
