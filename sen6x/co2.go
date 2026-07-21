// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"encoding/binary"
	"errors"
)

// PerformForcedCO2Recalibration executes a forced recalibration (FRC) of the CO2
// signal. It returns the correction value in ppm CO2.
//
// To successfully conduct an accurate FRC the following steps need to be taken:
//  1. Start a measurement with [Dev.StartContinuousMeasurement] and operate the sensor for
//     at least 3 minutes in an environment with homogeneous and constant CO2 concentration.
//     If applicable, the reference value for altitude or pressure compensation must be provided
//     to the sensor beforehand with [Dev.SetSensorAltitude] or [Dev.SetAmbientPressure],
//     respectively.
//  2. Stop the measurement with [Dev.StopMeasurement] and wait at least 1400 ms.
//  3. Call [Dev.PerformForcedCO2Recalibration] with the reference CO2 concentration that
//     the sensor should be set to. The recalibration procedure will take about 500 ms to
//     complete, during which time no other functions can be executed. A return value of
//     0xffff indicates that the FRC has failed, and this method will return a non-nil
//     error in that case.
//
// Note: This configuration is persistent, i.e. the parameters will be retained
// during a device reset or power cycle.
func (d *Dev) PerformForcedCO2Recalibration(refCO2PPM uint16) (int16, error) {
	if !d.model.hasCO2() {
		return 0, errors.New("sen6x: PerformForcedCO2Recalibration requires a CO2-equipped model")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdPerformForcedCO2RecalibrationSEN63CSEN66SEN69C, packWordsWithCRC([]uint16{refCO2PPM}))
	if err != nil {
		return 0, err
	}

	correction := binary.BigEndian.Uint16(data[0:2])
	if correction == 0xffff {
		return 0, errors.New("sen6x: Forced CO2 recalibration failed, got return value of 0xffff")
	}

	// The datasheet specifies that the FRC correction in ppm is equal to the
	// return value - 0x8000 (i.e., int16's max plus 1). The STCC4's datasheet
	// corroborates this and provides the example of a -100 ppm correction:
	// -100 ppm = 32668 - 0x8000. Since the raw correction value returned from
	// the device is a uint16 and 0xffff is reserved to indicate failure, the
	// range of the final computed ppm value is [0 - 0x8000, 0xffff - 1 - 0x8000] =
	// [-32768, 32766], which is within the range of int16. We can therefore
	// safely cast back to int16 after performing the calculation.
	return int16(int32(correction) - 0x8000), nil
}

// PerformCO2SensorFactoryReset resets all CO2 sensor configuration settings stored
// in the EEPROM and erases the forced recalibration (FRC) and automatic self-calibration
// (ASC) algorithm history of the CO2 sensor, restarting the bypass phase. Refer
// to the [datasheet of the STCC4] for more information.
//
// NOTE: On the SEN66, this command is available only on firmware versions >= 1.2.
// It is available in all firmware versions on the SEN63C and SEN69C.
//
// [datasheet of the STCC4]: https://sensirion.com/media/documents/6AED4B15/69295E41/CD_DS_STCC4_D1.pdf
func (d *Dev) PerformCO2SensorFactoryReset() error {
	if !d.model.hasCO2() {
		return errors.New("sen6x: PerformCO2SensorFactoryReset requires a CO2-equipped model")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdPerformCO2SensorFactoryResetSEN63CSEN66SEN69C, nil)
}

// GetCO2SensorAutomaticSelfCalibration gets the status of the CO2 sensor automatic
// self-calibration (ASC). The CO2 sensor supports ASC for long-term stability of
// the CO2 output. It can be enabled or disabled. By default, it is enabled.
func (d *Dev) GetCO2SensorAutomaticSelfCalibration() (bool, error) {
	if !d.model.hasCO2() {
		return false, errors.New("sen6x: GetCO2SensorAutomaticSelfCalibration requires a CO2-equipped model")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdGetCO2AutoSelfCalibrationSEN63CSEN66SEN69C, nil)
	if err != nil {
		return false, err
	}

	return data[1] == 1, nil
}

// SetCO2SensorAutomaticSelfCalibration sets the status of the CO2 sensor automatic
// self-calibration (ASC). The CO2 sensor supports ASC for long-term stability of
// the CO2 output. This feature can be enabled or disabled. By default, it is enabled.
//
// ASC can be disabled for testing under lab conditions where concentrations below
// 400ppm are expected, to avoid an alteration of the baseline. In the field, ASC must
// be enabled and exposure to fresh air (i.e. CO2 concentration at 400 ppm) at least
// once per week is required to reach datasheet specifications.
//
// Note: This configuration is volatile, i.e. it will be reverted to its default
// value after a device reset.
func (d *Dev) SetCO2SensorAutomaticSelfCalibration(enable bool) error {
	if !d.model.hasCO2() {
		return errors.New("sen6x: SetCO2SensorAutomaticSelfCalibration requires a CO2-equipped model")
	}

	// The datasheet breaks down the input into two bytes, the first of which is
	// always 0x00 and the second of which is 0x01 to enable and 0x00 to disable.
	// That's equivalent to a 16-bit word set to 0x0000 or 0x0001.
	var enableWord uint16
	if enable {
		enableWord = 1
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdSetCO2AutoSelfCalibrationSEN63CSEN66SEN69C, packWordsWithCRC([]uint16{enableWord}))
}

// GetAmbientPressure gets the ambient pressure (in hPa) that was set with
// [Dev.SetAmbientPressure]. It is used for pressure compensation by the CO2 sensor.
func (d *Dev) GetAmbientPressure() (uint16, error) {
	if !d.model.hasCO2() {
		return 0, errors.New("sen6x: GetAmbientPressure requires a CO2-equipped model")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdGetAmbientPressureSEN63CSEN66SEN69C, nil)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(data[0:2]), nil
}

// SetAmbientPressure sets the ambient pressure value, in hPa. It is used for
// pressure compensation by the CO2 sensor. Setting an ambient pressure overrides
// any pressure compensation based on a previously set sensor altitude.
//
// Use of this command is recommended for applications experiencing significant
// ambient pressure changes to ensure CO2 sensor accuracy. Valid input values are
// 700 to 1,200 hPa. Device default: 1013 hPa
//
// Note: This configuration is volatile, i.e. the pressure will be reverted to
// its default value after a device reset
func (d *Dev) SetAmbientPressure(hPa uint16) error {
	if !d.model.hasCO2() {
		return errors.New("sen6x: SetAmbientPressure requires a CO2-equipped model")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdSetAmbientPressureSEN63CSEN66SEN69C, packWordsWithCRC([]uint16{hPa}))
}

// GetSensorAltitude gets the current sensor altitude, in meters. It is used for
// pressure compensation by the CO2 sensor.
func (d *Dev) GetSensorAltitude() (uint16, error) {
	if !d.model.hasCO2() {
		return 0, errors.New("sen6x: GetSensorAltitude requires a CO2-equipped model")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdGetSensorAltitudeSEN63CSEN66SEN69C, nil)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(data[0:2]), nil
}

// SetSensorAltitude sets the current sensor altitude, in meters. It is used for
// pressure compensation by the CO2 sensor. The default sensor altitude value is
// 0 meters above sea level. Valid input values are 0 to 3000 m.
//
// Note: This configuration is volatile, i.e. the altitude will be reverted to
// its default value after a device reset.
func (d *Dev) SetSensorAltitude(meters uint16) error {
	if !d.model.hasCO2() {
		return errors.New("sen6x: SetSensorAltitude requires a CO2-equipped model")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdSetSensorAltitudeSEN63CSEN66SEN69C, packWordsWithCRC([]uint16{meters}))
}
