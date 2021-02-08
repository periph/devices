// Copyright 2018 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package ep0099

import (
	"errors"

	"periph.io/x/conn/v3/i2c"
)

var InvalidAddressError = errors.New("Invalid EP-0099 address")
var InvalidChannelError = errors.New("Invalid EP-0099 channel")

type State byte

const (
	StateOff State = 0x00
	StateOn  State = 0x01
)

type Dev struct {
	i2c   i2c.Dev
	state map[uint8]State
}

func New(bus i2c.Bus, address uint16) (*Dev, error) {
	if err := isValidAddress(address); err != nil {
		return nil, err
	}

	d := &Dev{
		i2c:   i2c.Dev{Bus: bus, Addr: address},
		state: buildChannelsList(),
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
	if err := d.isValidChannel(channel); err != nil {
		return err
	}

	_, err := d.i2c.Write([]byte{channel, byte(StateOn)})
	d.state[channel] = StateOn
	return err
}

func (d *Dev) Off(channel uint8) error {
	if err := d.isValidChannel(channel); err != nil {
		return err
	}

	_, err := d.i2c.Write([]byte{channel, byte(StateOff)})
	d.state[channel] = StateOff
	return err
}

func (d *Dev) State(channel uint8) (State, error) {
	if err := d.isValidChannel(channel); err != nil {
		return 0, err
	}
	return d.state[channel], nil
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

// Reset resets the registers to the default values.
func (d *Dev) reset() error {
	for channel := range d.state {
		d.Off(channel)
	}
	return nil
}

// Addresses in EP0099 are configured via DIP Switches on the board.
// Up to 4 HATs can be stacked and each one need a different address to
// work.
func isValidAddress(address uint16) error {
	validAddresses := [...]uint16{0x10, 0x11, 0x12, 0x13}

	for _, addr := range validAddresses {
		if address == addr {
			return nil
		}
	}

	return InvalidAddressError
}

func (d *Dev) isValidChannel(channel uint8) error {
	if _, exists := d.state[channel]; !exists {
		return InvalidChannelError
	}
	return nil
}

// EP-0099 offers 4 channels per board
func buildChannelsList() map[uint8]State {
	// Using a map instead of list since indexes of channels are not zero-based
	// values. That would cause loops to have to correct channel ids while
	// looping through items or reading/setting values.
	// With maps, keys correspond to actual channels on the board.
	return map[uint8]State{
		1: StateOff,
		2: StateOff,
		3: StateOff,
		4: StateOff,
	}
}
