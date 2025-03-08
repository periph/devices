// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package hd44780

import (
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/devices/v3/pcf857x"
)

const (
	// Name is the LCD pin name, and the integer value is the GPIO
	// number (not physical) of the PCF8574 I2C GPIO Expander.
	pcf_d4           = 4
	pcf_d5           = 5
	pcf_d6           = 6
	pcf_d7           = 7
	pcf_rsPin        = 0
	pcf_enablePin    = 2
	pcf_backlightPin = 3
	pcf_rwPin        = 1
)

// This function returns a display configured to use the pcf8574 i2c backpacks.
//
// # Product Information
//
// https://www.handsontec.com/dataspecs/I2C_2004_LCD.pdf
//
// This function creates a PCF8574 backpack device with the required pin
// configuration. To use this, get an I2C bus, and call this function with the
// bus, i2c address, number of rows, and columns.
func NewPCF857xBackpack(bus i2c.Bus, address uint16, rows, cols int) (*HD44780, error) {
	pcf, err := pcf857x.New(bus, address, pcf857x.PCF8574)
	if err != nil {
		return nil, err
	}
	// R/W is connected on this backpack. Set it to low.
	_ = pcf.Pins[pcf_rwPin].Out(gpio.Low)

	// Create our gpio.Group
	gr, _ := pcf.Group(pcf_d4, pcf_d5, pcf_d6, pcf_d7, pcf_rsPin, pcf_enablePin, pcf_backlightPin)
	grPins := gr.Pins()
	reset := grPins[4].(gpio.PinOut)
	enable := grPins[5].(gpio.PinOut)
	bl := grPins[6].(gpio.PinOut)
	return NewHD44780(gr, &reset, &enable, NewBacklight(bl), rows, cols)
}
