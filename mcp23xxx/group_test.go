// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package mcp23xxx

import (
	"testing"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/i2c/i2ctest"
)

const (
	liveTest = false
)

func TestGroup(t *testing.T) {
	bus := i2ctest.Playback{Ops: recordingData["TestGroup"]}
	extender, err := NewI2C(&bus, MCP23008, 0x20)
	if err != nil {
		t.Fatal(err)
	}

	for portnum, port := range extender.Pins {
		for _, pin := range port {
			t.Logf("Port: %d Pin: %d %s", portnum, pin.Number(), pin)
		}
	}

	// The test fixture has an led on pin 0, and also wires pin 0 -> pin 4.
	// If I don't set pin 4 for input, then it breaks the LED blinking.
	// It's not important to the test, but it's nice to have a visual indicator.
	reader, _ := interface{}(extender.Pins[0][4]).(gpio.PinIn)
	_ = reader.In(gpio.PullNoChange, gpio.NoEdge)
	var ifpin interface{} = extender.Pins[0][0]
	if writer, ok := ifpin.(gpio.PinOut); ok {
		l := gpio.High
		for range 20 {
			writer.Out(l)
			l = !l
			if liveTest {
				time.Sleep(500 * time.Millisecond)
			}
		}
	} else {
		t.Error("pin[0] not converted to gpio.PinOut!")
	}
}

// TestReadWrite exercises the group Out()/Read() functions. It's expected
// that pin 0 of MCP23xxx port 0 is connected to pin 4 of port 0, pin 1
// is connected to pin 5, etc...
func TestReadWrite(t *testing.T) {
	bus := i2ctest.Playback{Ops: recordingData["TestReadWrite"]}
	extender, err := NewI2C(&bus, MCP23008, 0x20)
	if err != nil {
		t.Fatal(err)
	}
	defMask := gpio.GPIOValue(0xf)
	gOut := *extender.Group(0, []int{0, 1, 2, 3})
	if gOut == nil {
		t.Error("gOut is nil!")
	}
	gRead := *extender.Group(0, []int{4, 5, 6, 7})
	if gRead == nil {
		t.Error("gRead is nil!")
	}
	// Turn off the GPIOs
	defer gOut.Out(0, 0)
	defer gRead.Out(0, 0)

	for i := range 2 {
		if i == 1 {
			/* Now invert it. */
			x := gRead
			gRead = gOut
			gOut = x
		}
		for i := range gpio.GPIOValue(16) {
			err := gOut.Out(i, 0)
			if err != nil {
				t.Error(err)
			}
			if liveTest {
				time.Sleep(time.Millisecond)
			}
			r, err := gRead.Read(defMask)
			if err != nil {
				t.Error(err)
			}
			if r != i {
				t.Errorf("Error reading/writing GPIO Group(). Wrote 0x%x Read 0x%x", i, r)
			}
		}
	}

	// For this test, write to the pins individually, and then
	// confirm read on the other set works as expected.
	x := gRead
	gRead = gOut
	gOut = x
	t.Log(gRead)
	pinset := extender.Pins[0][:4]
	for i := range gpio.GPIOValue(16) {
		wvalue := i
		for bit := range 4 {
			if (wvalue & (1 << bit)) == (1 << bit) {
				pinset[bit].Out(gpio.High)
			} else {
				pinset[bit].Out(gpio.Low)
			}
		}
		r, err := gRead.Read(0)
		if err != nil {
			t.Error(err)
		}
		if r != i {
			t.Errorf("Error writing GPIO pins and reading back result. Read 0x%x Expected 0x%x", r, i)
		}
	}
}
