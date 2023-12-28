package adxl345

import (
	"encoding/binary"
	"fmt"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
)

type Sensitivity byte

const (
	DeviceID = 0x00 // Device ID, expected to be 0xE5 when using ADXL345

	S2G  Sensitivity = 0x00 // Sensitivity at 2g
	S4G  Sensitivity = 0x01 // Sensitivity at 4g
	S8G  Sensitivity = 0x02 // Sensitivity at 8g
	S16G Sensitivity = 0x03 // Sensitivity at 16g

	ThreshTap        = 0x1D // Tap threshold
	OfsX             = 0x1E // X-axis offset
	OfsY             = 0x1F // Y-axis offset
	OfsZ             = 0x20 // Z-axis offset
	Dur              = 0x21 // Tap duration
	Latent           = 0x22 // Tap latency
	Window           = 0x23 // Tap window
	ThreshAct        = 0x24 // Activity threshold
	ThreshInact      = 0x25 // Inactivity threshold
	TimeInact        = 0x26 // Inactivity time
	ActInactCtl      = 0x27 // Axis control for activity/inactivity detection
	ThreshFf         = 0x28 // Free-fall threshold
	TapAxes          = 0x2A // Axis control for single tap/double tap
	TapStatus        = 0x2B // Source of single tap/double tap
	ActivityStatus   = 0x2A // Source of activity detection
	InactivityStatus = 0x2B // Source of inactivity detection

	// Control registers

	BwRate     = 0x2C // Data rate and power mode control
	PowerCtl   = 0x2D // Power saving features control
	IntEnable  = 0x2E // Interrupt enable control
	IntMap     = 0x2F // Interrupt mapping control
	IntSource  = 0x30 // Source of interrupts
	DataFormat = 0x31 // Data format control

	// Data registers
	DataX0 = 0x32 // X-Axis Data 0
	DataX1 = 0x33 // X-Axis Data 1
	DataY0 = 0x34 // Y-Axis Data 0
	DataY1 = 0x35 // Y-Axis Data 1
	DataZ0 = 0x36 // Z-Axis Data 0
	DataZ1 = 0x37 // Z-Axis Data 1

	// FIFO control
	FifoCtl    = 0x38 // FIFO control
	FifoStatus = 0x39 // FIFO status

)

// DefaultSpiFrequency is the default SPI frequency used to communicate with the device.
var (
	SpiFrequency = physic.KiloHertz * 50
	SpiMode      = spi.Mode3 // Defines the base clock signal, along with the polarity and phase of the data signal.
	SpiBits      = 8
)

var DefaultOpts = Opts{
	TurnOnOnStart:    true,
	ExpectedDeviceID: 0xE5,
	Sensitivity:      S2G,
}

type Opts struct {
	TurnOnOnStart    bool        // Turn on the device in measurement mode on start.
	ExpectedDeviceID byte        // Expected device ID used to verify that the device is an ADXL345.
	Sensitivity      Sensitivity // Sensitivity of the device (2G, 4G, 8G, 16G)
}

// Dev is a driver for the ADXL345 accelerometer
// It uses the SPI interface to communicate with the device.
type Dev struct {
	name string
	s    spi.Conn
}

func (d *Dev) String() string {
	return fmt.Sprintf("ADXL345{Sensitivity:%d}", d.Sensitivity())
}

// New creates a new ADXL345 Dev or returns an error.
// The bus and chip parameters define the SPI bus and chip select to use.
// The SPI s is configured.
// The device is turned on.
// The device is verified to be an ADXL345.
func New(p spi.Port, o *Opts) (*Dev, error) {
	// Convert the spi.Port into a spi.Conn so it can be used for communication.
	c, err := p.Connect(SpiFrequency, SpiMode, SpiBits)
	if err != nil {
		return nil, err
	}
	d := &Dev{
		name: "ADXL345",
		s:    c,
	}
	if o.TurnOnOnStart {
		err = d.TurnOn()
		if err != nil {
			return nil, err
		}
	}
	if o.Sensitivity != S2G { // default
		err = d.setSensitivity(o.Sensitivity)
		if err != nil {
			return nil, err
		}
	}
	// Verify that the device is an ADXL345.
	rx, _ := d.RawReadRegister(DeviceID)
	if rx[1] != o.ExpectedDeviceID {
		return nil, fmt.Errorf("wrong device connected should be an adxl345  should be\"%#x\" rx0=\"%#x\" rx1=\"%#x\"", o.ExpectedDeviceID, rx[0], rx[1])
	}
	return d, nil
}

// SetSensitivity sets the sensitivity of the ADXL345.
// The sensitivity parameter should be one of 2, 4, 8, or 16, representing ±2g, ±4g, ±8g, or ±16g respectively.
func (d *Dev) setSensitivity(sensitivity Sensitivity) error {
	switch sensitivity {
	case S2G, S4G, S8G, S16G:
		// Write to the DataFormat register
		return d.WriteRegister(DataFormat, byte(sensitivity))
	default:
		return fmt.Errorf("invalid sensitivity: %d. Valid values are 2, 4, 8, 16", sensitivity)
	}
}

func (d *Dev) Sensitivity() Sensitivity {
	rx, _ := d.RawReadRegister(DataFormat)
	return Sensitivity(rx[1])
}

// TurnOn turns on the measurement mode of the ADXL345.
// This is required before reading data from the device.
func (d *Dev) TurnOn() error {
	return d.WriteRegister(PowerCtl, 0x08)
}

// TurnOff turns off the measurement mode of the ADXL345.
func (d *Dev) TurnOff() error {
	return d.WriteRegister(PowerCtl, 0x00)
}

// Update reads the acceleration values from the ADXL345.
// By reading the acceleration the 3 axes acceleration values.
// This is a simple synchronous implementation.
func (d *Dev) Update() Acceleration {
	return Acceleration{
		X: d.readAndCombine(DataX0, DataX1),
		Y: d.readAndCombine(DataY0, DataY1),
		Z: d.readAndCombine(DataZ0, DataZ1),
	}
}

// readAndCombine combines two registers to form a 16-bit value.
// Example:
// `DATAX0` is the address of the lower byte (LSB, least significant byte) of the X-axis data
// `DATAX1` is the address of the upper byte (MSB, most significant byte) of the X-axis data
// The ADXL345 combines both registers to deliver 16-bit output for each acceleration axis.
// A similar approach is used for the Y and Z axes. This technique provides higher precision in the measurements.
func (d *Dev) readAndCombine(reg1, reg2 byte) int16 {
	low, _ := d.ReadRegister(reg1)
	high, _ := d.ReadRegister(reg2)
	return int16(uint16(high)<<8) | int16(low)
}

// ReadRegister reads a 16-bit value from the specified register address.
func (d *Dev) ReadRegister(regAddress byte) (int16, error) {
	// Send a two-byte sequence:
	// - The first byte contains the address with bit 7 set high to indicate read op
	// - The second byte is a "don't care" value, usually zero
	tx := []byte{regAddress | 0x80, 0x00}
	rx := make([]byte, len(tx))
	err := d.s.Tx(tx, rx)
	r := int16(binary.LittleEndian.Uint16(rx))
	return r, err
}

// RawReadRegister reads a []byte value from the specified register address.
func (d *Dev) RawReadRegister(regAddress byte) ([]byte, error) {
	// Send a two-byte sequence:
	// - The first byte contains the address with bit 7 set high to indicate read op
	// - The second byte is a "don't care" value, usually zero
	tx := []byte{regAddress | 0x80, 0x00}
	rx := make([]byte, len(tx))
	err := d.s.Tx(tx, rx)
	return rx, err
}

// WriteRegister writes a 1 byte value to the specified register address.
func (d *Dev) WriteRegister(regAddress byte, value byte) error {
	// Prepare a 2-byte buffer with the register address and the desired value.
	tx := []byte{regAddress, value}
	// Prepare a receiving buffer of the same size as the transmit buffer.
	rx := make([]byte, len(tx))
	// Perform the transfer. We expect the SPI device to write back an acknowledgement.
	err := d.s.Tx(tx, rx)
	return err
}

// Acceleration represents the acceleration on the three axes
type Acceleration struct {
	X int16
	Y int16
	Z int16
}

// String returns a string representation of the Acceleration
func (a Acceleration) String() string {
	return fmt.Sprintf("X:%d Y:%d Z:%d", a.X, a.Y, a.Z)
}
