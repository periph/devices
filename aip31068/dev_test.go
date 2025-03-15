// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package aip31068_test

import (
	"errors"
	"testing"
	"time"

	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/display/displaytest"
	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/devices/v3/aip31068"
	"periph.io/x/devices/v3/waveshare1602"
)

var pause time.Duration = 0
var liveDevice bool

func getDev(recordingName string) (*aip31068.Dev, error) {
	bus := &i2ctest.Playback{Ops: recordingData[recordingName], DontPanic: true}
	dev, err := waveshare1602.New(bus, waveshare1602.LCD1602RGBBacklight, 2, 16)
	return dev, err
}

func TestBasic(t *testing.T) {
	dev, err := getDev("TestBasic")
	if err != nil {
		t.Fatal(err)
	}
	s := dev.String()
	if len(s) == 0 {
		t.Error("error on String()")
	}
	t.Log(s)
	t.Cleanup(func() {
		_ = dev.Halt()
	})

	err = dev.Clear()

	if err != nil {
		t.Error(err)
	}
	err = dev.Backlight(0xff)
	if err != nil {
		t.Error(err)
	}
	n, err := dev.WriteString("aip31068")
	if err != nil {
		t.Error(err)
	}
	if n != 8 {
		t.Error("expected 8 bytes written")
	}
	time.Sleep(5 * pause)
}

func TestComplete(t *testing.T) {
	dev, err := getDev("TestComplete")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = dev.Halt()
	})
	testErrs := displaytest.TestTextDisplay(dev, liveDevice)
	for _, err := range testErrs {
		if !errors.Is(err, display.ErrNotImplemented) {
			t.Error(err)
		}
	}
}

func TestBacklights(t *testing.T) {
	dev, err := getDev("TestBacklights")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = dev.Halt()
	})

	for ix := range 3 {
		leds := make([]display.Intensity, 3)
		leds[ix] = 0xff
		err = dev.RGBBacklight(leds[0], leds[1], leds[2])
		if err != nil {
			t.Error(err)
		}
		time.Sleep(pause)
	}
	err = dev.Backlight(0)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(pause)
	err = dev.Backlight(1)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(pause)
}
