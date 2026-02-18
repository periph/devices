// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package bme68x

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
)

// I2CAddr Default I²C address for BME680.
const I2CAddr uint16 = 0x77

// Chip variants
const (
	VariantBME680 uint8 = iota
	VariantBME688
)

// SensorVariant Variant names
type SensorVariant string

const (
	VariantNameBME680 SensorVariant = "BME680"
	VariantNameBME688 SensorVariant = "BME688"
)

// OperatingMode represents the BME680 sensor's power and measurement mode.
const (
	SleepMode  uint8 = iota // SleepMode puts the sensor in low-power standby.
	ForcedMode              // ForcedMode triggers a single measurement and then returns to sleep.
)

// Oversampling bit positions.
const (
	tempOSBit  uint8 = 0x5
	pressOSBit uint8 = 0x2
	humOSBit   uint8 = 0x0
)

// Oversampling options for sensor measurements.
const (
	OSSkipped uint8 = iota
	OS1x
	OS2x
	OS4x
	OS8x
	OS16x
)

// iirFilterBit Bit position.
const iirFilterBit = 0x2

// IIR filter coefficients for smoothing sensor data.
const (
	NoFilter uint8 = iota
	C1Filter       //nolint:unused
	C3Filter
	C7Filter
	C15Filter
	C31Filter
	C63Filter
	C127Filter
)

// GasResistance is returned in Ohms.
type GasResistance uint32

// GasProfile defines one BME680 gas sensor profile (0-9).
type GasProfile struct {
	TargetTempC       uint32 // Heater target temperature in °C
	HeatingDurationMs uint16 // Heating duration in milliseconds
}

// SensorConfig holds all configurable parameters for the BME680 sensor,
// including oversampling, filter, gas sensor profiles, and operating mode.
type SensorConfig struct {
	TempOversampling     uint8          // Temperature oversampling setting
	PressureOversampling uint8          // Pressure oversampling setting
	HumidityOversampling uint8          // Humidity oversampling setting
	IIRFilter            uint8          // IIR filter coefficient
	GasEnabled           bool           // Enable gas measurements
	GasProfiles          [10]GasProfile // Array of Gas sensor profiles
	AmbientTempC         float32        // Ambient temperature for heater calculations
	OperatingMode        uint8          // Sensor operating mode (Sleep/Forced), Default:Sleep
}

var (
	ErrI2cAddress           = errors.New("i2c: provided address is not supported by the device")
	ErrInvalidChipId        = errors.New("bme68x: invalid chip ID")
	ErrInvalidVariantId     = errors.New("bme68x: invalid variant ID")
	ErrNoGasProfileSelected = errors.New("bme68x: no gas profile selected, but gas measurements are enabled")
	ErrRunSetupSensor       = errors.New("bme68x: gas measurement disabled; run SetupSensor()")
	ErrNilSensorConfig      = errors.New("bme680: nil SensorConfig")
)

// Device represents a handle to a BME680 sensor.
type Device struct {
	d                     i2c.Dev           // I²C device handle
	variant               SensorVariant     // Sensor variant identifier
	mutex                 sync.Mutex        // Mutex for concurrent access
	cfg                   SensorConfig      // User-provided configuration
	c                     SensorCalibration // Calibration data
	activeGasProfileIndex int8              // Currently active gas profile index; -1 if none selected
	ops                   sensorOps         // Interface for low-level chip operations (read/write registers, measure)
	ambientTempC          float32           // Ambient temperature used for gas sensor compensation
}

// sensorOps  defines low-level operations implemented by a specific BME68x chip variant.
type sensorOps interface {
	prepareGasConfig() []byte                        // prepares the gas heater configuration buffer
	sense() (physic.Env, GasResistance, bool, error) // triggers a single measurement and returns TPH optionally gas resistance with validity flag
	status() (sensorStatus, error)                   // reads the current sensor status
	setGasProfile(profile uint8) error               // activates a specific gas profile (0-9) without triggering measurement
}

// NewI2C initializes a BME68x sensor over I²C.
func NewI2C(b i2c.Bus, addr uint16) (*Device, error) {
	// Validate I2C Address
	if addr != I2CAddr {
		return nil, ErrI2cAddress
	}
	device := Device{d: i2c.Dev{Bus: b, Addr: addr}}
	// Validate Device and Variant ID
	if err := device.validateDeviceID(); err != nil {
		return nil, err
	}
	// Calibration Initialization - Common for both variant
	if err := device.InitCalibration(); err != nil {
		return nil, err
	}
	return &device, nil
}

// GetSensorVariant  return the type of BME68X sensor connected
func (dev *Device) GetSensorVariant() SensorVariant {
	return dev.variant
}

// SetupSensor configures oversampling, filter, and gas heater profiles.
// Does NOT trigger measurements; call Sense() for actual data. Copies user-provided SensorConfig and validates values.
// Builds a write buffer for control registers and optionally gas heater registers.
func (dev *Device) SetupSensor(config *SensorConfig) error {
	if config == nil {
		return ErrNilSensorConfig
	}
	dev.mutex.Lock()
	defer dev.mutex.Unlock()
	dev.cfg = *config
	dev.activeGasProfileIndex = -1 // default: no active profile
	dev.ambientTempC = dev.cfg.AmbientTempC
	if err := dev.validateOversampling(); err != nil {
		return err
	}
	writeBuf := []byte{
		byte(regCtrlHum), dev.cfg.HumidityOversampling << humOSBit,
		byte(regCtrlMeas), dev.cfg.PressureOversampling<<pressOSBit | dev.cfg.TempOversampling<<tempOSBit,
		byte(regConfig), dev.cfg.IIRFilter << iirFilterBit,
	}
	if dev.cfg.GasEnabled {
		gasBuf := dev.ops.prepareGasConfig()
		if len(gasBuf) > 0 {
			writeBuf = append(writeBuf, gasBuf...)
		}
	}
	return dev.regWrite(writeBuf)
}

// SensorSoftReset performs a software reset of the BME680. (It restores the device to its default state without power cycling)
// Writes the reset command to regReset and waits 20ms for the sensor to reboot.
func (dev *Device) SensorSoftReset() error {
	dev.mutex.Lock()
	defer dev.mutex.Unlock()
	writeBuf := []byte{byte(regReset), byte(DeviceSoftReset)}
	if err := dev.regWrite(writeBuf); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	return nil
}

// IsNewMeasurementReady returns true if a new TPHG measurement is available.
// Returns false if reading the status fails.
func (dev *Device) IsNewMeasurementReady() (bool, error) {
	dev.mutex.Lock()
	defer dev.mutex.Unlock()
	status, err := dev.ops.status()
	if err != nil {
		return false, err
	}
	return status.MeasurementReady, nil
}

// ActiveGasProfile returns the currently active gas profile index (0-9).
// Returns -1 if no gas profile is active or reading the status fails.
func (dev *Device) ActiveGasProfile() (int8, error) {
	dev.mutex.Lock()
	defer dev.mutex.Unlock()
	status, err := dev.ops.status()
	if err != nil {
		return -1, err
	}
	return int8(status.GasProfileInProgress), nil
}

// Sense triggers a single forced-mode measurement and returns the compensated data.
// Returns temperature, pressure, humidity, and optionally gas resistance (with validity flag).
// If gas measurements are enabled, a gas profile and GasEnabled must be selected beforehand.
// Blocks until the measurement is ready.
func (dev *Device) Sense() (physic.Env, GasResistance, bool, error) {
	dev.mutex.Lock()
	defer dev.mutex.Unlock()
	return dev.ops.sense()
}

// SetGasProfile sets the active gas profile (0-9) on the BME680 sensor.
// Does not trigger measurement; call Sense() afterwards
// Updates the sensor hardware and the internal activeGasProfileIndex.
func (dev *Device) SetGasProfile(profile uint8) error {
	// Validate profile index
	if profile > 9 {
		return fmt.Errorf("bme680: gas profile index must be 0-9, got %d", profile)
	}
	// Ensure gas measurements are enabled
	if !dev.cfg.GasEnabled {
		return ErrRunSetupSensor
	}
	dev.mutex.Lock()
	defer dev.mutex.Unlock()
	return dev.ops.setGasProfile(profile)
}

// validateDeviceID verifies the BME68x chip ID and variant ID.
// It reads the regID and regVariantID registers and sets the device variant.
// Returns an error if the device is not a BME680 or BME688 (unsupported).
func (dev *Device) validateDeviceID() error {
	var id, vid []byte
	var err error
	if id, err = dev.regRead(regID, 0x1); err != nil {
		return err
	}
	if id[0] != ChipDeviceID {
		return fmt.Errorf("bme68x: invalid chip ID (expected=0x%x, got=0x%x): %w", ChipDeviceID, id[0], ErrInvalidChipId)
	}
	if vid, err = dev.regRead(regVariantID, 0x1); err != nil {
		return err
	}
	switch vid[0] {
	case VariantBME680:
		dev.variant = VariantNameBME680
		dev.ops = dev.newBME680()
	case VariantBME688:
		dev.variant = VariantNameBME688
		return fmt.Errorf("bme68x: BME688 support not implemented yet")
	default:
		return fmt.Errorf("bme68x: invalid variant ID (expected=0 or 1, got=0x%x): %w", vid[0], ErrInvalidVariantId)
	}
	return nil
}

// triggerForcedMeasurement starts a forced-mode measurement on the BME680,
// waits for it to complete based on sensor configuration (TPH oversampling, gas heater, etc.),
// and ensures the data is ready to read.
func (dev *Device) triggerForcedMeasurement() error {
	// Ensure device is in forced mode
	if dev.cfg.OperatingMode != ForcedMode {
		dev.cfg.OperatingMode = ForcedMode
	}
	// Set forced mode to start measurement
	if err := dev.regWrite([]byte{byte(regCtrlMeas),
		dev.cfg.PressureOversampling<<pressOSBit | dev.cfg.TempOversampling<<tempOSBit | ForcedMode}); err != nil {
		return err
	}
	// Wait for measurements to complete
	//	time.Sleep(d.forcedModeWaitTime())
	return nil
}

// forcedModeWaitTime computes the time required for a complete forced-mode measurement.
// Takes into account:
//   - Oversampling for Temperature, Pressure, Humidity
//   - TPH switching overhead
//   - Gas measurement overhead
//   - Gas heater duration
//   - Wake-up time
//   - Gas heating duration (if enabled)
//
// Returns the total wait time as time.Duration
func (dev *Device) forcedModeWaitTime() time.Duration {
	cycles := dev.osCycles(dev.cfg.TempOversampling) +
		dev.osCycles(dev.cfg.PressureOversampling) + dev.osCycles(dev.cfg.HumidityOversampling)
	// Reference : Bosch - BME68x_SensorAPI
	duration := time.Duration(cycles*1963)*time.Microsecond + // TPH switching overhead
		time.Duration(477*(4+5))*time.Microsecond // TPH switch + gas overhead
	// Add Gas Heating Duration if applicable
	if dev.cfg.GasEnabled && dev.activeGasProfileIndex >= 0 {
		duration += time.Duration(dev.cfg.GasProfiles[dev.activeGasProfileIndex].HeatingDurationMs) * time.Millisecond
	}
	return duration
}

// validateOversampling  validates the oversampling threshold
func (dev *Device) validateOversampling() error {
	if dev.cfg.TempOversampling > OS16x {
		return fmt.Errorf("invalid temperature oversampling value: %d", dev.cfg.TempOversampling)
	}
	if dev.cfg.PressureOversampling > OS16x {
		return fmt.Errorf("invalid pressure oversampling value: %d", dev.cfg.PressureOversampling)
	}
	if dev.cfg.HumidityOversampling > OS16x {
		return fmt.Errorf("invalid humidity oversampling value: %d", dev.cfg.HumidityOversampling)
	}
	return nil
}

// osCycles  helper to return the measurement cycles as per oversampling
func (dev *Device) osCycles(os uint8) int {
	osToMeasureCycles := [...]int{0, 1, 2, 4, 8, 16}
	if int(os) >= len(osToMeasureCycles) {
		return 0 // safe fallback
	}
	return osToMeasureCycles[os]
}
