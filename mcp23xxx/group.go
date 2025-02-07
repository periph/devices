// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package mcp23xxx

import (
	"fmt"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/pin"
)

// The internal structure for a group of pins.
type pinGroup struct {
	dev         *Dev
	port        int
	pins        []*portpin
	defaultMask gpio.GPIOValue
}

// Group returns a gpio.Group that is made up of the specified pins.
func (dev *Dev) Group(port int, pins []int) *gpio.Group {
	grouppins := make([]*portpin, len(pins))
	for ix, number := range pins {
		pp, ok := dev.Pins[port][number].(*portpin)
		if !ok {
			return nil
		}
		grouppins[ix] = pp
	}
	defMask := gpio.GPIOValue((1 << len(pins)) - 1)
	var pgif interface{} = &pinGroup{dev: dev, port: port, pins: grouppins, defaultMask: defMask}
	if gpiogroup, ok := pgif.(gpio.Group); ok {
		return &gpiogroup
	}

	return nil
}

// Pins returns the set of pin.Pin that make up that group.
func (pg *pinGroup) Pins() []pin.Pin {
	pins := make([]pin.Pin, len(pg.pins))

	for ix, p := range pg.pins {
		pins[ix] = p
	}
	return pins
}

// Given the offset within the group, return the corresponding GPIO pin.
func (pg *pinGroup) ByOffset(offset int) pin.Pin {
	return pg.pins[offset]
}

// Given the specific name of a pin, return it. If it can't be found, nil is
// returned.
func (pg *pinGroup) ByName(name string) pin.Pin {
	for _, pin := range pg.pins {
		if pin.Name() == name {
			return pin
		}
	}
	return nil
}

// Given the GPIO pin number, return that pin from the set.
func (pg *pinGroup) ByNumber(number int) pin.Pin {
	for _, pin := range pg.pins {
		if pin.Number() == number {
			return pin
		}
	}
	return nil
}

// Out writes value to the specified pins of the device/port. If mask is 0,
// the default mask of all pins in the group is used.
func (pg *pinGroup) Out(value, mask gpio.GPIOValue) error {
	if mask == 0 {
		mask = pg.defaultMask
	} else {
		mask &= pg.defaultMask
	}
	value &= mask
	// Convert the write value which is relative to the pins to the
	// absolute value for the port.
	wr := uint8(0)
	wrMask := uint8(0)
	for bit := range len(pg.pins) {
		if (mask & (1 << bit)) > 0 {
			if (value & 0x01) == 0x01 {
				wr |= 1 << pg.pins[bit].Number()
			}
			wrMask |= 1 << pg.pins[bit].Number()
		}
		value = value >> 1
	}
	port := pg.pins[0].port
	// Verify pins are set for output
	outputPins, err := port.iodir.readValue(true)
	if err != nil {
		return err
	}

	if ((outputPins ^ 0xff) & wrMask) != wrMask {
		outputPins &= (wrMask ^ 0xff)
		err = port.iodir.writeValue(outputPins, false)
		if err != nil {
			return err
		}
	}

	// Read the current value
	currentValue, err := port.olat.readValue(true)
	// Apply the mask to clear bits we're writing.
	currentValue &= (0xff ^ wrMask)
	// Or the value with the bits to modify
	currentValue |= wr
	// And, write the value out the port.
	return port.olat.writeValue(currentValue, true)
}

// Read reads from the device and port and returns the state of the GPIO
// pins in the group. If a pin specified by mask is not configured for
// input, it is transparently re-configured.
func (pg *pinGroup) Read(mask gpio.GPIOValue) (result gpio.GPIOValue, err error) {
	if mask == 0 {
		mask = pg.defaultMask
	} else {
		mask &= pg.defaultMask
	}
	// Compute the read mask
	rmask := uint8(0)
	for bit := range 8 {
		if (mask & (1 << bit)) > 0 {
			rmask |= (1 << pg.pins[bit].Number())
		}
	}
	// Make sure the direction for the pins involved in this write read is
	// Input.
	port := pg.pins[0].port
	currentIn, err := port.iodir.readValue(true)
	if err != nil {
		return
	}
	// We need to make some pins Input. Write the value to the iodir register.
	if (currentIn & rmask) != rmask {
		err = port.iodir.writeValue(currentIn|rmask, false)
		if err != nil {
			return
		}
	}
	// Now, perform the read itself.
	v, err := port.gpio.readValue(false)
	if err != nil {
		return
	}
	// Now convert the set pins into the Group value
	for ix, pin := range pg.pins {
		if (v & (1 << pin.Number())) > 0 {
			result |= 1 << ix
		}
	}
	return
}

// WaitForEdge listens for a GPIO pin change event. The MCP23XXXX devices
// can't directly signal an edge event. To do this, you must call
// Dev.SetEdgePin() with a HOST GPIO pin configured for falling edge
// detection. That pin should be connected to the MCP23XXX INT pin. When
// a falling edge is detected on the supplied host GPIO pin, the code
// will return the GPIO Pin number on the device that changed.
//
// Note that the MCP23XXX devices only detect change. You can't configure
// falling or rising edge. Consequently, the returned edge will always be
// gpio.NoEdge.
//
// For a change event to be detected, the pin must be configured for input.
// This function will NOT set pins for input. Additionally, the calling
// code must set the INTCON register appropriately. Refer to the datasheet.
//
// In the event that the changed pin is NOT part of the io group, the
// triggering pin number will be returned, along with the error
// ErrPinNotInGroup
func (pg *pinGroup) WaitForEdge(timeout time.Duration) (number int, edge gpio.Edge, err error) {
	return -1, gpio.NoEdge, gpio.ErrGroupFeatureNotImplemented
}

// Halt() interrupts a pending WaitForEdge() call if one is in process.
func (pg *pinGroup) Halt() error {
	if pg.dev.edgePin != nil {
		var ifpin interface{} = pg.dev.edgePin
		if r, ok := ifpin.(conn.Resource); ok {
			return r.Halt()
		}
	}
	// TODO: I think we want to call Dev.Halt()
	return nil
}

// String returns the device variant name and configured pins for the group.
func (pg *pinGroup) String() string {
	s := fmt.Sprintf("%s - [ ", pg.dev)
	for ix := range len(pg.pins) {
		s += fmt.Sprintf("%d ", pg.pins[ix].Number())
	}
	s += "]"
	return s
}

var _ gpio.Group = &pinGroup{}
