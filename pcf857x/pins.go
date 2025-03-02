// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package pcf857x

import (
	"log"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
)

type pcfPin struct {
	dev    *Dev
	number int
	name   string
}

func (pin *pcfPin) DefaultPull() gpio.Pull {
	return gpio.Float
}

func (pin *pcfPin) Function() string {
	return "Out"
}

func (pcf *pcfPin) Halt() error {
	return nil
}

func (pin *pcfPin) In(pull gpio.Pull, edge gpio.Edge) error {
	// To use a pin for input, you must write a High to that pin, and then
	// perform the read. The chip doesn't natively support pullup/pulldown.
	//
	// Refer to the datasheet for more information.
	v := gpio.GPIOValue(1 << pin.number)
	return pin.dev.write(v, v)
}

func (pin *pcfPin) Name() string {
	return pin.name
}

func (pin *pcfPin) Number() int {
	return pin.number
}

func (pin *pcfPin) Out(l gpio.Level) error {
	value := gpio.GPIOValue(0)
	mask := gpio.GPIOValue(1 << pin.number)
	if l {
		value = mask
	}
	return pin.dev.write(value, mask)
}

func (pin *pcfPin) Pull() gpio.Pull {
	return gpio.Float
}

func (pin *pcfPin) Read() gpio.Level {

	result := gpio.Low
	mask := gpio.GPIOValue(1 << pin.number)

	value, err := pin.dev.read(mask)
	if err == nil {
		result = (value & mask) == mask
	} else {
		log.Println(err)
	}

	return result
}

func (pin *pcfPin) PWM(duty gpio.Duty, f physic.Frequency) error {
	return ErrNotImplmented
}

func (pin *pcfPin) String() string {
	return pin.name
}

// This device has an interrupt pin that can detect a change on the GPIO lines,
// however it doesn't let you detect a change on a specific pin.
func (pin *pcfPin) WaitForEdge(timeout time.Duration) bool {
	return false
}

var _ gpio.PinIO = &pcfPin{}
