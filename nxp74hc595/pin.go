// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package nxp74hc595

import (
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
)

type Pin struct {
	dev    *Dev
	name   string
	number int
}

// Halt implements conn.Resource.
func (pin *Pin) Halt() error {
	return nil
}

// Name returns the name of the GPIO pin.
func (pin *Pin) Name() string {
	return pin.name
}

// Number returns the number of the GPIO pin.
func (pin *Pin) Number() int {
	return pin.number
}

// Deprecated: returns "Out"
func (pin *Pin) Function() string {
	return "Out"
}

// Write the specified gpio.Level to the pin.
func (pin *Pin) Out(l gpio.Level) error {
	mask := gpio.GPIOValue(1 << pin.number)
	v := gpio.GPIOValue(0)
	if l {
		v = mask
	}
	return pin.dev.write(v, mask)
}

// Not implemented.
func (pin *Pin) PWM(duty gpio.Duty, f physic.Frequency) error {
	return ErrNotImplemented
}

func (pin *Pin) String() string {
	return pin.name
}
