// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This package provides a driver for the TI/NXP PCF857X I2C I/O Expander. These
// devices provide 8 pins (PCF8574) or 16 pins (PCF8575) of
// "quasi-bidirectional" input/output. This device is commonly used in LCD
// backpacks, particularly those sold as LCD2004, LCD1602.
//
// The PCF8575 is a 16-pin device that is functionally identical to the PCF8574.
// When communicating with the PCF8575 reads and writes are 2 bytes wide, while
// they're one byte wide with the PCF85754
//
// # Datasheet
//
// https://www.ti.com/lit/ds/symlink/pcf8574.pdf
//
// A good description of the I2C LCD backpack usage can be found here:
//
// https://www.handsontec.com/dataspecs/I2C_2004_LCD.pdf
//
// Adafruit also sells a breakout board with these chips. See here:
//
// https://www.adafruit.com/product/5611
//
// # Notes
//
// This device is very simple and doesn't have functionality that similar
// devices do. Specifically, GPIO Read() consists of writing a High out a pin,
// and then reading it to see if it is still high, or if it has transitioned to
// low.
//
// Setting a pin to Low activates an Open Drain to ground.
//
// You cannot detect edge change on a specific pin. There is an interrupt pin
// that can be used to detect a change on the GPIO pins, but it doesn't tell you
// which pin changed.
//
// This chip doesn't implement normal i2c register architectures. You write 8 or
// 16 bits out, and that sets the corresponding pins, or you read 8/16 bits and
// get the state of the pins.
package pcf857x

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/pin"
)

// Variant represents the actual chip model.
type Variant string

const (
	PCF8574 Variant = "PCF8574"
	PCF8575 Variant = "PCF8575"

	DefaultAddress uint16 = 0x20
)

var (
	ErrNotImplmented error = errors.New("pcf857x: not implemented")
)

// Dev is representation of a PCF857x device.
type Dev struct {
	// The pins exposed by the device. For PCF8574, this will be 8 pins, and
	// 16 pins for the PCF8575
	Pins     []gpio.PinIO
	mask     gpio.GPIOValue
	width    int
	chipType Variant

	mu     sync.Mutex
	d      *i2c.Dev
	value  gpio.GPIOValue
	groups []Group
}

type Group struct {
	pins []pcfPin
	dev  *Dev
}

// New creates a new PCF857x io expander and returns it. chip should be one of
// the Variant constants above.
func New(bus i2c.Bus, address uint16, chip Variant) (*Dev, error) {
	dev := &Dev{d: &i2c.Dev{Bus: bus, Addr: address},
		chipType: chip}
	if chip == PCF8574 {
		dev.width = 8
	} else {
		dev.width = 16
	}
	dev.mask = gpio.GPIOValue((1 << dev.width) - 1)
	dev.Pins = make([]gpio.PinIO, dev.width)
	sDev := dev.String()
	for ix := range dev.width {
		name := fmt.Sprintf("%s_GPIO%d", sDev, ix)
		dev.Pins[ix] = &pcfPin{dev: dev, number: ix, name: name}
		_ = gpioreg.Register(dev.Pins[ix])
	}
	return dev, nil
}

// Group returns a GPIO Group comprised of the specified pin numbers. A
// gpio.Group allows you to perform writes to multiple pins in one operation.
func (dev *Dev) Group(pinNumbers ...int) (gpio.Group, error) {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	gr := Group{dev: dev, pins: make([]pcfPin, len(pinNumbers))}
	for ix, pinNumber := range pinNumbers {
		if p, ok := dev.Pins[pinNumber].(*pcfPin); ok {
			gr.pins[ix] = *p
		}
	}
	dev.groups = append(dev.groups, gr)
	return &gr, nil
}

// Halt shuts down the device, and frees any pin groups.
func (dev *Dev) Halt() error {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	for _, gr := range dev.groups {
		_ = gr.Halt()
	}
	dev.groups = make([]Group, 0)
	dev.Pins = make([]gpio.PinIO, 0)
	return nil
}

// read performs the low level i2c read operation from the device.
func (dev *Dev) read(mask gpio.GPIOValue) (gpio.GPIOValue, error) {

	// Before you can read a pin, you must have set it to high. If nothing
	// pulls that down, then it's high. If it's pulled down, it's low.
	err := dev.write(mask, mask)
	if err != nil {
		return 0, fmt.Errorf("pcf857x: %w", err)
	}

	dev.mu.Lock()
	defer dev.mu.Unlock()
	byteCount := 1
	if dev.width > 8 {
		byteCount += 1
	}

	r := make([]byte, byteCount)
	err = dev.d.Tx(nil, r)
	if err != nil {
		return 0, fmt.Errorf("pcf857x: %w", err)
	}
	result := gpio.GPIOValue(r[0])
	if byteCount > 1 {
		result |= gpio.GPIOValue(r[1]) << 8

	}
	// turn off the bits we just read so that the next time through, we force
	// the write high on them.
	dev.value = result
	result &= mask

	return result, nil
}

// write performs the low-level write to the device. If the resulting value of
// the device is unchanged, the write is skipped.
func (dev *Dev) write(value, mask gpio.GPIOValue) error {
	// fmt.Printf("pcf857x.write(value=0x%x, mask=0x%x)\n", value, mask)
	// fmt.Printf("dev.mask=0x%x\n", dev.mask)
	dev.mu.Lock()
	defer dev.mu.Unlock()
	wrValue := dev.value & (dev.mask ^ mask)
	wrValue |= (value & mask)
	// fmt.Printf("pcf857x.write() wrValue=0x%x, mask=0x%x\n", wrValue, mask)
	if dev.value == wrValue {
		return nil
	}
	byteCount := 1
	if dev.width > 8 {
		byteCount += 1
	}
	w := make([]byte, byteCount)
	for ix := range byteCount {
		w[ix] = byte(wrValue >> (ix * 8))
	}
	err := dev.d.Tx(w, nil)
	if err == nil {
		dev.value = wrValue
	} else {
		err = fmt.Errorf("pcf857x: %w", err)
	}
	return err
}

func (dev *Dev) String() string {
	return fmt.Sprintf("%s_%x", dev.chipType, dev.d.Addr)
}

// Pins returns the set of pins that make up this group.
func (gr *Group) Pins() []pin.Pin {
	pins := make([]pin.Pin, len(gr.pins))
	for ix := range len(gr.pins) {
		pins[ix] = &gr.pins[ix]
	}
	return pins
}

// This converts a mask for a group operation into a mask suitable for writing
// to the device.
func (gr *Group) groupMaskToDevMask(mask gpio.GPIOValue) gpio.GPIOValue {
	m := gpio.GPIOValue(0)
	for ix := range len(gr.pins) {
		currentBit := gpio.GPIOValue(1 << ix)
		if (mask & currentBit) == currentBit {
			pinBit := gpio.GPIOValue(1) << gr.pins[ix].number
			m |= pinBit
		}
	}
	return m
}

// Return the GPIO pin by offset within the group.
func (gr *Group) ByOffset(offset int) pin.Pin {
	return &gr.pins[offset]
}

// Return the GPIO pin by name.
func (gr *Group) ByName(name string) pin.Pin {
	for ix := range len(gr.pins) {
		if gr.pins[ix].name == name {
			return &gr.pins[ix]
		}
	}
	return nil
}

// Return the GPIO pin by it's pin number on the device.
func (gr *Group) ByNumber(number int) pin.Pin {
	for ix := range len(gr.pins) {
		if gr.pins[ix].number == number {
			return &gr.pins[ix]
		}
	}
	return nil
}

// Out writes the specified value to the device. Only pins identified by mask
// are modified.
func (gr *Group) Out(value, mask gpio.GPIOValue) error {
	if mask == 0 {
		mask = (1 << len(gr.pins)) - 1
	}
	wrMask := gr.groupMaskToDevMask(mask)
	wr := gpio.GPIOValue(0)
	for ix, pin := range gr.pins {
		if (value & gpio.GPIOValue(1<<ix)) > 0 {
			wr |= 1 << pin.number
		}
	}
	// fmt.Printf("group.out wr=0x%x wrMask=0x%x\n",wr,wrMask)
	return gr.dev.write(wr, wrMask)
}

// Read returns the current values of the pins within the group identified by
// mask.
func (gr *Group) Read(mask gpio.GPIOValue) (gpio.GPIOValue, error) {
	if mask == 0 {
		mask = (1 << len(gr.pins)) - 1
	}
	devMask := gr.groupMaskToDevMask(mask)

	v, err := gr.dev.read(devMask)
	if err != nil {
		return 0, fmt.Errorf("pcf857x: %w", err)
	}

	// Now, convert it back to a group value.
	result := gpio.GPIOValue(0)
	for ix, pin := range gr.pins {
		currentBit := gpio.GPIOValue(1 << ix)
		if (mask & currentBit) == currentBit {
			if (v & gpio.GPIOValue(1<<pin.number)) > 0 {
				result |= currentBit
			}
		}
	}

	return result, nil
}

// This chip does not support waiting for edge on either a pin or a group. There
// is an interrupt pin, but you can't set a mask of pins that will trigger it. To
// do that, you connect a GPIO pin from the host device that supports WaitForEdge
// to monitor the INTR pin.
func (gr *Group) WaitForEdge(timeout time.Duration) (number int, edge gpio.Edge, err error) {
	// TODO: Implement wait for edge in the same way that it is for mcp23008
	return 0, gpio.NoEdge, ErrNotImplmented
}

// Halt stops the pin group. It cannot be used after this call.
func (gr *Group) Halt() error {
	gr.pins = make([]pcfPin, 0)
	gr.dev = nil
	return nil
}

func (gr *Group) String() string {
	s := gr.dev.String() + "[ "
	for ix := range len(gr.pins) {
		s += fmt.Sprintf("%d ", gr.pins[ix].Number())
	}
	s += "]"
	return s
}

var _ gpio.Group = &Group{}
