// Copyright 2023 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package ina260

import (
	"periph.io/x/conn/v3/i2c"
)

const (
	INA260_CHIP_ID     uint8 = 0x40
	INA260_CONFIG      uint8 = 0x00 // CONFIGURATION REGISTER (R/W)
	INA260_CURRENT     uint8 = 0x01 // SHUNT VOLTAGE REGISTER (R)
	INA260_BUSVOLTAGE  uint8 = 0x02 // BUS VOLTAGE REGISTER (R)
	INA260_POWER       uint8 = 0x03 // POWER REGISTER (R)
	INA260_MASK_ENABLE uint8 = 0x06 // MASK ENABLE REGISTER (R/W)
	INA260_ALERT_LIMIT uint8 = 0x07 // ALERT LIMIT REGISTER (R/W)
	INA260_MFG_UID     uint8 = 0xFE // MANUFACTURER UNIQUE ID REGISTER (R)
	INA260_DIE_UID     uint8 = 0xFF // DIE UNIQUE ID REGISTER (R)

)

type Power struct {
	Current float64
	Voltage float64
	Power   float64
}

type ina260 struct {
	Conn  *i2c.Dev
	Power Power
}

func New(bus i2c.Bus) *ina260 {

	dev := &i2c.Dev{Bus: bus, Addr: uint16(INA260_CHIP_ID)}
	power := Power{
		Current: 0,
		Voltage: 0,
		Power:   0,
	}

	ina260 := &ina260{
		Conn:  dev,
		Power: power,
	}

	return ina260
}

func (i *ina260) Read() (Power, error) {

	var power Power

	reg := []byte{byte(INA260_CURRENT)}
	currentBytes := make([]byte, 2)
	if err := i.Conn.Tx(reg, currentBytes); err != nil {
		return power, err
	}

	reg = []byte{byte(INA260_BUSVOLTAGE)}
	voltageBytes := make([]byte, 2)
	if err := i.Conn.Tx(reg, voltageBytes); err != nil {
		return power, err
	}

	reg = []byte{byte(INA260_POWER)}
	powerBytes := make([]byte, 2)
	if err := i.Conn.Tx(reg, powerBytes); err != nil {
		return power, err
	}

	amps := uint16(currentBytes[0])<<8 + uint16(currentBytes[1])
	volts := uint16(voltageBytes[0])<<8 + uint16(voltageBytes[1])
	watts := uint16(powerBytes[0])<<8 + uint16(powerBytes[1])

	power.Voltage = 0.00125 * float64(volts)
	power.Current = 0.00125 * float64(amps)
	power.Power = 0.01 * float64(watts) // 10mW/bit

	return power, nil
}

func ManufacturerId(d *i2c.Dev) (uint16, error) {
	reg := []byte{byte(INA260_MFG_UID)}
	manufacturerBytes := make([]byte, 2)
	if err := d.Tx(reg, manufacturerBytes); err != nil {
		return 0, err
	}
	manufacturerID := uint16(manufacturerBytes[0])<<8 + uint16(manufacturerBytes[1])
	return manufacturerID, nil
}

func DieId(d *i2c.Dev) (uint16, error) {
	reg := []byte{byte(INA260_DIE_UID)}
	dieBytes := make([]byte, 2)
	if err := d.Tx(reg, dieBytes); err != nil {
		return 0, err
	}
	dieID := uint16(dieBytes[0])<<8 + uint16(dieBytes[1])
	return dieID, nil
}
