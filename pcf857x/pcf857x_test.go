// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.
//
// This test assumes you have a PCF8575 and that the data pins are jumpered
//
// 0 => 8
// 1 => 9
// ...
// 7 => 15

package pcf857x

import (
	"errors"
	"strings"
	"testing"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/i2c/i2ctest"
)

func getDev(recordingName string, t *testing.T) (*Dev, error) {
	// Create a new I2C IO extender
	bus := i2ctest.Playback{Ops: recordingData[recordingName]}
	extender, err := New(&bus, DefaultAddress, PCF8575)
	if err != nil {
		t.Error(err)
	}
	return extender, err
}

// Test basic dev and pin functions.
func TestBasic(t *testing.T) {
	dev, err := getDev(t.Name(), t)
	if err != nil {
		return
	}
	t.Logf("dev=%#v", dev)
	pin := dev.Pins[1]
	t.Logf("pin=%#v", pin)

	s := dev.String()
	if len(s) == 0 {
		t.Error("String() failure")
	} //

	if len(dev.Pins) != 16 {
		t.Errorf("expected 16 GPIO pins. Found %d", len(dev.Pins))
	}

	e := pin.PWM(10, 10)
	if !errors.Is(e, ErrNotImplmented) {
		t.Errorf("PWM() expected ErrNotImplemented. Received %#v", e)
	}
	if pin.Halt() != nil {
		t.Error("expected nil on pin.Halt()")
	}

	if pin.Name() != pin.String() {
		t.Error("pin.Name()!=pin.String()")
	}

	if !strings.HasPrefix(pin.Name(), dev.String()) {
		t.Errorf("Expected pin.Name()=%s to start with dev.String()=%s", pin.Name(), dev.String())
	}

	err = dev.Halt()
	if err != nil {
		t.Error(err)
	}
}

// Test that the pins are registered in gpioreg as expected.
func TestGPIOReg(t *testing.T) {
	dev, err := getDev(t.Name(), t)
	if err != nil {
		return
	}

	for ix := range dev.width {
		p := dev.Pins[ix]
		if p.Number() != ix {
			t.Errorf("pin.Number() does not match ordinal position %d! Found %d", ix, p.Number())
		}
		pReg := gpioreg.ByName(p.Name())
		if pReg == nil {
			t.Errorf("pin %s not found in gpioreg", p.Name())
		}
	}
}

// This test goes through the pins from 0-7, and writes to them and then reads
// the value on pin+8 and verifies it's correct. Then, it reverses direction and
// writes to pin[8] and reads from pin[0].
func TestPinsSequentially(t *testing.T) {
	dev, err := getDev(t.Name(), t)
	if err != nil {
		return
	}
	limit := dev.width >> 1
	for ixOuter := range limit {
		// Iterate over the lower 8 pins
		for ixInner := range limit {
			// Sequentially, set each pin to High if it's not the value of ix
			p := dev.Pins[ixInner]
			pRead := dev.Pins[ixInner+limit]
			for direction := range 2 {
				writeLevel := gpio.Level(ixInner != ixOuter)
				err = p.Out(writeLevel)
				if err != nil {
					t.Error(err)
				}
				readVal := pRead.Read()
				if readVal != writeLevel {
					t.Errorf("wrote %t to pin[%d]. Expected same on pin[%d], found %t",
						writeLevel,
						p.Number(),
						pRead.Number(),
						readVal)
				}
				if direction == 0 {
					// swap the direction so we're now going pin[8] => pin[0] and stay in
					// the loop to repeat the test.
					x := pRead
					pRead = p
					p = x
				}
			}
		}
	}
}

// This tests the group functionality.
func TestGroup(t *testing.T) {
	dev, err := getDev(t.Name(), t)

	if err != nil {
		return
	}
	defer func() { _ = dev.Halt() }()

	set1 := make([]int, dev.width>>1)
	set2 := make([]int, dev.width>>1)
	for ix := range len(set1) {
		set1[ix] = ix
		set2[ix] = ix + len(set1)
	}
	gr1, err := dev.Group(set1...)
	if err != nil {
		return
	}
	gr2, err := dev.Group(set2...)
	if err != nil {
		return
	}
	// Test the basic group functionality. Note that for group1, pinOffset==pin.Number, but
	// for group2, pinOffset!=pin.Number
	grTest := gr1
	for range 2 {
		for pinNumber, pin := range grTest.Pins() {
			x := grTest.ByNumber(pin.Number())
			if x == nil {
				t.Errorf("group.ByNumber() returned nil for pin %d", pin.Number())
			}
			x = grTest.ByOffset(pinNumber)
			if x == nil {
				t.Errorf("group.ByOffset returned nil for pin number %d", pinNumber)
			} else {
				if x.Number() != pin.Number() {
					t.Errorf("group.ByOffset() didn't return the expected pin. Expected %d, found %d", pin.Number(), x.Number())
				}
			}
			x = grTest.ByName(pin.Name())
			if x == nil || x.Name() != pin.Name() {
				t.Error("group.ByName() didn't find a pin or returned the wrong pin!")
			}
		}
		grTest = gr2
	}
	if len(gr1.String()) == 0 {
		t.Error("group.String() didn't return a value")
	}
	// Test the read/write functionality.
	limit := (1 << len(set1))
	for groupNumber := range 2 {
		for val := range limit {
			err = gr1.Out(gpio.GPIOValue(val), 0)
			if err != nil {
				t.Error(err)
			}
			read, err := gr2.Read(0)
			if err != nil {
				t.Error(err)
			}
			if read != gpio.GPIOValue(val) {
				t.Errorf("Error writing/reading groups. Wrote %d on write group %s, read %d on read group%s", val, gr1, read, gr2)
			}
		}
		if groupNumber == 0 {
			x := gr1
			gr1 = gr2
			gr2 = x
		}
	}
	err = gr1.Halt()
	if err != nil {
		t.Error(err)
	}
	err = gr2.Halt()
	if err != nil {
		t.Error(err)
	}
}
