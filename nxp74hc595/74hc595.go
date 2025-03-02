// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// The 74HC595 is a serial shift register. It converts a serial stream to a
// parallel output. For example, you can use it as an SPI => Parallel
// converter.
//
// # Datasheet
//
// https://www.nexperia.com/product/74HC595D
//
// There's a nice tutorial on the device here:
//
// https://docs.arduino.cc/tutorials/communication/guide-to-shift-out/
package nxp74hc595

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/pin"
	"periph.io/x/conn/v3/spi"
)

const (
	devMask = 0xff
	devName = "74HC595"
	numPins = 8
)

var (
	ErrNotImplemented = errors.New("nxp74hc595: not implemented")
)

// Dev represents a 74hc595 device.
type Dev struct {
	Pins []gpio.PinOut

	mu    sync.Mutex
	conn  spi.Conn
	value gpio.GPIOValue
}

// Group implements gpio.Group and provides a way to write to multiple GPO pins
// in a single transaction.
type Group struct {
	dev  *Dev
	pins []Pin
}

// New accepts an spi.Conn and returns a new HC74595 device.
func New(conn spi.Conn) (*Dev, error) {
	// setting value to an invalid initial state forces the first write to
	// happen, even if it's 0.
	dev := Dev{conn: conn, value: gpio.GPIOValue(1 << 9), Pins: make([]gpio.PinOut, numPins)}
	for ix := range numPins {
		dev.Pins[ix] = &Pin{number: ix, name: fmt.Sprintf("%s_GPO%d", devName, ix), dev: &dev}
	}
	return &dev, nil
}

// write does the low-level write to the device.
func (dev *Dev) write(value, mask gpio.GPIOValue) error {

	dev.mu.Lock()
	defer dev.mu.Unlock()
	newValue := (dev.value & (devMask ^ mask)) | (value & mask)
	if dev.value == newValue {
		return nil
	}
	var err error
	var w = []byte{byte(newValue)}
	err = dev.conn.Tx(w, nil)
	if err == nil {
		dev.value = newValue
	}
	return err
}

// Group returns a subset of pins on the device as a gpio.Group. A Group
// allows you to write to multiple pins in a single transaction.
func (dev *Dev) Group(pins ...int) (gpio.Group, error) {
	gr := Group{dev: dev, pins: make([]Pin, len(pins))}
	for ix, pinNumber := range pins {
		if p, ok := dev.Pins[pinNumber].(*Pin); ok {
			gr.pins[ix] = *p
		}
	}
	return &gr, nil
}

// Halt disables the device
func (dev *Dev) Halt() (err error) {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	dev.Pins = make([]gpio.PinOut, 0)
	dev.conn = nil
	return
}

func (dev *Dev) String() string {
	return devName
}

// Return the set of GPO Pins that are associated with this group.
func (gr *Group) Pins() []pin.Pin {
	result := make([]pin.Pin, len(gr.pins))
	for ix, p := range gr.pins {
		result[ix] = &p
	}
	return result

}

// Given an offset of a pin into the group, return that pin.
func (gr *Group) ByOffset(offset int) pin.Pin {
	return &gr.pins[offset]
}

// Given a name of a pin in the group, return that pin.
func (gr *Group) ByName(name string) pin.Pin {
	for _, p := range gr.pins {
		if p.name == name {
			return &p
		}
	}
	return nil
}

// Given the pin number of a pin within the group, return that pin.
func (gr *Group) ByNumber(number int) pin.Pin {
	for _, p := range gr.pins {
		if p.number == number {
			return &p
		}
	}
	return nil
}

// Out writes the value to the device. Only pins identified by mask are
// modified.
func (gr *Group) Out(value, mask gpio.GPIOValue) error {
	if mask == 0 {
		mask = gpio.GPIOValue(1<<len(gr.pins)) - 1
	}
	wrMask := gpio.GPIOValue(0)
	wrValue := gpio.GPIOValue(0)
	for ix := range len(gr.pins) {
		currentBit := gpio.GPIOValue(1 << ix)
		if (mask & currentBit) == currentBit {
			wrMask |= gpio.GPIOValue(1 << gr.pins[ix].number)
		}
		if (value & currentBit) == currentBit {
			wrValue |= gpio.GPIOValue(1 << gr.pins[ix].number)
		}
	}
	return gr.dev.write(wrValue, wrMask)
}

// Read is not available for this device.
func (gr *Group) Read(mask gpio.GPIOValue) (gpio.GPIOValue, error) {
	return 0, ErrNotImplemented
}

// WaitForEdge is not available for this device.
func (gr *Group) WaitForEdge(timeout time.Duration) (int, gpio.Edge, error) {
	return 0, gpio.NoEdge, ErrNotImplemented
}

// Halt frees the group's resources and prevents it from being used again.
func (gr *Group) Halt() error {
	gr.pins = nil
	return nil
}

func (gr *Group) String() string {
	s := gr.dev.String() + "[ "
	for ix := range len(gr.pins) {
		s += fmt.Sprintf("%d ", gr.pins[ix].number)
	}
	s += "]"
	return s
}

var _ gpio.PinOut = &Pin{}
var _ gpio.Group = &Group{}
