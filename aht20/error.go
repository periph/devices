// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package aht20

import (
	"fmt"
	"time"
)

// NotInitializedError is returned when the sensor is not initialized but a measurement is requested.
type NotInitializedError struct{}

func (e *NotInitializedError) Error() string {
	return "AHT20 is not initialized."
}

// ReadTimeoutError is returned when the sensor does not finish a measurement in time.
type ReadTimeoutError struct {
	// Timeout is the configured timeout.
	Timeout time.Duration
}

func (e *ReadTimeoutError) Error() string {
	return fmt.Sprintf("Read timeout after %s. AHT20 did not finish measurement in time.", e.Timeout)
}

// DataCorruptionError is returned when the data from the sensor does not match the CRC8 hash.
type DataCorruptionError struct {
	// Calculated is the calculated CRC8 hash using the received data bytes.
	Calculated uint8
	// Received is the CRC8 hash received from the sensor.
	Received uint8
}

func (e *DataCorruptionError) Error() string {
	return fmt.Sprintf("Data is corrupt. The CRC8 hashes did not match. Calculated: 0x%X, Received: 0x%X", e.Calculated, e.Received)
}
