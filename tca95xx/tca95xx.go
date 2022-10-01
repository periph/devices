// Copyright 2022 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package tca95xx provides an interface to the Texas Instruments TCA95 series
// of 8-bit I²C extenders.
//
// The following variants are supported:
//
//   - PCA9536 - address: 0x41
//   - TCA6408A - addresses: 0x20, 0x21
//   - TCA6416 - addresses: 0x20, 0x21
//   - TCA6416A - addresses: 0x20, 0x21
//   - TCA9534 - addresses: 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27
//   - TCA9534A - addresses: 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f
//   - TCA9535 - addresses: 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27
//   - TCA9537 - address: 0x49
//   - TCA9538 - address: 0x70, 0x71, 0x72, 0x73
//   - TCA9539 - address: 0x74, 0x75, 0x76, 0x77
//   - TCA9554 - addresses: 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27
//   - TCA9555 - addresses: 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27
//
// Both gpio.Pin and conn.Conn interfaces are supported.
package tca95xx

import (
	"fmt"
	"strconv"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/i2c"
)

// Dev is a TCA95xx series I²C extender with two ways to interact with the pins
// on the extender chip - as per pin gpio.Pin, or as per port conn.Conn
// connections.
type Dev struct {
	Pins  [][]Pin     // Pins is a double array structured as: [port][pin].
	Conns []conn.Conn // Conns uses the same [port] array structure.
}

// New returns a device object that communicates over I²C to the TCA95xx device
// family of I/O extenders.
func New(bus i2c.Bus, variant Variant, addr uint16) (*Dev, error) {
	v, found := variants[variant]
	if !found {
		return nil, fmt.Errorf("%s: Unsupported variant", string(variant))
	}
	if v.isAddrInvalid(addr) {
		return nil, fmt.Errorf("tca95xx: address not supported by device type %s", string(variant))
	}

	i2c := i2c.Dev{
		Bus:  bus,
		Addr: addr,
	}

	devicename := string(variant) + "_" + strconv.FormatInt(int64(addr), 16)
	ports := v.getPorts(&i2c, devicename)

	// Map the register maps and ports into gpio.Pins.
	pins := make([][]Pin, len(ports))
	pinsLeft := v.pins
	for i := range ports {
		// pre-cache iodir
		_, err := ports[i].iodir.readValue(false)
		if err != nil {
			return nil, err
		}
		if pinsLeft > 8 {
			pins[i] = ports[i].pins(8)
			pinsLeft -= 8
		} else {
			pins[i] = ports[i].pins(pinsLeft)
			pinsLeft = 0
		}
		for j := range pins[i] {
			pin := pins[i][j]
			// Ignore registration failure.
			_ = gpioreg.Register(pin)
		}
	}

	// Convert to an array of Conn interfaces.
	var conns []conn.Conn
	for i := range ports {
		conns = append(conns, ports[i])
	}

	d := Dev{
		Pins:  pins,
		Conns: conns,
	}

	return &d, nil
}

// Close removes any registration to the device.
func (d *Dev) Close() error {
	for _, port := range d.Pins {
		for _, pin := range port {
			err := gpioreg.Unregister(pin.Name())
			if err != nil {
				return err
			}
		}
	}
	return nil
}
