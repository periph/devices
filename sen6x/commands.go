// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"time"
)

// command is a SEN6x I2C command, including execution time
// and the number of bytes sent in response.
type command struct {
	id       uint16
	execTime time.Duration

	// The number of bytes sent by the device in response
	// to the command, including CRC bytes.
	rxDataLen int
}

var (
	// I2C sequence type: Send
	// During measurement: no
	cmdStartContinuousMeasurement = command{
		id:       0x0021,
		execTime: 50 * time.Millisecond,
	}

	// I2C sequence type: Send
	// During measurement: yes
	cmdStopMeasurement = command{
		id:       0x0104,
		execTime: 1400 * time.Millisecond,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdGetDataReady = command{
		id:        0x0202,
		execTime:  20 * time.Millisecond,
		rxDataLen: 3,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadMeasuredValuesSEN62 = command{
		id:        0x04a3,
		execTime:  20 * time.Millisecond,
		rxDataLen: 18,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadMeasuredValuesSEN63C = command{
		id:        0x0471,
		execTime:  20 * time.Millisecond,
		rxDataLen: 21,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadMeasuredValuesSEN65 = command{
		id:        0x0446,
		execTime:  20 * time.Millisecond,
		rxDataLen: 24,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadMeasuredValuesSEN66 = command{
		id:        0x0300,
		execTime:  20 * time.Millisecond,
		rxDataLen: 27,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadMeasuredValuesSEN68 = command{
		id:        0x0467,
		execTime:  20 * time.Millisecond,
		rxDataLen: 27,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadMeasuredValuesSEN69C = command{
		id:        0x04b5,
		execTime:  20 * time.Millisecond,
		rxDataLen: 30,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadMeasuredRawValuesSEN62SEN63C = command{
		id:        0x0492,
		execTime:  20 * time.Millisecond,
		rxDataLen: 6,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadMeasuredRawValuesSEN65SEN68SEN69C = command{
		id:        0x0455,
		execTime:  20 * time.Millisecond,
		rxDataLen: 12,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadMeasuredRawValuesSEN66 = command{
		id:        0x0405,
		execTime:  20 * time.Millisecond,
		rxDataLen: 15,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadNumberConcentrationValues = command{
		id:        0x0316,
		execTime:  20 * time.Millisecond,
		rxDataLen: 15,
	}

	// I2C sequence type: Write
	// During measurement: yes
	cmdSetTemperatureOffsetParams = command{
		id:       0x60b2,
		execTime: 20 * time.Millisecond,
	}

	// I2C sequence type: Write
	// During measurement: no
	cmdSetTemperatureAccelParams = command{
		id:       0x6100,
		execTime: 20 * time.Millisecond,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdGetProductName = command{
		id:        0xd014,
		execTime:  20 * time.Millisecond,
		rxDataLen: 48,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdGetSerialNumber = command{
		id:        0xd033,
		execTime:  20 * time.Millisecond,
		rxDataLen: 48,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadDeviceStatus = command{
		id:        0xd206,
		execTime:  20 * time.Millisecond,
		rxDataLen: 6,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdReadAndClearDeviceStatus = command{
		id:        0xd210,
		execTime:  20 * time.Millisecond,
		rxDataLen: 6,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdGetVersion = command{
		id:        0xd100,
		execTime:  20 * time.Millisecond,
		rxDataLen: 3,
	}

	// I2C sequence type: Send
	// During measurement: no
	cmdDeviceReset = command{
		id:       0xd304,
		execTime: 1200 * time.Millisecond,
	}

	// I2C sequence type: Send
	// During measurement: no
	cmdStartFanCleaning = command{
		id:       0x5607,
		execTime: 20 * time.Millisecond,
	}

	// I2C sequence type: Send
	// During measurement: no
	cmdActivateSHTHeater = command{
		id: 0x6765,
		// Execution time depends on the sensor's firmware version. See
		// [Dev.ActivateSHTHeater] for details.
		execTime: 20 * time.Millisecond,
	}

	// I2C sequence type: Read
	// During measurement: no
	//
	// This command is available only in certain sensor firmware versions.
	// See [Dev.GetSHTHeaterMeasurements] for details.
	cmdGetSHTHeaterMeasurements = command{
		id:        0x6790,
		execTime:  20 * time.Millisecond,
		rxDataLen: 6,
	}

	// I2C sequence type: Read
	// During measurement: no
	cmdGetVOCAlgorithmTuningParamsSEN65SEN66SEN68SEN69C = command{
		id:        0x60d0,
		execTime:  20 * time.Millisecond,
		rxDataLen: 18,
	}

	// I2C sequence type: Write
	// During measurement: no
	cmdSetVOCAlgorithmTuningParamsSEN65SEN66SEN68SEN69C = command{
		id:       0x60d0,
		execTime: 20 * time.Millisecond,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdGetVOCAlgorithmStateSEN65SEN66SEN68SEN69C = command{
		id:        0x6181,
		execTime:  20 * time.Millisecond,
		rxDataLen: 12,
	}

	// I2C sequence type: Write
	// During measurement: no
	cmdSetVOCAlgorithmStateSEN65SEN66SEN68SEN69C = command{
		id:       0x6181,
		execTime: 20 * time.Millisecond,
	}

	// I2C sequence type: Read
	// During measurement: no
	cmdGetNOxAlgorithmTuningParamsSEN65SEN66SEN68SEN69C = command{
		id:        0x60e1,
		execTime:  20 * time.Millisecond,
		rxDataLen: 18,
	}

	// I2C sequence type: Write
	// During measurement: no
	cmdSetNOxAlgorithmTuningParamsSEN65SEN66SEN68SEN69C = command{
		id:       0x60e1,
		execTime: 20 * time.Millisecond,
	}

	// I2C sequence type: Send and read
	// During measurement: no
	cmdPerformForcedCO2RecalibrationSEN63CSEN66SEN69C = command{
		id:        0x6707,
		execTime:  500 * time.Millisecond,
		rxDataLen: 3,
	}

	// I2C sequence type: Send
	// During measurement: no
	//
	// On the SEN66, this command is available only in firmware versions >= 1.2.
	// It is available in all firmware versions on the SEN63C and SEN69C.
	cmdPerformCO2SensorFactoryResetSEN63CSEN66SEN69C = command{
		id:       0x6754,
		execTime: 1400 * time.Millisecond,
	}

	// I2C sequence type: Read
	// During measurement: no
	cmdGetCO2AutoSelfCalibrationSEN63CSEN66SEN69C = command{
		id:        0x6711,
		execTime:  20 * time.Millisecond,
		rxDataLen: 3,
	}

	// I2C sequence type: Write
	// During measurement: no
	cmdSetCO2AutoSelfCalibrationSEN63CSEN66SEN69C = command{
		id:       0x6711,
		execTime: 20 * time.Millisecond,
	}

	// I2C sequence type: Read
	// During measurement: yes
	cmdGetAmbientPressureSEN63CSEN66SEN69C = command{
		id:        0x6720,
		execTime:  20 * time.Millisecond,
		rxDataLen: 3,
	}

	// I2C sequence type: Write
	// During measurement: yes
	cmdSetAmbientPressureSEN63CSEN66SEN69C = command{
		id:       0x6720,
		execTime: 20 * time.Millisecond,
	}

	// I2C sequence type: Read
	// During measurement: no
	cmdGetSensorAltitudeSEN63CSEN66SEN69C = command{
		id:        0x6736,
		execTime:  20 * time.Millisecond,
		rxDataLen: 3,
	}

	// I2C sequence type: Write
	// During measurement: no
	cmdSetSensorAltitudeSEN63CSEN66SEN69C = command{
		id:       0x6736,
		execTime: 20 * time.Millisecond,
	}
)
