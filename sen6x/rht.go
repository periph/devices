// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import "encoding/binary"

type TemperatureOffsetParameters struct {
	// Offset is the constant temperature factor in °C.
	Offset int16

	// Slope is the temperature offset slope.
	Slope int16

	// TimeConstant determines how fast the new slope and offset will be applied.
	//
	// After the specified value in seconds, 63% of the new slope and offset are
	// applied. A time constant of zero means the new values will be applied
	// immediately (within the next measure interval of 1 second).
	TimeConstant uint16

	// Slot is the temperature offset slot to be modified. Valid values are in [0, 4].
	// If the value is outside this range, the parameters will not be applied.
	//
	// A total of five slots are available. Each slot represents one temperature
	// offset. Usually slot 0 is used to compensate for the base self-heating and
	// the other slots allow compensation for additional heating of components like
	// screens, Wi-Fi modules, etc. that can be switched on and off independently.
	Slot uint16
}

func (params TemperatureOffsetParameters) pack() []byte {
	return packWordsWithCRC([]uint16{
		uint16(params.Offset * 200),
		uint16(params.Slope * 10000),
		params.TimeConstant,
		params.Slot,
	})
}

type TemperatureAccelerationParameters struct {
	// Filter constant K.
	K uint16

	// Filter constant P.
	P uint16

	// Time constant T1.
	T1 uint16

	// Time constant T2.
	T2 uint16
}

func (params TemperatureAccelerationParameters) pack() []byte {
	return packWordsWithCRC([]uint16{
		params.K * 10,
		params.P * 10,
		params.T1 * 10,
		params.T2 * 10,
	})
}

type SHTHeaterMeasurements struct {
	// If SHT sensor heating is completed, RH indicates the relative humidity
	// of the SHT4x sensor.
	RH *float32

	// If SHT sensor heating is completed, Temp indicates the temperature in °C
	// of the SHT4x sensor.
	Temp *float32
}

// SetTemperatureOffsetParameters allows for compensation of temperature effects
// due to the design-in of the sensor by applying custom offsets to the ambient
// temperature.
//
// The compensated ambient temperature is calculated as follows:
//
// T_compensated = T_measured + (Slope * T_measured) + C_offset
//
// where Slope and C_offset are set with this command, smoothed with the given
// time constant. All temperatures (T_compensated, T_measured, and C_offset)
// are represented in °C.
//
// There are 5 temperature offset slots available that all contribute additively
// to T_compensated. The default values for the temperature offset parameters are
// all zero, meaning that T_compensated is equal to T_measured by default.
//
// Note: This configuration is volatile, i.e. the parameters will be reverted to
// their default value of zero after a device reset.
//
// For more details on how to compensate the temperature on the SEN6x platform,
// refer to ["SEN6x – Temperature Acceleration and Compensation Instructions"].
//
// ["SEN6x – Temperature Acceleration and Compensation Instructions"]: https://sensirion.com/media/documents/C964FCC8/69709EC3/PS_AN_SEN6x_Temperature_Compensation_and_Acceleration_Application_No.pdf
func (d *Dev) SetTemperatureOffsetParameters(params TemperatureOffsetParameters) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdSetTemperatureOffsetParams, params.pack())
}

// SetTemperatureAccelerationParameters sets custom RH/T engine temperature
// acceleration parameters.
//
// It overwrites the RH/T engine's default temperature acceleration parameters
// with custom values.
//
// Note: This configuration is volatile, i.e. the parameters will be reverted to
// their default values after a device reset.
//
// For more details on how to compensate the temperature on the SEN6x platform,
// refer to ["SEN6x – Temperature Acceleration and Compensation Instructions"].
//
// ["SEN6x – Temperature Acceleration and Compensation Instructions"]: https://sensirion.com/media/documents/C964FCC8/69709EC3/PS_AN_SEN6x_Temperature_Compensation_and_Acceleration_Application_No.pdf
func (d *Dev) SetTemperatureAccelerationParameters(params TemperatureAccelerationParameters) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdSetTemperatureAccelParams, params.pack())
}

// ActivateSHTHeater activates the SHT sensor's built-in heater to reverse
// humidity creep at high humidity. It activates the heater at 200 mW for 1 s,
// after which the heater is deactivated.
//
// "SHT" refers to the SHT4x family of relative humidity and temperature sensors.
// See the [SHT4x datasheet].
//
// For firmware versions listed below, [Dev.GetSHTHeaterMeasurements] may be polled
// to check whether heating is finished in order to trigger another cycle and maximize
// the duty cycle. Older firmware versions do not support [Dev.GetSHTHeaterMeasurements].
//
// Wait at least 20 s after this command before starting a measurement to get
// coherent temperature values (i.e. heating consequence to disappear).
//
// The following firmware versions have a command execution time of 20 ms and
// support [Dev.GetSHTHeaterMeasurements]. Older firmware versions have an execution
// time of 1300 ms:
//   - SEN62 >= 6.0
//   - SEN63C >= 5.0
//   - SEN65 >= 5.0
//   - SEN66 >= 4.0
//   - SEN68 >= 7.0
//   - SEN69C >= 9.0
//
// In this package the execution time for ActivateSHTHeater is set to 20 ms, so if
// you're using a device with firmware version below those listed above then you
// should wait an additional 1280 ms after calling this method.
//
// For more information on humidity creep, see the application note
// ["Using the Integrated Heater of SHT4x in High-Humidity Environments"].
//
// ["Using the Integrated Heater of SHT4x in High-Humidity Environments"]: https://sensirion.com/media/documents/A88858C9/629626D4/Application_Note_Creep_Mitigation_SHT4x.pdf
// [SHT4x datasheet]: https://sensirion.com/media/documents/33FD6951/67EB9032/HT_DS_Datasheet_SHT4x_5.pdf
func (d *Dev) ActivateSHTHeater() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdActivateSHTHeater, nil)
}

// GetSHTHeaterMeasurements gets the measurement values when SHT sensor heating is finished.
//
// "SHT" refers to the SHT4x family of relative humidity and temperature sensors.
// See the [SHT4x datasheet].
//
// This command is available only in these firmware versions:
//   - SEN62 >= 6.0
//   - SEN63C >= 5.0
//   - SEN65 >= 5.0
//   - SEN66 >= 4.0
//   - SEN68 >= 7.0
//   - SEN69C >= 9.0
//
// It must be used after [Dev.ActivateSHTHeater]. It may be polled every 50 ms to
// check if the heating cycle is finished and measurements are available.
//
// For more information on humidity creep, see the application note
// ["Using the Integrated Heater of SHT4x in High-Humidity Environments"].
//
// ["Using the Integrated Heater of SHT4x in High-Humidity Environments"]: https://sensirion.com/media/documents/A88858C9/629626D4/Application_Note_Creep_Mitigation_SHT4x.pdf
// [SHT4x datasheet]: https://sensirion.com/media/documents/33FD6951/67EB9032/HT_DS_Datasheet_SHT4x_5.pdf
func (d *Dev) GetSHTHeaterMeasurements() (*SHTHeaterMeasurements, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdGetSHTHeaterMeasurements, nil)
	if err != nil {
		return nil, err
	}

	hm := &SHTHeaterMeasurements{}

	if rawRH := binary.BigEndian.Uint16(data[0:2]); rawRH != 0x7fff {
		hm.RH = ptr(float32(rawRH) / 100.0)
	}

	if rawTemp := int16(binary.BigEndian.Uint16(data[2:4])); rawTemp != 0x7fff {
		hm.Temp = ptr(float32(rawTemp) / 200.0)
	}

	return hm, nil
}
