// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package bme68x

import (
	"fmt"

	"periph.io/x/conn/v3/physic"
)

// bme680 represents a BME680 sensor and provides methods to perform measurements.
type bme680 struct {
	dev *Device // Pointer to device context (configuration, registers, and helper methods)
}

// adcData stores raw ADC readings and gas sensor status from the BME680.
type adcData struct {
	tRaw, pRaw, hRaw, gRaw uint32    // Raw temperature, pressure, humidity, and gas ADC values
	gResRange              gasRangeR // Gas resistance range (from GAS_R_LSB register)
	gasValid, heaterStable bool      // gas measurement validity and heater stability
}

// sensorStatus represents the current measurement status of the BME680.
type sensorStatus struct {
	MeasurementReady     bool  // True, New TPHG data available (bit 7) of eas_status_0 register
	GasInProgress        bool  // True, Gas measurement in progress (bit 6) of eas_status_0 register
	TPHInProgress        bool  // True, Temp/Pressure/Humidity measurement in progress (bit 5) of eas_status_0 register
	GasProfileInProgress uint8 // Index of Current gas profile (bits 0-2) of eas_status_0 register
}

func (dev *Device) newBME680() sensorOps {
	return &bme680{dev}
}

// Sense triggers a single forced-mode measurement and returns the compensated data(Temperature, pressure, humidity, and optionally gas resistance)
// The BME680 automatically returns to sleep after each measurement.
// If gas measurements are enabled, a gas profile must be selected beforehand.
func (b *bme680) sense() (physic.Env, GasResistance, bool, error) {
	// Check Gas Profile selection
	if !b.dev.cfg.GasEnabled || b.dev.activeGasProfileIndex < 0 {
		return physic.Env{}, 0, false, ErrNoGasProfileSelected
	}
	// Trigger one-shot forced measurement (Blocking until complete)
	if err := b.dev.triggerForcedMeasurement(); err != nil {
		return physic.Env{}, 0, false, err
	}
	return b.readSensorData() // return new measurement
}

// setGasProfile activates a specific gas profile (0-9)
// It does NOT trigger a measurement; call Sense() to measure
func (b *bme680) setGasProfile(profile uint8) error {
	// Enable Gas Point
	writeBuf := []byte{byte(regCtrlGas1), (1 << 4) | (profile)}
	writeBuf = append(writeBuf,
		byte(gasWaitRegs[profile]), b.dev.gasWaitXCalculationForcedMode(profile),
		byte(resHeatRegs[profile]), b.dev.resHeatXCalculation(profile),
	)
	if err := b.dev.regWrite(writeBuf); err != nil {
		return err
	}
	b.dev.activeGasProfileIndex = int8(profile) // Update active profile index
	return nil
}

// readSensorData reads the latest measurement data from the BME680.
// It returns temperature, pressure, humidity, and optionally gas resistance
// along with a validity flag for the gas measurement.
func (b *bme680) readSensorData() (physic.Env, GasResistance, bool, error) {
	// Read raw Temperature, Pressure, Humidity registers (8 bytes)
	tph, err := b.dev.regRead(regPressMSB, 0x8)
	if err != nil {
		return physic.Env{}, 0, false, err
	}
	// Read raw Gas registers (2 bytes)
	g, err := b.dev.regRead(regGasRMsb, 0x2)
	if err != nil {
		return physic.Env{}, 0, false, err
	}
	// Validate raw data length
	if len(tph) < 8 || len(g) < 2 {
		return physic.Env{}, 0, false, fmt.Errorf("bme680: raw sensor data incomplete")
	}
	// Parse raw TPH & gas data into structured adc values
	adc := b.parseRawSensorData(tph, g)
	// Compensate raw sensor values to human-readable units
	return b.compensateSensorValues(adc)
}

// compensateSensorValues converts raw ADC values into compensated
// temperature, pressure, humidity, and gas resistance readings.
func (b *bme680) compensateSensorValues(adc adcData) (physic.Env, GasResistance, bool, error) {
	var env physic.Env
	var gas GasResistance
	// Compensated Temperature (°C)
	tComp := float32(b.dev.compensatedTemperature(adc.tRaw)) // Deg C
	env.Temperature = physic.Temperature(tComp)*10*physic.MilliCelsius + physic.ZeroCelsius
	// Initialize ambient temperature if not given in the user config
	if b.dev.cfg.AmbientTempC == 0.0 {
		b.dev.ambientTempC = float32(env.Temperature.Celsius())
	}
	// Compensate humidity  (%RH)
	hComp := b.dev.compensatedHumidity(adc.hRaw)                                     // %RH × 1000
	env.Humidity = physic.RelativeHumidity(float64(hComp)/1000.0) * physic.PercentRH // %rH
	// Clamp humidity to valid range
	if env.Humidity < 0*physic.PercentRH {
		env.Humidity = 0 * physic.PercentRH
	} else if env.Humidity > 100*physic.PercentRH {
		env.Humidity = 100 * physic.PercentRH
	}
	// Compensate Pressure (Pa)
	pComp := b.dev.compensatedPressure(adc.pRaw)
	env.Pressure = physic.Pressure(pComp) * physic.Pascal
	// Validate Gas measurement
	gasValid := false
	if b.dev.cfg.GasEnabled && adc.gasValid && adc.heaterStable {
		gas = GasResistance(b.dev.compensatedGasSensor(adc.gRaw, adc.gResRange))
		gasValid = true
	}
	// Return compensated sensor readings (temperature, humidity, pressure, and gas resistance)
	return env, gas, gasValid, nil
}

// parseRawSensorData extracts raw ADC fields from register bytes.
func (b *bme680) parseRawSensorData(tph, g []byte) adcData {
	var adc adcData
	adc.tRaw = uint32(tph[3])<<12 | uint32(tph[4])<<4 | (uint32(tph[5])&0xF0)>>4
	adc.pRaw = uint32(tph[0])<<12 | uint32(tph[1])<<4 | (uint32(tph[2])&0xF0)>>4
	adc.hRaw = uint32(tph[6])<<8 | uint32(tph[7])
	adc.gRaw = uint32(g[0])<<2 | uint32(g[1]&0xC0)>>6
	adc.gResRange = gasRangeR(g[1] & 0x0F)
	adc.gasValid = ((g[1] & 0x20) >> 5) == 1
	adc.heaterStable = ((g[1] & 0x10) >> 4) == 1
	return adc
}

// status reads the BME680 status register and returns the current sensor state.
func (b *bme680) status() (sensorStatus, error) {
	var status sensorStatus
	// Read a byte from status register
	s, err := b.dev.regRead(regEASStatus0, 0x1)
	if err != nil {
		return status, err
	}
	// Status bit positions
	const (
		measReadyBit = 7
		gasInProgBit = 6
		tphInProgBit = 5
		gasProfMask  = 0x07 // bits 0-2
	)
	// Mask and shift bits properly
	status.MeasurementReady = ((s[0] >> measReadyBit) & 0x01) == 1
	status.GasInProgress = ((s[0] >> gasInProgBit) & 0x01) == 1
	status.TPHInProgress = ((s[0] >> tphInProgBit) & 0x01) == 1
	status.GasProfileInProgress = s[0] & gasProfMask
	return status, nil
}

// prepareGasConfig constructs the gas heater configuration register sequence for the current configuration.
func (b *bme680) prepareGasConfig() []byte {
	var firstProfileFound = -1
	var gasBuf []byte
	for idx, profile := range b.dev.cfg.GasProfiles {
		if profile.TargetTempC > 0 && firstProfileFound == -1 {
			firstProfileFound = idx // Select the First valid profile
		}
		gasBuf = append(gasBuf, byte(resHeatRegs[idx]), b.dev.resHeatXCalculation(uint8(idx)))
		if b.dev.cfg.OperatingMode == ForcedMode {
			gasBuf = append(gasBuf, byte(gasWaitRegs[idx]), b.dev.gasWaitXCalculationForcedMode(uint8(idx)))
		}
	}
	// Activate the first valid profile if any
	if firstProfileFound != -1 {
		gasBuf = append(gasBuf, byte(regCtrlGas1), byte((1<<4)|(firstProfileFound)))
		b.dev.activeGasProfileIndex = int8(firstProfileFound)
	}
	return gasBuf
}
