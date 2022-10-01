// Copyright 2022 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This file is largely a copy of mcp23xxx/pins.go, but with a reduced feature
// set for controlling the chips.

package tca95xx

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/pin"
)

// Pin extends gpio.PinIO interface with features supported by tca95xx devices.
type Pin interface {
	gpio.PinIO
	// SetPolarityInverted if set to true, GPIO register bit reflects the same logic state of the input pin.
	SetPolarityInverted(p bool) error
	// IsPolarityInverted returns true if the value of the input pin reflects inverted logic state.
	IsPolarityInverted() (bool, error)
}

type port struct {
	name string

	// GPIO basic registers
	input  registerCache // input at the pin
	output registerCache // output control, or flipflop state if read
	iodir  registerCache // direction
	ipol   registerCache // polarity setting
}

func (p *port) pins(count int) []Pin {
	result := make([]Pin, count)
	var i uint8
	for i = 0; i < uint8(count); i++ {
		result[i] = &portpin{
			port:   p,
			pinbit: i,
		}
	}
	return result
}

// Tx takes bytes to either read or write.  Only half duplex is supported so it
// is an error to pass 2 buffers at once.  The bytes are written or read from
// the same connection sequentially.
func (p *port) Tx(w, r []byte) (err error) {
	send := len(w)
	get := len(r)
	switch {
	case send > 0 && get > 0:
		return fmt.Errorf("tca95xx: only conn.Half duplex is supported")
	case send > 0:
		for i := 0; i < send; i++ {
			err = p.output.writeValue(w[i], false)
			if err != nil {
				return err
			}
		}
	case get > 0:
		var in uint8
		for i := 0; i < get; i++ {
			in, err = p.input.readValue(false)
			if err != nil {
				return err
			}
			r[i] = in
		}
	}

	return nil
}

// Duplex returns that this is a half duplex connection.
func (p *port) Duplex() conn.Duplex {
	return conn.Half
}

// String provides the name of this connection.
func (p *port) String() string {
	return p.name
}

type portpin struct {
	port   *port
	pinbit uint8
}

func (p *portpin) String() string {
	return p.Name()
}

func (p *portpin) Halt() error {
	// To halt all drive, set to high-impedance input
	return p.In(gpio.Float, gpio.NoEdge)
}

func (p *portpin) Name() string {
	return p.port.name + "_" + strconv.Itoa(int(p.pinbit))
}

func (p *portpin) Number() int {
	return int(p.pinbit)
}

func (p *portpin) Function() string {
	return string(p.Func())
}

func (p *portpin) In(pull gpio.Pull, edge gpio.Edge) error {
	// Set pullup
	switch pull {
	case gpio.PullDown:
		// pull down is not supported by any device
		return errors.New("tca95xx: PullDown is not supported")
	case gpio.PullUp:
		return errors.New("tca95xx: PullUp is not supported")
	case gpio.Float, gpio.PullNoChange:
		// Do nothing, supported.
	}

	// Interrupts are not via I2C bus, so supporting them is less than
	// ideal.
	if edge != gpio.NoEdge {
		return errors.New("tca95xx: edge detection not supported")
	}

	// Set pin to input
	return p.port.iodir.getAndSetBit(p.pinbit, true, true)
}

func (p *portpin) Read() gpio.Level {
	v, _ := p.port.input.getBit(p.pinbit, false)
	if v {
		return gpio.High
	}
	return gpio.Low
}

func (p *portpin) WaitForEdge(timeout time.Duration) bool {
	return false
}

func (p *portpin) Pull() gpio.Pull {
	return gpio.Float
}

func (p *portpin) DefaultPull() gpio.Pull {
	return gpio.Float
}

func (p *portpin) Out(l gpio.Level) error {
	err := p.port.iodir.getAndSetBit(p.pinbit, false, true)
	if err != nil {
		return err
	}
	return p.port.output.getAndSetBit(p.pinbit, l == gpio.High, true)
}

func (p *portpin) PWM(duty gpio.Duty, f physic.Frequency) error {
	return errors.New("tca95xx: PWM is not supported")
}

func (p *portpin) Func() pin.Func {
	v, _ := p.port.iodir.getBit(p.pinbit, true)
	if v {
		return gpio.IN
	}
	return gpio.OUT
}

func (p *portpin) SupportedFuncs() []pin.Func {
	return supportedFuncs[:]
}

func (p *portpin) SetFunc(f pin.Func) error {
	var v bool
	switch f {
	case gpio.IN:
		v = true
	case gpio.OUT:
		v = false
	default:
		return errors.New("tca95xx: Function not supported: " + string(f))
	}
	return p.port.iodir.getAndSetBit(p.pinbit, v, true)
}

func (p *portpin) SetPolarityInverted(pol bool) error {
	return p.port.ipol.getAndSetBit(p.pinbit, pol, true)
}
func (p *portpin) IsPolarityInverted() (bool, error) {
	return p.port.ipol.getBit(p.pinbit, true)
}

var supportedFuncs = [...]pin.Func{gpio.IN, gpio.OUT}
