// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package aht20

import (
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/conn/v3/physic"
	"testing"
)

const byteStatusInitialized = bitInitialized | 0x10

func TestNewI2C(t *testing.T) {
	type TestCase struct {
		name string
		ops  []i2ctest.IO
	}

	testCases := []TestCase{
		{
			name: "device already initialized",
			ops: []i2ctest.IO{
				// Read status
				{Addr: deviceAddress, W: []byte{cmdStatus}, R: []byte{byteStatusInitialized}},
			},
		},
		{
			name: "device not initialized",
			ops: []i2ctest.IO{
				// Read status
				{Addr: deviceAddress, W: []byte{cmdStatus}, R: []byte{0x00}},
				// Initialize
				{Addr: deviceAddress, W: argsInitialize},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bus := i2ctest.Playback{Ops: tc.ops}
			if dev, err := NewI2C(&bus, nil); err != nil {
				t.Fatal(err)
			} else if dev == nil {
				t.Fatal("expected device")
			}
		})
	}
}

func TestDev_IsInitialized_true(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Read status
			{Addr: deviceAddress, W: []byte{cmdStatus}, R: []byte{byteStatusInitialized}},
		},
	}
	dev := Dev{d: &i2c.Dev{Bus: &bus, Addr: deviceAddress}, opts: DefaultOpts}
	if err, initialized := dev.IsInitialized(); err != nil {
		t.Fatal(err)
	} else if !initialized {
		t.Fatal("expected initialized")
	}
}

func TestDev_IsInitialized_false(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Read status
			{Addr: deviceAddress, W: []byte{cmdStatus}, R: []byte{0x00}},
		},
	}
	dev := Dev{d: &i2c.Dev{Bus: &bus, Addr: deviceAddress}, opts: DefaultOpts}
	if err, initialized := dev.IsInitialized(); err != nil {
		t.Fatal(err)
	} else if initialized {
		t.Fatal("expected not initialized")
	}
}

func TestDev_Initialize(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Initialize
			{Addr: deviceAddress, W: argsInitialize},
		},
	}
	dev := Dev{d: &i2c.Dev{Bus: &bus, Addr: deviceAddress}, opts: DefaultOpts}
	if err := dev.Initialize(); err != nil {
		t.Fatal(err)
	}
}

func TestDev_Sense_successful(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Trigger measurement
			{Addr: deviceAddress, W: argsMeasure},
			// Read measurement
			{Addr: deviceAddress, R: []byte{byteStatusInitialized, 0x75, 0x52, 0x05, 0x8E, 0x40, 0x7F}},
		},
	}
	dev := Dev{d: &i2c.Dev{Bus: &bus, Addr: deviceAddress}, opts: DefaultOpts}
	e := physic.Env{}
	if err := dev.Sense(&e); err != nil {
		t.Fatal(err)
	}
	if expected := 19445800781*physic.NanoKelvin + physic.ZeroCelsius; e.Temperature != expected {
		t.Fatalf("temperature %s(%d) != %s(%d)", expected, expected, e.Temperature, e.Temperature)
	}
	if expected := 4582824 * physic.TenthMicroRH; e.Humidity != expected {
		t.Fatalf("humidity %s(%d) != %s(%d)", expected, expected, e.Humidity, e.Humidity)
	}
	if expected := 0 * physic.Pascal; e.Pressure != expected {
		t.Fatalf("pressure %s(%d) != %s(%d)", expected, expected, e.Pressure, e.Pressure)
	}
	if err := bus.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDev_Sense_error(t *testing.T) {
	type TestCase struct {
		name  string
		data  []byte
		error error
	}

	testCases := []TestCase{
		{
			name:  "data corrupt",
			data:  []byte{byteStatusInitialized, 0x75, 0x52, 0x05, 0x8E, 0x40, 0x7E},
			error: &DataCorruptionError{0x7F, 0x7E},
		},
		{
			name:  "not initialized",
			data:  []byte{0x00, 0x75, 0x52, 0x05, 0x8E, 0x40, 0x20},
			error: &NotInitializedError{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bus := i2ctest.Playback{
				Ops: []i2ctest.IO{
					// Trigger measurement
					{Addr: deviceAddress, W: argsMeasure},
					// Read measurement
					{Addr: deviceAddress, R: tc.data},
				},
			}
			dev := Dev{d: &i2c.Dev{Bus: &bus, Addr: deviceAddress}, opts: DefaultOpts}
			e := physic.Env{}
			if err := dev.Sense(&e); err == nil {
				t.Fatal("expected error")
			} else if err.Error() != tc.error.Error() {
				t.Fatalf("expected error %s, got %s", tc.error, err)
			}
		})
	}
}

func TestDev_SoftReset(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Soft reset
			{Addr: deviceAddress, W: []byte{cmdSoftReset}},
		},
	}
	dev := Dev{d: &i2c.Dev{Bus: &bus, Addr: deviceAddress}, opts: DefaultOpts}
	if err := dev.SoftReset(); err != nil {
		t.Fatal(err)
	}
}
