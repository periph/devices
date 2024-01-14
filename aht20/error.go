package aht20

// NotInitializedError is returned when the sensor is not initialized but a measurement is requested.
type NotInitializedError struct{}

func (e *NotInitializedError) Error() string {
	return "AHT20 is not initialized."
}

// ReadTimeoutError is returned when the sensor does not finish a measurement in time.
type ReadTimeoutError struct{}

func (e *ReadTimeoutError) Error() string {
	return "Read timeout. AHT20 did not finish measurement in time."
}

// DataCorruptionError is returned when the data from the sensor does not match the CRC8 hash.
type DataCorruptionError struct{}

func (e *DataCorruptionError) Error() string {
	return "Data is corrupt. The CRC8 hashes did not match."
}
