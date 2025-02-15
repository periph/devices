// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package hd44780

import (
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/devices/v3/mcp23xxx"
)

const (
	// Name is the LCD pin name, and the integer value is the GPIO
	// number (not physical) of the MCP23008 I2C GPIO Expander.
	d4           = 3
	d5           = 4
	d6           = 5
	d7           = 6
	rsPin        = 1
	enablePin    = 2
	backlightPin = 7
)

// This function returns a display configured to use the Adafruit I2C/SPI LCD Backpack.
//
// # Product Information
//
// https://www.adafruit.com/product/292
//
// The I2C side of this backpack uses an MCP23008 I/O expander. This function
// creates an MCP23008 device with the required pin configuration. To use this,
// get an I2C bus, and call this function with the bus, i2c address, number of
// rows, and columns.
func NewAdafruitI2CBackpack(bus i2c.Bus, address uint16, rows, cols int) (*HD44780, error) {
	mcp, err := mcp23xxx.NewI2C(bus, mcp23xxx.MCP23008, address)
	if err != nil {
		return nil, err
	}
	gr := *mcp.Group(0, []int{d4, d5, d6, d7, rsPin, enablePin, backlightPin})
	grPins := gr.Pins()
	reset := grPins[4].(gpio.PinOut)
	enable := grPins[5].(gpio.PinOut)
	bl := grPins[6].(gpio.PinOut)
	return NewHD44780(gr, &reset, &enable, &bl, rows, cols)
}
