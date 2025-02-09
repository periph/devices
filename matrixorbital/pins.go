// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package matrixorbital

import (
	"errors"
	"fmt"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
)

// A gpoPin is an output pin that can be toggled on the display. On the
// LK204-7T-1U, there are six GPO pins wired to LEDs. On the Adafruit
// USB LCD Backpack, there are four bare pins exposed. gpoPin implements
// gpio.PinOut.
type gpoPin struct {
	name    string
	number  int
	display *GPOEnabledDisplay
}

// A generic routine to create our set of pins.
func makePins(display *GPOEnabledDisplay, pins []gpio.PinOut) {
	for ix := range len(pins) {
		pin := &gpoPin{name: fmt.Sprintf("GPO%d", ix+1), number: ix + 1, display: display}
		pins[ix] = pin
	}
}

func (pin *gpoPin) Name() string {
	return pin.name
}

func (pin *gpoPin) Number() int {
	return pin.number
}

func (pin *gpoPin) String() string {
	return fmt.Sprintf("matrixorbital Pin: Name: %s Number %d", pin.name, pin.number)
}

func (pin *gpoPin) Halt() error {
	return nil
}

func (pin *gpoPin) Out(l gpio.Level) error {
	d := *pin.display
	return d.GPO(pin.number, l)
}

func (pin *gpoPin) Function() string {
	return "Out"
}

func (pin *gpoPin) PWM(duty gpio.Duty, f physic.Frequency) error {
	return errors.New("not implemented")
}

var _ gpio.PinOut = &gpoPin{}
