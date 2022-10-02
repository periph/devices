// Copyright 2022 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package tca95xx

import (
	"periph.io/x/conn/v3/i2c"
)

// Variant is the type denoting a specific variant of the family.
type Variant string

const (
	PCA9536  Variant = "PCA9536"  // PCA9536  8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/pca9536
	TCA6408A Variant = "TCA6408A" // TCA6408A 8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/tca6408a
	TCA6416  Variant = "TCA6416"  // TCA6416  8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/tca6416
	TCA6416A Variant = "TCA6416A" // TCA6416A 8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/tca6416a
	TCA9534  Variant = "TCA9534"  // TCA9534  8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/tca9534
	TCA9534A Variant = "TCA9534A" // TCA9534A 8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/tca9534a
	TCA9535  Variant = "TCA9535"  // TCA9535  8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/tca9535
	TCA9537  Variant = "TCA9537"  // TCA9537  8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/tca9537
	TCA9538  Variant = "TCA9538"  // TCA9538  8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/tca9538
	TCA9539  Variant = "TCA9539"  // TCA9539  8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/tca9539
	TCA9554  Variant = "TCA9554"  // TCA9554  8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/tca9554
	TCA9555  Variant = "TCA9555"  // TCA9555  8-bit I²C extender. Datasheet: https://www.ti.com/lit/gpn/tca9555
)

type variant struct {
	addStart uint16
	addEnd   uint16
	pins     int
}

var variants = map[Variant]variant{
	PCA9536:  {addStart: 0x41, addEnd: 0x41, pins: 4},
	TCA6408A: {addStart: 0x20, addEnd: 0x21, pins: 8},
	TCA6416:  {addStart: 0x20, addEnd: 0x21, pins: 16},
	TCA6416A: {addStart: 0x20, addEnd: 0x21, pins: 16},
	TCA9534:  {addStart: 0x20, addEnd: 0x27, pins: 8},
	TCA9534A: {addStart: 0x38, addEnd: 0x3f, pins: 8},
	TCA9535:  {addStart: 0x20, addEnd: 0x27, pins: 16},
	TCA9537:  {addStart: 0x49, addEnd: 0x49, pins: 4},
	TCA9538:  {addStart: 0x70, addEnd: 0x73, pins: 8},
	TCA9539:  {addStart: 0x74, addEnd: 0x77, pins: 16},
	TCA9554:  {addStart: 0x20, addEnd: 0x27, pins: 8},
	TCA9555:  {addStart: 0x20, addEnd: 0x27, pins: 16},
}

// isAddrInvalid checks to see if the address is used by the chip.
func (v variant) isAddrInvalid(addr uint16) bool {
	if addr < v.addStart || v.addEnd < addr {
		return true
	}
	return false
}

// getVariantRegMap returns the register map based on the number of pins the
// chip expands to.
func (v variant) getPorts(i2c *i2c.Dev, devicename string) []*port {
	if v.pins == 16 {
		return []*port{
			{
				name:   devicename + "_P0",
				input:  newRegister(i2c, 0x00),
				output: newRegister(i2c, 0x02),
				ipol:   newRegister(i2c, 0x04),
				iodir:  newRegister(i2c, 0x06),
			},
			{
				name:   devicename + "_P1",
				input:  newRegister(i2c, 0x01),
				output: newRegister(i2c, 0x03),
				ipol:   newRegister(i2c, 0x05),
				iodir:  newRegister(i2c, 0x07),
			},
		}
	}

	return []*port{
		{
			name:   devicename + "_P0",
			input:  newRegister(i2c, 0x00),
			output: newRegister(i2c, 0x01),
			ipol:   newRegister(i2c, 0x02),
			iodir:  newRegister(i2c, 0x03),
		},
	}
}
