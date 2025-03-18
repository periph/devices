// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package hd44780

import (
	"errors"
	"testing"
	"time"

	periphDisplay "periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/display/displaytest"
	"periph.io/x/conn/v3/i2c/i2ctest"
)

func getLCD(t *testing.T, recordingName string) (*HD44780, error) {
	bus := &i2ctest.Playback{Ops: recordingData[recordingName], DontPanic: true}
	dev, err := NewAdafruitI2CBackpack(bus, 0x20, 2, 16)

	if err != nil {
		t.Fatal(err)
	}

	return dev, err
}

const (
	testRows = 2
	testCols = 16
)

var liveDevice = false

func TestBasic(t *testing.T) {
	display, err := getLCD(t, "TestBasic")

	if err != nil {
		t.Fatal(err)
	}
	s := display.String()
	t.Log(s)
	if len(s) == 0 {
		t.Error("display.String()")
	}

	_, err = display.WriteString("1234567890")
	if err != nil {
		t.Error(err)
	}
	err = display.MoveTo(2, 2)
	if err != nil {
		t.Error(err)
	}
	_, err = display.WriteString("2345678901")
	if err != nil {
		t.Error(err)
	}
	rows := display.Rows()
	if rows != testRows {
		t.Errorf("display.Rows() expected %d, received %d", testRows, rows)
	}
	cols := display.Cols()
	if cols != testCols {
		t.Errorf("display.Cols() expected %d, received %d", testCols, cols)
	}
	if liveDevice {
		time.Sleep(5 * time.Second)
	}
	err = display.Halt()
	if err != nil {
		t.Error(err)
	}
}

func TestInterface(t *testing.T) {
	display, err := getLCD(t, "TestInterface")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = display.Halt() }()
	errs := displaytest.TestTextDisplay(display, liveDevice)
	for _, err := range errs {
		if !errors.Is(err, periphDisplay.ErrNotImplemented) {
			t.Error(err)
		}
	}
	if liveDevice {
		time.Sleep(5 * time.Second)
	}
}

func TestBacklights(t *testing.T) {
	display, err := getLCD(t, "TestBacklights")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = display.Halt()
	})

	err = display.Backlight(0)
	if err != nil {
		t.Error(err)
	}
	err = display.Backlight(0xff)
	if err != nil {
		t.Error(err)
	}
	for ix := range 3 {
		colors := make([]periphDisplay.Intensity, 3)
		colors[ix] = 0xff
		err = display.RGBBacklight(colors[0], colors[1], colors[2])
		if err != nil {
			t.Error(err)
		}
	}
}
