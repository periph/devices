// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// The Waveshare 1602 LCD is a 2 line by 16 column LCD display. It's available
// in multiple variants:
//
//   - LCD1602 5V Blue Backlight
//   - LCD1602 3.3V Yellow Backlight
//   - LCD1602 3.3V Blue Backlight
//
// These are bare LCD displays with no backpack. They have an hd44780 compatible
// driver chip. Use the driver located in the [hd44780] package.
//
//   - LCD1602 I²C Module, White color w/ Blue Background, 16x2 characters, 3.3V/5V
//   - LCD1602 I²C Module, Options for 3 Colors 3.3v/5v Backlight Adjustable
//
// These displays use the [aip31068] I²C LCD Driver chip. The command set is
// compatible with the HD44780. The tri-color version has purchase options to
// select a backlight color and uses an SN3193 to dim the backlight.
//
//   - LCD1602 RGB Module, 16x2 Characters LCD, RGB Backlight, 3.3V/5V, I²C Bus
//
// This display uses the AiP31068 I²C LCD Driver w/ a PCA9633 RGB LED PWM
// controller.
package waveshare1602

import (
	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/devices/v3/aip31068"
	"periph.io/x/devices/v3/pca9633"
)

type Variant string

const (
	// SKU 19537 - RGB Backlight
	LCD1602RGBBacklight Variant = "LCD1602RGBBacklight"
	// SKU 23991 - I²C w/ Monochrome Backlight
	LCD1602MonoBacklight Variant = "LCD1602MonoBacklight"
	// Not Implemented. SKU 30494, 30495, and 30496. Uses an SN3193 for
	// controlling the backlight.
	LCD1602DimmableMonoBacklight Variant = "LCD1602DimmableMonoBacklight"

	_LCD_ADDRESS uint16 = 0x3e
	_RGB_ADDRESS uint16 = 0x60
)

type RGBBLController struct {
	controller *pca9633.Dev
	variant    Variant
}

// Create new LCD display.
func New(bus i2c.Bus, variant Variant, rows, cols int) (*aip31068.Dev, error) {
	var bl any

	if variant == LCD1602RGBBacklight {
		blcontroller, err := pca9633.New(bus, _RGB_ADDRESS, pca9633.STRUCT_OPENDRAIN)
		if err != nil {
			return nil, err
		}
		bl = &RGBBLController{variant: variant, controller: blcontroller}
	} else if variant == LCD1602DimmableMonoBacklight {
		return nil, display.ErrNotImplemented
	}
	return aip31068.New(bus, _LCD_ADDRESS, bl, rows, cols)
}

func (bl *RGBBLController) String() string {
	return string(bl.variant)
}

// For units that have an RGB Backlight, set the backlight color/intensity.
// This unit does not persist settings in EEPROM, so you can call it as often
// as desired. The range of the values is 0-255.
func (bl *RGBBLController) RGBBacklight(red, green, blue display.Intensity) error {
	// The device is really connected to the LEDs in this channel order...
	return bl.controller.Out(blue, green, red)
}

var _ display.DisplayRGBBacklight = &RGBBLController{}
