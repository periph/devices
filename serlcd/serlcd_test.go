// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package serlcd

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/display/displaytest"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/host/v3"
)

var liveDevice bool = false
var bus i2c.Bus
var eepromTests bool = false
var sleepDuration time.Duration = time.Millisecond

// Playback for interface tests.
var pbInterface = []i2ctest.IO{
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53, 0x70, 0x61, 0x72, 0x6b, 0x46, 0x75, 0x6e, 0x20, 0x53, 0x65, 0x72, 0x4c, 0x43, 0x44, 0x20, 0x32, 0x30, 0x78, 0x34, 0x20, 0x44, 0x69, 0x73, 0x70, 0x6c, 0x61, 0x79, 0x20, 0x2d, 0x20, 0x70}},
	{Addr: DefaultI2CAddress, W: []uint8{0x65, 0x72, 0x69, 0x70, 0x68, 0x2e, 0x69, 0x6f, 0x2e, 0x43, 0x6f, 0x6e, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x6e}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x41, 0x75, 0x74, 0x6f, 0x20, 0x53, 0x63, 0x72, 0x6f, 0x6c, 0x6c, 0x20, 0x54, 0x65, 0x73, 0x74}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x80}},
	{Addr: DefaultI2CAddress, W: []uint8{0x41}},
	{Addr: DefaultI2CAddress, W: []uint8{0x42}},
	{Addr: DefaultI2CAddress, W: []uint8{0x43}},
	{Addr: DefaultI2CAddress, W: []uint8{0x44}},
	{Addr: DefaultI2CAddress, W: []uint8{0x45}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x47}},
	{Addr: DefaultI2CAddress, W: []uint8{0x48}},
	{Addr: DefaultI2CAddress, W: []uint8{0x49}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4a}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4c}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4e}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4f}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x51}},
	{Addr: DefaultI2CAddress, W: []uint8{0x52}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53}},
	{Addr: DefaultI2CAddress, W: []uint8{0x54}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc0}},
	{Addr: DefaultI2CAddress, W: []uint8{0x41}},
	{Addr: DefaultI2CAddress, W: []uint8{0x42}},
	{Addr: DefaultI2CAddress, W: []uint8{0x43}},
	{Addr: DefaultI2CAddress, W: []uint8{0x44}},
	{Addr: DefaultI2CAddress, W: []uint8{0x45}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x47}},
	{Addr: DefaultI2CAddress, W: []uint8{0x48}},
	{Addr: DefaultI2CAddress, W: []uint8{0x49}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4a}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4c}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4e}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4f}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x51}},
	{Addr: DefaultI2CAddress, W: []uint8{0x52}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53}},
	{Addr: DefaultI2CAddress, W: []uint8{0x54}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x94}},
	{Addr: DefaultI2CAddress, W: []uint8{0x41}},
	{Addr: DefaultI2CAddress, W: []uint8{0x42}},
	{Addr: DefaultI2CAddress, W: []uint8{0x43}},
	{Addr: DefaultI2CAddress, W: []uint8{0x44}},
	{Addr: DefaultI2CAddress, W: []uint8{0x45}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x47}},
	{Addr: DefaultI2CAddress, W: []uint8{0x48}},
	{Addr: DefaultI2CAddress, W: []uint8{0x49}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4a}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4c}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4e}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4f}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x51}},
	{Addr: DefaultI2CAddress, W: []uint8{0x52}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53}},
	{Addr: DefaultI2CAddress, W: []uint8{0x54}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xd4}},
	{Addr: DefaultI2CAddress, W: []uint8{0x41}},
	{Addr: DefaultI2CAddress, W: []uint8{0x42}},
	{Addr: DefaultI2CAddress, W: []uint8{0x43}},
	{Addr: DefaultI2CAddress, W: []uint8{0x44}},
	{Addr: DefaultI2CAddress, W: []uint8{0x45}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x47}},
	{Addr: DefaultI2CAddress, W: []uint8{0x48}},
	{Addr: DefaultI2CAddress, W: []uint8{0x49}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4a}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4c}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4e}},
	{Addr: DefaultI2CAddress, W: []uint8{0x4f}},
	{Addr: DefaultI2CAddress, W: []uint8{0x20}},
	{Addr: DefaultI2CAddress, W: []uint8{0x51}},
	{Addr: DefaultI2CAddress, W: []uint8{0x52}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53}},
	{Addr: DefaultI2CAddress, W: []uint8{0x54}},
	{Addr: DefaultI2CAddress, W: []uint8{0x61, 0x75, 0x74, 0x6f, 0x20, 0x73, 0x63, 0x72, 0x6f, 0x6c, 0x6c, 0x20, 0x68, 0x61, 0x70, 0x70, 0x65, 0x6e}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x41, 0x62, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x65, 0x20, 0x50, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x69, 0x6e, 0x67}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x80}},
	{Addr: DefaultI2CAddress, W: []uint8{0x28, 0x30, 0x2c, 0x30, 0x29}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc1}},
	{Addr: DefaultI2CAddress, W: []uint8{0x28, 0x31, 0x2c, 0x31, 0x29}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x96}},
	{Addr: DefaultI2CAddress, W: []uint8{0x28, 0x32, 0x2c, 0x32, 0x29}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xd7}},
	{Addr: DefaultI2CAddress, W: []uint8{0x28, 0x33, 0x2c, 0x33, 0x29}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc0}},
	{Addr: DefaultI2CAddress, W: []uint8{0x43, 0x75, 0x72, 0x73, 0x6f, 0x72, 0x3a, 0x20, 0x4f, 0x66, 0x66}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc0}},
	{Addr: DefaultI2CAddress, W: []uint8{0x43, 0x75, 0x72, 0x73, 0x6f, 0x72, 0x3a, 0x20, 0x55, 0x6e, 0x64, 0x65, 0x72, 0x6c, 0x69, 0x6e, 0x65}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xe}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc0}},
	{Addr: DefaultI2CAddress, W: []uint8{0x43, 0x75, 0x72, 0x73, 0x6f, 0x72, 0x3a, 0x20, 0x42, 0x6c, 0x6f, 0x63, 0x6b}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xd}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc0}},
	{Addr: DefaultI2CAddress, W: []uint8{0x43, 0x75, 0x72, 0x73, 0x6f, 0x72, 0x3a, 0x20, 0x42, 0x6c, 0x69, 0x6e, 0x6b}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xd}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x54, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67, 0x20, 0x3e}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x14}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x14}},
	{Addr: DefaultI2CAddress, W: []uint8{0x30}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x10}},
	{Addr: DefaultI2CAddress, W: []uint8{0x31}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x10}},
	{Addr: DefaultI2CAddress, W: []uint8{0x32}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x10}},
	{Addr: DefaultI2CAddress, W: []uint8{0x33}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x10}},
	{Addr: DefaultI2CAddress, W: []uint8{0x34}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x10}},
	{Addr: DefaultI2CAddress, W: []uint8{0x35}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x10}},
	{Addr: DefaultI2CAddress, W: []uint8{0x36}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x10}},
	{Addr: DefaultI2CAddress, W: []uint8{0x37}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x10}},
	{Addr: DefaultI2CAddress, W: []uint8{0x38}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x10}},
	{Addr: DefaultI2CAddress, W: []uint8{0x39}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x10}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53, 0x65, 0x74, 0x20, 0x64, 0x65, 0x76, 0x20, 0x6f, 0x66, 0x66}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0x8}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53, 0x65, 0x74, 0x20, 0x64, 0x65, 0x76, 0x20, 0x6f, 0x6e}}}

var pbBacklight = []i2ctest.IO{
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53, 0x65, 0x74, 0x20, 0x42, 0x61, 0x63, 0x6b, 0x6c, 0x69, 0x67, 0x68, 0x74, 0x20, 0x4f, 0x66, 0x66}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2b, 0x0, 0x0, 0x0}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2b, 0xff, 0xff, 0xff}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53, 0x65, 0x74, 0x20, 0x42, 0x61, 0x63, 0x6b, 0x6c, 0x69, 0x67, 0x68, 0x74, 0x20, 0x4f, 0x6e}}}

var pbRGBBacklight = []i2ctest.IO{
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53, 0x65, 0x74, 0x20, 0x42, 0x61, 0x63, 0x6b, 0x6c, 0x69, 0x67, 0x68, 0x74, 0x20, 0x5b, 0x5d, 0x64, 0x69, 0x73, 0x70, 0x6c, 0x61, 0x79, 0x2e, 0x49, 0x6e, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x74}},
	{Addr: DefaultI2CAddress, W: []uint8{0x79, 0x7b, 0x32, 0x35, 0x35, 0x2c, 0x20, 0x30, 0x2c, 0x20, 0x30, 0x7d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2b, 0xff, 0x0, 0x0}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53, 0x65, 0x74, 0x20, 0x42, 0x61, 0x63, 0x6b, 0x6c, 0x69, 0x67, 0x68, 0x74, 0x20, 0x5b, 0x5d, 0x64, 0x69, 0x73, 0x70, 0x6c, 0x61, 0x79, 0x2e, 0x49, 0x6e, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x74}},
	{Addr: DefaultI2CAddress, W: []uint8{0x79, 0x7b, 0x30, 0x2c, 0x20, 0x32, 0x35, 0x35, 0x2c, 0x20, 0x30, 0x7d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2b, 0x0, 0xff, 0x0}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53, 0x65, 0x74, 0x20, 0x42, 0x61, 0x63, 0x6b, 0x6c, 0x69, 0x67, 0x68, 0x74, 0x20, 0x5b, 0x5d, 0x64, 0x69, 0x73, 0x70, 0x6c, 0x61, 0x79, 0x2e, 0x49, 0x6e, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x74}},
	{Addr: DefaultI2CAddress, W: []uint8{0x79, 0x7b, 0x30, 0x2c, 0x20, 0x30, 0x2c, 0x20, 0x32, 0x35, 0x35, 0x7d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2b, 0x0, 0x0, 0xff}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2b, 0xff, 0xff, 0xff}}}

var pbContrast = []i2ctest.IO{
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53, 0x65, 0x74, 0x20, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x73, 0x74, 0x20, 0x35}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x18, 0x5}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc0}},
	{Addr: DefaultI2CAddress, W: []uint8{0x53, 0x65, 0x74, 0x20, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x73, 0x74, 0x20, 0x30, 0x78, 0x38, 0x30}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x18, 0x80}},
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x18, 0x28}}}

var pbHalt = []i2ctest.IO{
	{Addr: DefaultI2CAddress, W: []uint8{0x7c, 0x2d}},
	{Addr: DefaultI2CAddress, W: []uint8{0xfe, 0xc}}}

func init() {
	var err error
	// If the environment variable is set, assume we have a live device on
	// the default i2c bus and use it for testing. If the variable is not
	// present, then use the playback/read values.
	liveDevice = os.Getenv("SERLCD") != ""
	// If EEPROM is set, then tests that write to EEPROM (display intensity /
	// contrast are tested if it's a live test.
	eepromTests = os.Getenv("EEPROM") != ""
	if _, err = host.Init(); err != nil {
		fmt.Println(err)
	}

	if liveDevice {
		sleepDuration = 5 * time.Second
		bus, err = i2creg.Open("")
		if err != nil {
			fmt.Println(err)
		}
		// Add the recorder to dump the data stream when we're using a live device.
		bus = &i2ctest.Record{Bus: bus}
	} else {
		bus = &i2ctest.Playback{DontPanic: true}
	}

}

// getDev returns a SerLCD device for testing connected to either a live
// bus, or a playback bus. playbackOps is a slice of i2ctest.IO
// operations to be used for playback mode. Ignored for live device
// testing.
func getDev(t *testing.T, playbackOps ...[]i2ctest.IO) (*Dev, error) {
	if liveDevice {
		if recorder, ok := bus.(*i2ctest.Record); ok {
			// Clear the operations buffer.
			recorder.Ops = make([]i2ctest.IO, 0, 32)
		}
	} else {
		if len(playbackOps) == 1 {
			pb := bus.(*i2ctest.Playback)
			pb.Ops = playbackOps[0]
			pb.Count = 0
		}
	}
	conn := &i2c.Dev{Bus: bus, Addr: DefaultI2CAddress}
	dev := NewConnSerLCD(conn, 4, 20)

	return dev, nil
}

// shutdown dumps the recorder values if we we're running a live device.
func shutdown(t *testing.T) {
	if recorder, ok := bus.(*i2ctest.Record); ok {
		t.Logf("%#v", recorder.Ops)
	}
}

// This tests all functions in the TextDisplay interface.
func TestInterface(t *testing.T) {
	dev, err := getDev(t, pbInterface)
	defer shutdown(t)
	if err != nil {
		t.Fatal(err)
	}
	errs := displaytest.TestTextDisplay(dev, liveDevice)
	for _, err := range errs {
		if !errors.Is(err, display.ErrNotImplemented) {
			t.Error(err)
		}
	}
}

// TestBacklight verifies the backlight turns off.
func TestBacklight(t *testing.T) {
	if liveDevice && !eepromTests {
		return
	}
	dev, err := getDev(t, pbBacklight)
	defer shutdown(t)
	if err != nil {
		t.Fatal(err)
	}
	_ = dev.Clear()
	_, err = dev.WriteString("Set Backlight Off")
	if err != nil {
		t.Error(err)
	}

	err = dev.Backlight(0)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(sleepDuration)
	_ = dev.Clear()
	err = dev.Backlight(0xff)
	if err != nil {
		t.Error(err)
	}
	_, _ = dev.WriteString("Set Backlight On")
	time.Sleep(sleepDuration)
}

// TestRGBBacklight tests the RGB Backlight works as expected
func TestRGBBacklight(t *testing.T) {
	if liveDevice && !eepromTests {
		return
	}
	dev, err := getDev(t, pbRGBBacklight)
	defer shutdown(t)
	if err != nil {
		t.Fatal(err)
	}
	_ = dev.Clear()
	for ix := range 3 {
		w := make([]display.Intensity, 3)
		w[ix] = 0xff
		_, err = dev.WriteString(fmt.Sprintf("Set Backlight %#v", w))
		if err != nil {
			t.Error(err)
		}
		err = dev.RGBBacklight(w[0], w[1], w[2])
		if err != nil {
			t.Error(err)
		}
		time.Sleep(sleepDuration)
	}
	err = dev.RGBBacklight(0xff, 0xff, 0xff)
	if err != nil {
		t.Error(err)
	}
}

// TestContrast() checks contrast operations.
func TestContrast(t *testing.T) {
	if liveDevice && !eepromTests {
		return
	}
	dev, err := getDev(t, pbContrast)
	defer shutdown(t)
	if err != nil {
		t.Fatal(err)
	}
	_ = dev.Clear()
	_, _ = dev.WriteString("Set Contrast 5")
	err = dev.Contrast(5)
	if err != nil {
		t.Error(err)
	}
	_ = dev.MoveTo(dev.MinRow()+1, dev.MinCol())
	time.Sleep(sleepDuration)
	_, _ = dev.WriteString("Set Contrast 0x80")
	time.Sleep(sleepDuration)
	err = dev.Contrast(0x80)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(sleepDuration)
	_ = dev.Contrast(40)
}

func TestHalt(t *testing.T) {
	dev, err := getDev(t, pbHalt)
	defer shutdown(t)
	if err != nil {
		t.Fatal(err)
	}
	err = dev.Halt()
	if err != nil {
		t.Error(err)
	}
}
