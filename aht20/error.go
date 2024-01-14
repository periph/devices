package aht20

type NotInitializedError struct{}

func (e *NotInitializedError) Error() string {
	return "AHT20 is not initialized."
}

type ReadTimeoutError struct{}

func (e *ReadTimeoutError) Error() string {
	return "Read timeout. AHT20 did not finish measurement in time."
}

type DataCorruptionError struct{}

func (e *DataCorruptionError) Error() string {
	return "Data is corrupt. The CRC8 hashes did not match."
}
