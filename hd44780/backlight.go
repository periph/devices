// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package hd44780

import (
	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/gpio"
)

// A monochrome backlight  Implements display.Backlight It uses a single
// GPIO Pin to turn the backlight on or off.
type GPIOMonoBacklight struct {
	blPin gpio.PinOut
}

// Given a GPIO pin that turns the backlight on/off, construct a monobacklight
// to use with HD44780.
func NewBacklight(blPin gpio.PinOut) *GPIOMonoBacklight {
	return &GPIOMonoBacklight{blPin: blPin}
}

// Turn the display backlight on or off.
func (bl *GPIOMonoBacklight) Backlight(intensity display.Intensity) (err error) {
	if intensity == 0 {
		err = bl.blPin.Out(gpio.Low)
	} else {
		err = bl.blPin.Out(gpio.High)
	}
	return err
}

var _ display.DisplayBacklight = &GPIOMonoBacklight{}
