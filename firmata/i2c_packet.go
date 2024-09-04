package firmata

const (
	I2CRestartTransmission uint8 = 0b01000000
	I2CModeMask            uint8 = 0b00011000
)

type I2CMode uint8

const (
	I2CModeWrite            I2CMode = 0b00000000
	I2CModeRead             I2CMode = 0b00001000
	I2CModeReadContinuously I2CMode = 0b00010000
	I2CModeStopReading      I2CMode = 0b00011000
)

type I2CPacket struct {
	Register uint8
	Data     []byte
}
