// Copyright 2022 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This file is largely a copy of mcp23xxx/registers.go without the spi interface.

package tca95xx

import "periph.io/x/conn/v3/i2c"

type registerCache struct {
	i2c     *i2c.Dev
	address uint8
	got     bool
	cache   uint8
}

func newRegister(i2c *i2c.Dev, address uint8) registerCache {
	return registerCache{
		i2c:     i2c,
		address: address,
		got:     false,
	}
}

func (r *registerCache) readRegister(address uint8) (uint8, error) {
	rx := make([]byte, 1)
	err := r.i2c.Tx([]byte{address}, rx)
	return rx[0], err
}

func (r *registerCache) writeRegister(address uint8, value uint8) error {
	return r.i2c.Tx([]byte{address, value}, nil)
}

func (r *registerCache) readValue(cached bool) (uint8, error) {
	if cached && r.got {
		return r.cache, nil
	}
	v, err := r.readRegister(r.address)
	if err == nil {
		r.got = true
		r.cache = v
	}
	return v, err
}

func (r *registerCache) writeValue(value uint8, cached bool) error {
	if cached && r.got && value == r.cache {
		return nil
	}

	err := r.writeRegister(r.address, value)
	if err != nil {
		return err
	}
	r.got = true
	r.cache = value
	return nil
}

func (r *registerCache) getAndSetBit(bit uint8, value bool, cached bool) error {
	v, err := r.readValue(cached)
	if err != nil {
		return err
	}
	if value {
		v |= 1 << bit
	} else {
		v &= ^(1 << bit)
	}
	return r.writeValue(v, cached)
}

func (r *registerCache) getBit(bit uint8, cached bool) (bool, error) {
	v, err := r.readValue(cached)
	return (v & (1 << bit)) != 0, err
}
