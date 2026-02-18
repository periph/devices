// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package bme68x

import (
	"fmt"
)

// gasRangeR represents the gas sensor range index 0 - 15
type gasRangeR uint8

// gasConstant List of gas ranges and corresponding constants used for the resistance calculation
// Table 16 of Datasheet
var gasConstant = map[gasRangeR][]uint32{
	0:  {2147483647, 4096000000},
	1:  {2147483647, 2048000000},
	2:  {2147483647, 1024000000},
	3:  {2147483647, 512000000},
	4:  {2147483647, 255744255},
	5:  {2126008810, 127110228},
	6:  {2147483647, 64000000},
	7:  {2130303777, 32258064},
	8:  {2147483647, 16016016},
	9:  {2147483647, 8000000},
	10: {2143188679, 4000000},
	11: {2136746228, 2000000},
	12: {2147483647, 1000000},
	13: {2126008810, 500000},
	14: {2147483647, 250000},
	15: {2147483647, 125000},
}

// SensorCalibration holds the calibration coefficients read from a BME680 sensor.
// These values are used internally to convert raw sensor readings into
// accurate temperature, pressure, humidity, and gas measurements.
type SensorCalibration struct {
	t1                                     uint16
	t2                                     int16
	t3                                     int8
	p1                                     uint16
	p2, p4, p5, p8, p9                     int16
	p3, p6, p7                             int8
	p10                                    uint8
	h1, h2                                 uint16
	h3, h4, h5, h6, h7                     int8
	g1, g3                                 int8
	g2                                     int16
	resHeatVal, resHeatRange, switchingErr uint8
	tFine                                  int32
	tempComp                               int32
	pressureComp                           int32
	humidityComp                           int32
}

// InitCalibration loads the sensor's calibration data for accurate measurements.
func (dev *Device) InitCalibration() error {
	// Read all calibration registers
	t, err := dev.regRead(regParT1, 2)
	if err != nil {
		return err
	}
	tp, err := dev.regRead(regParT2, 23)
	if err != nil {
		return err
	}
	h, err := dev.regRead(regParH2, 8)
	if err != nil {
		return err
	}
	g, err := dev.regRead(regParG2, 4)
	if err != nil {
		return err
	}
	r, err := dev.regRead(regResHeatVal, 5)
	if err != nil {
		return err
	}
	// Combined length check
	if len(t) < 2 || len(tp) < 23 || len(h) < 8 || len(g) < 4 || len(r) < 5 {
		return fmt.Errorf("calibration data incomplete: t=%d, tp=%d, h=%d, g=%d, r=%d", len(t), len(tp), len(h), len(g), len(r))
	}
	// Populate calibration struct
	dev.c = SensorCalibration{}
	dev.c.t1 = uint16(t[1])<<8 | uint16(t[0])
	dev.c.t2 = int16(tp[1])<<8 | int16(tp[0])
	dev.c.t3 = int8(tp[2])
	dev.c.p1 = uint16(tp[5])<<8 | uint16(tp[4])
	dev.c.p2 = int16(tp[7])<<8 | int16(tp[6])
	dev.c.p3 = int8(tp[8])
	dev.c.p4 = int16(tp[11])<<8 | int16(tp[10])
	dev.c.p5 = int16(tp[13])<<8 | int16(tp[12])
	dev.c.p7 = int8(tp[14])
	dev.c.p6 = int8(tp[15])
	dev.c.p8 = int16(tp[19])<<8 | int16(tp[18])
	dev.c.p9 = int16(tp[21])<<8 | int16(tp[20])
	dev.c.p10 = tp[22]
	dev.c.h1 = uint16(h[2])<<4 | uint16(h[1]&0xF)
	dev.c.h2 = uint16(h[0])<<4 | (uint16(h[1]&0xF0) >> 4)
	dev.c.h3 = int8(h[3])
	dev.c.h4 = int8(h[4])
	dev.c.h5 = int8(h[5])
	dev.c.h6 = int8(h[6])
	dev.c.h7 = int8(h[7])
	dev.c.g2 = int16(g[1])<<8 | int16(g[0])
	dev.c.g1 = int8(g[2])
	dev.c.g3 = int8(g[3])
	dev.c.resHeatVal = r[0]
	dev.c.resHeatRange = (uint8(r[1]) & 0x30) >> 4
	dev.c.switchingErr = r[4]
	return nil
}

// compensatedTemperature in degrees Celsius, BME680: Refer Integer Section 3.3.1 of Datasheet
func (dev *Device) compensatedTemperature(tempAdc uint32) int32 {
	var1 := (int32(tempAdc) >> 3) - (int32(dev.c.t1) << 1)
	var2 := (var1 * int32(dev.c.t2)) >> 11
	var3 := ((((var1 >> 1) * (var1 >> 1)) >> 12) * (int32(dev.c.t3) << 4)) >> 14
	dev.c.tFine = var2 + var3
	tempComp := ((dev.c.tFine * 5) + 128) >> 8
	dev.c.tempComp = tempComp
	return dev.c.tempComp
}

// compensatedPressure in Pascal, Refer Integer Section 3.3.2 of Datasheet
func (dev *Device) compensatedPressure(pressureAdc uint32) int32 {
	var1 := (dev.c.tFine >> 1) - 64000
	var2 := ((((var1 >> 2) * (var1 >> 2)) >> 11) * int32(dev.c.p6)) >> 2
	var2 = var2 + ((var1 * int32(dev.c.p5)) << 1)
	var2 = (var2 >> 2) + (int32(dev.c.p4) << 16)
	var1 = (((((var1 >> 2) * (var1 >> 2)) >> 13) * (int32(dev.c.p3) << 5)) >> 3) + ((int32(dev.c.p2) * var1) >> 1)
	var1 = var1 >> 18
	var1 = ((32768 + var1) * int32(dev.c.p1)) >> 15
	pressComp := 1048576 - int32(pressureAdc)
	pressComp = (pressComp - (var2 >> 12)) * (int32(3125))
	if pressComp >= (1 << 30) {
		pressComp = (pressComp / var1) << 1
	} else {
		pressComp = (pressComp << 1) / var1
	}
	var1 = (int32(dev.c.p9) * (((pressComp >> 3) * (pressComp >> 3)) >> 13)) >> 12
	var2 = ((pressComp >> 2) * int32(dev.c.p8)) >> 13
	var3 := ((pressComp >> 8) * (pressComp >> 8) * (pressComp >> 8) * int32(dev.c.p10)) >> 17
	dev.c.pressureComp = pressComp + ((var1 + var2 + var3 + (int32(dev.c.p7) << 7)) >> 4)
	return dev.c.pressureComp
}

// compensatedHumidity %RH, Refer Integer Section 3.3.3 of Datasheet
func (dev *Device) compensatedHumidity(humidityAdc uint32) int32 {
	tempScaled := dev.c.tempComp
	var1 := int32(humidityAdc) - int32(dev.c.h1)<<4 - (((tempScaled * int32(dev.c.h3)) / (int32(100))) >> 1)
	var2 := (int32(dev.c.h2) * (((tempScaled * int32(dev.c.h4)) / (int32(100))) + (((tempScaled * ((tempScaled * int32(dev.c.h5)) /
		(int32(100)))) >> 6) / (int32(100))) + (int32(1 << 14)))) >> 10
	var3 := var1 * var2
	var4 := ((int32(dev.c.h6) << 7) + ((tempScaled * int32(dev.c.h7)) / (int32(100)))) >> 4
	var5 := ((var3 >> 14) * (var3 >> 14)) >> 10
	var6 := (var4 * var5) >> 1
	dev.c.humidityComp = (((var3 + var6) >> 10) * (int32(1000))) >> 12
	return dev.c.humidityComp
}

// compensatedGasSensor resistance in Ohms, Refer Integer Section 3.4.1 of Datasheet
func (dev *Device) compensatedGasSensor(gasAdc uint32, gasRange gasRangeR) int32 {
	if dev.variant == VariantNameBME680 {
		var1 := ((1340 + (5 * int64(dev.c.switchingErr))) * (int64(gasConstant[gasRange][0]))) >> 16
		var2 := int64(gasAdc<<15) - int64(1<<24) + var1
		// Prevent Division by Zero
		if var2 == 0 {
			return 0
		}
		gasRes := int32(((int64(gasConstant[gasRange][1]) * var1 >> 9) + (var2 >> 1)) / var2)
		return gasRes
	}
	return 0
}

// multiplicationFactor maps the GAS_WAIT register bits <7:6> to the corresponding
// gas sensor wait time multiplication factor as defined in the datasheet.
// Bits <7:6> | Wait time factor (ms)
var multiplicationFactor = map[uint8]uint8{
	1:  0 << 6, // 00 -> factor 1
	4:  1 << 6, // 01 -> factor 4
	16: 2 << 6, // 10 -> factor 16
	64: 3 << 6, // 11 -> factor 64
}

// gasWaitXCalculationForcedMode time between the beginning of the heat phase and the start of gas sensor resistance conversion
// Bits <7:6> = multiplication factor (1, 4, 16, 64), bits <5:0> = wait time in 1 ms steps(0 to 63/252/1008/4032 depending on factor).
// Refer Section 5.3.3.3 of datasheet
func (dev *Device) gasWaitXCalculationForcedMode(profile uint8) uint8 {
	var waitMS = dev.cfg.GasProfiles[profile].HeatingDurationMs
	var scale uint16 = 0
	switch {
	case waitMS >= 1 && waitMS <= 63:
		scale = 1
	case waitMS >= 64 && waitMS <= 252:
		scale = 4
	case waitMS >= 253 && waitMS <= 1008:
		scale = 16
	case waitMS >= 1009 && waitMS <= 4032:
		scale = 64
	default:
		// BME680: Section 3.3.5 - In practice, approximately 20–30 ms are necessary for the heater to reach
		// the intended target temperature
		scale = 4
		waitMS = 100
	}
	gasWaitX := multiplicationFactor[uint8(scale)] | uint8((waitMS+scale-1)/scale)
	return gasWaitX
}

// resHeatXCalculation convert the target temperature into a device specific target resistance before writing the resulting register code into the sensor
// memory map.
// Refer Section 3.3.5 of datasheet
func (dev *Device) resHeatXCalculation(profile uint8) uint8 {
	// Max limit of Target is 400 °C
	if dev.cfg.GasProfiles[profile].TargetTempC > 400 {
		dev.cfg.GasProfiles[profile].TargetTempC = 400
	}
	var1 := ((int32(dev.cfg.AmbientTempC) * int32(dev.c.g3)) / 10) << 8
	var2 := (int32(dev.c.g1) + 784) * (((((int32(dev.c.g2) + 154009) * int32(dev.cfg.GasProfiles[profile].TargetTempC) * 5) / 100) + 3276800) / 10)
	var3 := var1 + (var2 >> 1)
	var4 := var3 / (int32(dev.c.resHeatRange) + 4)
	var5 := (131 * int32(dev.c.resHeatVal)) + 65536
	resHeatX100 := ((var4 / var5) - 250) * 34
	resHeatX := uint8((resHeatX100 + 50) / 100) // rounds to the nearest integer.
	return resHeatX
}
