// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package pca9633

import (
	"testing"
	"time"

	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/i2c/i2ctest"
)

var recordingData = map[string][]i2ctest.IO{
	"TestBasic": {
		{Addr: 0x60, W: []uint8{0x0, 0x81}},
		{Addr: 0x60, W: []uint8{0x1, 0x5}},
		{Addr: 0x60, W: []uint8{0x8, 0x1}},
		{Addr: 0x60, W: []uint8{0x8, 0x4}},
		{Addr: 0x60, W: []uint8{0x8, 0x10}},
		{Addr: 0x60, W: []uint8{0x8, 0x40}},
		{Addr: 0x60, W: []uint8{0x1, 0x15}},
		{Addr: 0x60, W: []uint8{0x1, 0x5}},
		{Addr: 0x60, W: []uint8{0x6, 0x80}},
		{Addr: 0x60, W: []uint8{0x8, 0x3f}},
		{Addr: 0x60, W: []uint8{0x2, 0xff}},
		{Addr: 0x60, W: []uint8{0x3, 0xff}},
		{Addr: 0x60, W: []uint8{0x4, 0xff}},
		{Addr: 0x60, W: []uint8{0x7, 0x30}},
		{Addr: 0x60, W: []uint8{0x1, 0x25}},
		{Addr: 0x60, W: []uint8{0x6, 0x80}},
		{Addr: 0x60, W: []uint8{0x8, 0x0}}},
}

func TestBasic(t *testing.T) {
	bus := &i2ctest.Playback{Ops: recordingData["TestBasic"]}
	dev, err := New(bus, 0x60, STRUCT_OPENDRAIN)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 4 {
		values := make([]display.Intensity, 4)
		values[i] = 0xff
		err = dev.Out(values...)
		if err != nil {
			t.Error(err)
		}
	}

	err = dev.SetInvert(true)
	if err != nil {
		t.Error(err)
	}
	err = dev.SetInvert(false)
	if err != nil {
		t.Error(err)
	}

	err = dev.SetGroupPWMBlink(0x80, 0)
	if err != nil {
		t.Error(err)
	}
	err = dev.SetModes(MODE_PWM_PLUS_GROUP, MODE_PWM_PLUS_GROUP, MODE_PWM_PLUS_GROUP, MODE_FULL_OFF)
	if err != nil {
		t.Error(err)
	}

	err = dev.Out(0xff, 0xff, 0xff)
	if err != nil {
		t.Error(err)
	}

	err = dev.SetGroupPWMBlink(0x80, 2*time.Second)
	if err != nil {
		t.Error(err)
	}

	s := dev.String()
	if len(s) == 0 {
		t.Error("empty string")
	}

	err = dev.Halt()
	if err != nil {
		t.Error(err)
	}
}
