// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import "encoding/binary"

// Status represents the status of the sensor's various components, indicating
// whether or not each is in an error state.
//
// If a status value is sticky then it remains set even if the error disappears
// or if the sensor leaves measurement mode. The status will only be reset by
// [Dev.ReadAndClearDeviceStatus] or by a reset, either by calling [Dev.DeviceReset]
// or power cycling the sensor.
type Status struct {
	// FanSpeedErr is the fan speed error status. A value of true indicates that the fan
	// speed is too high or too low.
	//
	// Applies to: SEN62, SEN63C, SEN65, SEN66, SEN68, SEN69C
	//
	// Sticky: No
	//
	// Description from the datasheet:
	//
	// Fan is switched on, but its speed is more than 10% off the target speed
	// for multiple consecutive measurement intervals. During the first 10 seconds
	// after starting the measurement, the fan speed is not checked (settling time).
	// Very low or very high ambient temperature could trigger this warning during
	// startup. If this flag is set constantly, it might indicate a problem with
	// the power supply or with the fan, and the measured PM values might be wrong.
	// This flag is automatically cleared as soon as the measured speed is within 10%
	// of the target speed or when leaving the measurement mode.
	// Can occur only in measurement mode.
	FanSpeedErr bool

	// FanErr is the fan error status.
	//
	// Applies to: SEN62, SEN63C, SEN65, SEN66, SEN68, SEN69C
	//
	// Sticky: Yes
	//
	// Description from the datasheet:
	//
	// Fan is switched on, but 0 RPM is measured for multiple consecutive measurement
	// intervals. This can occur if the fan is mechanically blocked or broken. Note
	// that the measured values are most likely wrong if this error is reported.
	// Can occur only in measurement mode.
	FanErr bool

	// PMSensorErr is the particulate matter sensor error status.
	//
	// Applies to: SEN62, SEN63C, SEN65, SEN66, SEN68, SEN69C
	//
	// Sticky: Yes
	//
	// Description from the datasheet:
	//
	// Error related to the PM sensor. The particulate matter values might be unknown
	// or wrong if this flag is set, [and] relative humidity and temperature values
	// might be out of specs due to compensation algorithms depending on PM sensor state.
	// Can occur only in measurement mode.
	PMSensorErr bool

	// RHTSensorErr is the relative humidity and temperature sensor error status.
	//
	// Applies to: SEN62, SEN63C, SEN65, SEN66, SEN68, SEN69C
	//
	// Sticky: Yes
	//
	// Description from the datasheet:
	//
	// Error related to the RH&T sensor. The temperature and humidity values might be
	// unknown or wrong if this flag is set, and other measured values might be out of
	// specs due compensation algorithms depending on RH&T sensor values.
	// Can occur only in measurement mode.
	RHTSensorErr bool

	// CO2SensorErr is the CO2 sensor error status.
	//
	// Applies to: SEN63C, SEN66, SEN69C
	//
	// Sticky: Yes
	//
	// Description from the datasheet:
	//
	// Error related to the CO2 sensor. The CO2 values might be unknown or wrong if
	// this flag is set, [and] relative humidity and temperature values might be out of
	// specs due to compensation algorithms depending on CO2 sensor state.
	// Can occur only in measurement mode.
	CO2SensorErr bool

	// GasSensorErr is the gas (VOC and NOx) sensor error status.
	//
	// Applies to: SEN65, SEN66, SEN68, SEN69C
	//
	// Sticky: Yes
	//
	// Description from the datasheet:
	//
	// Error related to the gas sensor. The VOC index and NOx index might be unknown
	// or wrong if this flag is set, [and] relative humidity and temperature values
	// might be out of specs due to compensation algorithms depending on gas sensor state.
	// Can occur only in measurement mode.
	GasSensorErr bool

	// HCHOSensorErr is the formaldehyde sensor error status.
	//
	// Applies to: SEN68, SEN69C
	//
	// Sticky: Yes
	//
	// Description from the datasheet:
	//
	// Error related to the formaldehyde sensor. The formaldehyde values might be
	// unknown or wrong if this flag is set, [and] relative humidity and temperature
	// values might be out of specs due to compensation algorithms depending on
	// formaldehyde sensor state.
	// Can occur only in measurement mode.
	HCHOSensorErr bool
}

func (s *Status) AnyErr() bool {
	return s.FanSpeedErr ||
		s.FanErr ||
		s.PMSensorErr ||
		s.RHTSensorErr ||
		s.CO2SensorErr ||
		s.GasSensorErr ||
		s.HCHOSensorErr
}

// ReadDeviceStatus reads and decodes the current value of the device status register.
func (d *Dev) ReadDeviceStatus() (*Status, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdReadDeviceStatus, nil)
	if err != nil {
		return nil, err
	}

	return d.statusFromRegister(binary.BigEndian.Uint32(data)), nil
}

// ReadAndClearDeviceStatus reads the current device status (like [Dev.ReadDeviceStatus])
// and afterwards clears all flags.
func (d *Dev) ReadAndClearDeviceStatus() (*Status, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdReadAndClearDeviceStatus, nil)
	if err != nil {
		return nil, err
	}

	return d.statusFromRegister(binary.BigEndian.Uint32(data)), nil
}

func (d *Dev) statusFromRegister(register uint32) *Status {
	status := &Status{
		FanSpeedErr:  registerBitBool(register, 21),
		FanErr:       registerBitBool(register, 4),
		PMSensorErr:  registerBitBool(register, 11),
		RHTSensorErr: registerBitBool(register, 6),
	}

	if d.model.hasCO2() {
		if d.model == SEN66 {
			status.CO2SensorErr = registerBitBool(register, 9)
		} else {
			// SEN63C and SEN69C.
			status.CO2SensorErr = registerBitBool(register, 12)
		}
	}

	if d.model.hasVOCNOx() {
		status.GasSensorErr = registerBitBool(register, 7)
	}

	if d.model.hasHCHO() {
		status.HCHOSensorErr = registerBitBool(register, 10)
	}

	return status
}

func registerBitBool(register uint32, bit uint) bool {
	return uint8((register>>bit)&1) == 1
}
