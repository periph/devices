// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package bme68x

const (
	DeviceSoftReset = 0x38
	ChipDeviceID    = 0x61
)

type reg uint8

// BME680 Register Addresses
const (
	// Status, ID, reset & Ctl
	regStatus   reg = 0x73 //unused
	regReset    reg = 0xE0
	regID       reg = 0xD0
	regConfig   reg = 0x75
	regCtrlMeas reg = 0x74
	regCtrlHum  reg = 0x72
	regCtrlGas1 reg = 0x71
	regCtrlGas0 reg = 0x70

	// Gas wait time registers
	regGasWait0 reg = 0x64
	regGasWait1 reg = 0x65
	regGasWait2 reg = 0x66
	regGasWait3 reg = 0x67
	regGasWait4 reg = 0x68
	regGasWait5 reg = 0x69
	regGasWait6 reg = 0x6A
	regGasWait7 reg = 0x6B
	regGasWait8 reg = 0x6C
	regGasWait9 reg = 0x6D

	// Heater registers
	regResHeat0 reg = 0x5A
	regResHeat1 reg = 0x5B
	regResHeat2 reg = 0x5C
	regResHeat3 reg = 0x5D
	regResHeat4 reg = 0x5E
	regResHeat5 reg = 0x5F
	regResHeat6 reg = 0x60
	regResHeat7 reg = 0x61
	regResHeat8 reg = 0x62
	regResHeat9 reg = 0x63

	// IDAC heater registers : Retained for future use
	regIDACHeat0 reg = 0x50 //unused
	regIDACHeat1 reg = 0x51 //unused
	regIDACHeat2 reg = 0x52 //unused
	regIDACHeat3 reg = 0x53 //unused
	regIDACHeat4 reg = 0x54 //unused
	regIDACHeat5 reg = 0x55 //unused
	regIDACHeat6 reg = 0x56 //unused
	regIDACHeat7 reg = 0x57 //unused
	regIDACHeat8 reg = 0x58 //unused
	regIDACHeat9 reg = 0x59 //unused

	// Sensor data registers
	regGasRMsb   reg = 0x2A
	regGasRLsb   reg = 0x2B //unused
	regHumMSB    reg = 0x25 //unused
	regHumLSB    reg = 0x26 //unused
	regTempXLSB  reg = 0x24 //unused
	regTempLSB   reg = 0x23 //unused
	regTempMSB   reg = 0x22 //unused
	regPressXLSB reg = 0x21 //unused
	regPressLSB  reg = 0x20 //unused
	regPressMSB  reg = 0x1F

	// Extended status and variant ID
	regEASStatus0 reg = 0x1D
	regVariantID  reg = 0xF0

	// Temperature calibration registers
	regParT1 reg = 0xE9 // uint16, LSB @ 0xE9, MSB @ 0xEA
	regParT2 reg = 0x8A // int16,  LSB @ 0x8A, MSB @ 0x8B
	regParT3 reg = 0x8C // int8

	// Pressure calibration registers
	regParP1  reg = 0x8E // uint16, LSB @ 0x8E / MSB @ 0x8F
	regParP2  reg = 0x90 // int16  LSB @ 0x90 / MSB @ 0x91
	regParP3  reg = 0x92 // int8
	regParP4  reg = 0x94 // int16  LSB @ 0x94 / MSB @ 0x95
	regParP5  reg = 0x96 // int16 LSB @ 0x96 / MSB @ 0x97
	regParP6  reg = 0x99 // int8
	regParP7  reg = 0x98 // int8
	regParP8  reg = 0x9C // int16 LSB @ 0x9C / MSB @ 0x9D
	regParP9  reg = 0x9E // int16 LSB @ 0x9E / MSB @ 0x9F
	regParP10 reg = 0xA0 // uint8

	// Humidity calibration (packed) registers
	regParH2 reg = 0xE1 // H2[11:4] LSB @ 0xE2<7:4> / MSB @ 0xE1
	regParH1 reg = 0xE3 // H1[11:4] LSB @ 0xE2<3:0> / MSB @ 0xE3
	regParH3 reg = 0xE4 // int8
	regParH4 reg = 0xE5 // int8
	regParH5 reg = 0xE6 // int8
	regParH6 reg = 0xE7 // uint8
	regParH7 reg = 0xE8 // int8

	// Gas calibration registers
	regParG2 reg = 0xEB // int16 LSB @ 0xEB / MSB @ 0xEC
	regParG1 reg = 0xED // int8
	regParG3 reg = 0xEE // int8

	// Heater calibration (special registers)
	regResHeatVal          reg = 0x00
	regResHeatRange        reg = 0x02 // <5:4>
	regRangeSwitchingError reg = 0x04
)

// Global slices for gas sensor registers
var gasWaitRegs = []reg{
	regGasWait0, regGasWait1, regGasWait2, regGasWait3, regGasWait4,
	regGasWait5, regGasWait6, regGasWait7, regGasWait8, regGasWait9,
}

var resHeatRegs = []reg{
	regResHeat0, regResHeat1, regResHeat2, regResHeat3, regResHeat4,
	regResHeat5, regResHeat6, regResHeat7, regResHeat8, regResHeat9,
}

// regWrite writes a sequence of bytes to the device
func (dev *Device) regWrite(b []byte) error {
	if err := dev.d.Tx(b, nil); err != nil {
		return err
	}
	return nil
}

// regRead  reads a sequence of bytes from a given register
func (dev *Device) regRead(addr reg, length uint8) ([]byte, error) {
	readBuf := make([]byte, length)
	if err := dev.d.Tx([]byte{byte(addr)}, readBuf); err != nil {
		return nil, err
	}
	return readBuf, nil
}
