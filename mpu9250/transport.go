// Copyright 2020 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package mpu9250

import (
	"fmt"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
)

// DebugF the debug function type.
type DebugF func(string, ...interface{})

// Transport Encapsulates the SPI transport parameters.
type Transport struct {
	device spi.Conn
	d      *i2c.Dev
	cs     gpio.PinOut
	debug  DebugF
}

// NewSpiTransport Creates the SPI transport using the provided device path and chip select pin reference.
func NewSpiTransport(path string, cs gpio.PinOut) (*Transport, error) {
	dev, err := spireg.Open(path)
	if err != nil {
		return nil, wrapf("can't open SPI %v", err)
	}
	conn, err := dev.Connect(1*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return nil, wrapf("can't initialize SPI %v", err)
	}
	return &Transport{device: conn, cs: cs, debug: noop}, nil
}

func NewI2cTransport(bus i2c.Bus, address uint16) (*Transport, error) {
	return &Transport{d: &i2c.Dev{Bus: bus, Addr: address}, debug: noop}, nil
}

// EnableDebug Sets the debugging output using the local print function.
func (t *Transport) EnableDebug(f DebugF) {
	t.debug = f
}

func (t *Transport) writeByte(address byte, value byte) error {
	if t.d == nil {
		return t.writeByteSPI(address, value)
	}
	return t.writeByteI2C(address, value)
}

func (t *Transport) writeByteSPI(address, value byte) error {
	t.debug("write register %x value %x", address, value)
	var (
		buf = [...]byte{address, value}
		res [2]byte
	)
	if err := t.cs.Out(gpio.Low); err != nil {
		return err
	}
	if err := t.device.Tx(buf[:], res[:]); err != nil {
		return err
	}
	return t.cs.Out(gpio.High)
}

func (t *Transport) writeByteI2C(address, value byte) error {
	w := []byte{address, value}
	return t.d.Tx(w, nil)
}

func (t *Transport) writeMaskedReg(address byte, mask byte, value byte) error {
	t.debug("write masked %x, mask %x, value %x", address, mask, value)
	maskedValue := mask & value
	t.debug("masked value %x", maskedValue)
	regVal, err := t.readByte(address)
	if err != nil {
		return err
	}
	t.debug("current register %x", regVal)
	regVal = (regVal &^ maskedValue) | maskedValue
	t.debug("new value %x", regVal)
	return t.writeByte(address, regVal)
}

func (t *Transport) readMaskedReg(address byte, mask byte) (byte, error) {
	t.debug("read masked %x, mask %x", address, mask)
	reg, err := t.readByte(address)
	if err != nil {
		return 0, err
	}
	t.debug("masked value %x", reg)
	return reg & mask, nil
}

func (t *Transport) readByte(address byte) (byte, error) {
	if t.d == nil {
		return t.readByteSPI(address)
	}
	return t.readByteI2C(address)
}

func (t *Transport) readByteSPI(address byte) (byte, error) {
	t.debug("read register %x", address)
	var (
		buf = [...]byte{0x80 | address, 0}
		res [2]byte
	)
	if err := t.cs.Out(gpio.Low); err != nil {
		return 0, err
	}
	if err := t.device.Tx(buf[:], res[:]); err != nil {
		return 0, err
	}
	t.debug("register content %x:%x", res[0], res[1])
	if err := t.cs.Out(gpio.High); err != nil {
		return 0, err
	}
	return res[1], nil
}

func (t *Transport) readByteI2C(address byte) (byte, error) {
	r := make([]byte, 1)
	err := t.d.Tx([]byte{address}, r)
	return r[0], err
}

func (t *Transport) readUint16(address ...byte) (uint16, error) {
	if len(address) != 2 {
		return 0, fmt.Errorf("only 2 bytes per read")
	}
	h, err := t.readByte(address[0])
	if err != nil {
		return 0, err
	}
	l, err := t.readByte(address[1])
	if err != nil {
		return 0, err
	}
	return uint16(h)<<8 | uint16(l), nil
}

func noop(string, ...interface{}) {}
