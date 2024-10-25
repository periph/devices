// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package tic

import (
	"encoding/binary"
)

// getVar8 reads an 8 bit value from the Tic at a given register offset.
func (d *Dev) getVar8(offset Offset) (uint8, error) {
	const length = 1
	buffer, err := d.getSegment(cmdGetVariable, offset, length)
	if err != nil {
		return 0, err
	}

	return buffer[0], nil
}

// getVar16 reads a 16 bit value from the Tic at a given register offset.
func (d *Dev) getVar16(offset Offset) (uint16, error) {
	const length = 2
	buffer, err := d.getSegment(cmdGetVariable, offset, length)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint16(buffer), nil
}

// getVar32 reads a 32 bit value from the Tic at a given register offset.
func (d *Dev) getVar32(offset Offset) (uint32, error) {
	const length = 4
	buffer, err := d.getSegment(cmdGetVariable, offset, length)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(buffer), nil
}

// commandQuick sends a command without additional data.
func (d *Dev) commandQuick(cmd command) error {
	writeBuf := []byte{uint8(cmd)}
	err := d.c.Tx(writeBuf, nil)
	return err
}

// commandW7 sends a command with a 7 bit value. The MSB of val is ignored.
func (d *Dev) commandW7(cmd command, val uint8) error {
	writeBuf := []byte{byte(cmd), val & 0x7F}
	err := d.c.Tx(writeBuf, nil)
	return err
}

// commandW32 sends a command with a 32 bit value.
func (d *Dev) commandW32(cmd command, val uint32) error {
	writeBuf := make([]byte, 5)
	writeBuf[0] = byte(cmd)
	binary.LittleEndian.PutUint32(writeBuf[1:], val) // write the uint32 value

	err := d.c.Tx(writeBuf, nil)
	return err
}

// getSegment sends a command and receives "length" bytes back.
func (d *Dev) getSegment(
	cmd command, offset Offset, length uint,
) ([]byte, error) {
	// Transmit command and offset value
	writeBuf := []byte{byte(cmd), byte(offset)}
	err := d.c.Tx(writeBuf, nil)
	if err != nil {
		return nil, err
	}

	// Read the requested number of bytes
	readBuf := make([]byte, length)
	err = d.c.Tx(nil, readBuf)
	return readBuf, err
}
