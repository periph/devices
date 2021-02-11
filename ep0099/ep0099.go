// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package ep0099

import (
	"errors"

	"periph.io/x/conn/v3/i2c"
)

var errInvalidAddress = errors.New("Invalid EP-0099 address")
var errInvalidChannel = errors.New("Invalid EP-0099 channel")

type State byte

const (
	StateOff State = 0x00
	StateOn  State = 0xFF
)

type Dev struct {
	i2c   i2c.Dev
	state [4]State
}

func New(bus i2c.Bus, address uint16) (*Dev, error) {
	if err := isValidAddress(address); err != nil {
		return nil, err
	}

	d := &Dev{
		i2c: i2c.Dev{Bus: bus, Addr: address},
	}

	if err := d.reset(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Dev) Halt() error {
	return d.reset()
}

func (d *Dev) On(channel uint8) error {
	if !d.isValidChannel(channel) {
		return errInvalidChannel
	}

	_, err := d.i2c.Write([]byte{channel, byte(StateOn)})
	d.state[channel-1] = StateOn
	return err
}

func (d *Dev) Off(channel uint8) error {
	if !d.isValidChannel(channel) {
		return errInvalidChannel
	}

	_, err := d.i2c.Write([]byte{channel, byte(StateOff)})
	d.state[channel-1] = StateOff
	return err
}

func (d *Dev) State(channel uint8) (State, error) {
	if !d.isValidChannel(channel) {
		return 0, errInvalidChannel
	}
	return d.state[channel-1], nil
}

func (d *Dev) AvailableChannels() []uint8 {
	return []uint8{0x01, 0x02, 0x03, 0x04}
}

func (s State) String() string {
	if s == StateOff {
		return "off"
	}
	return "on"
}

func (d *Dev) reset() error {
	for _, channel := range d.AvailableChannels() {
		err := d.Off(channel)
		if err != nil {
			return err
		}
	}
	return nil
}

// Addresses in EP0099 are configured via DIP Switches on the board.
// Up to 4 HATs can be stacked and each one need a different address to
// work.
func isValidAddress(address uint16) error {
	switch address {
	case 0x10, 0x11, 0x12, 0x13:
		return nil
	default:
		return errInvalidAddress
	}
}

func (d *Dev) isValidChannel(channel uint8) bool {
	return channel >= 1 && channel <= 4
}
