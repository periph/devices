// Copyright 2023 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package adxl345

import (
	"encoding/binary"
	"fmt"
	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
)

// Sensitivity represents the sensitivity of the Accelerometer.
type Sensitivity byte

// The following constants are register used by the ADXL345
// Check the table 19 of the datasheet for more details.
// https://www.analog.com/media/en/technical-documentation/data-sheets/ADXL345.pdf
const (
	DeviceID = 0x00 // Device ID, expected to be 0xE5 when using ADXL345

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

// Sensitivity constants represents the sensitivity of the ADXL345.
// The ADXL345 supports 4 sensitivity settings, ±2g, ±4g, ±8g, and ±16g, with the default being ±2g.
// Sensitivity is an option that can be set at initialization in Opts.
// You can set the sensitivity of the ADXL345 with the DataFormat register.
const (
	S2G  Sensitivity = 0x00 // Sensitivity at 2g
	S4G  Sensitivity = 0x01 // Sensitivity at 4g
	S8G  Sensitivity = 0x02 // Sensitivity at 8g
	S16G Sensitivity = 0x03 // Sensitivity at 16g
)

// Currently tested  devices

const (
	AdxlXXX = 0x01 // No specific expectation. For non-detected devices the response is 0x00
	Adxl345 = 0xE5 // Expecting an Adxl345
)

var (
	SpiFrequency = physic.MegaHertz * 2
	SpiMode      = spi.Mode3 // Defines the base clock signal, along with the polarity and phase of the data signal.
	SpiBits      = 8
)

var DefaultOpts = Opts{
	ExpectedDeviceID: AdxlXXX, // No specific expectation by default
	Sensitivity:      S2G,
}

type Opts struct {
	ExpectedDeviceID byte        // Expected device ID used to verify that the device is an ADXL345.
	Sensitivity      Sensitivity // Sensitivity of the device (2G, 4G, 8G, 16G)
}

// Dev is a driver for the ADXL345 accelerometer
// It uses the SPI interface to communicate with the device.
type Dev struct {
	c     conn.Conn
	name  string
	isSPI bool
	// The sensitivity of the device (2G, 4G, 8G, 16G)
	// Set to 2G by default, can be changed in the Opts at initialization.
	sensitivity Sensitivity
}

func (d *Dev) Mode() string {
	if d.isSPI {
		return "SPI"
	} else {
		return "I²C"
	}
}

func (d *Dev) String() string {
	return fmt.Sprintf("%s{Sensitivity:%s, Mode:%s}", d.name, d.sensitivity, d.Mode())
}

// NewI2C returns an object that communicates over I²C to ADXL345
// accelerometer.
//
// The device is automatically turned on and the sensitivity is set to the Opts.Sensitivity.
func NewI2C(b i2c.Bus, addr uint16, opts *Opts) (*Dev, error) {
	d := &Dev{
		c:     &i2c.Dev{Bus: b, Addr: addr},
		isSPI: false}
	if err := d.makeDev(opts); err != nil {
		return nil, err
	}
	return d, nil
}

// NewSpi returns an object that communicates over spi to ADXL345
// accelerometer.
//
// The device is automatically turned on and the sensitivity is set to the Opts.Sensitivity.
func NewSpi(p spi.Port, o *Opts) (*Dev, error) {
	// Convert the spi.Port into a spi.Conn so it can be used for communication.
	c, err := p.Connect(SpiFrequency, SpiMode, SpiBits)
	if err != nil {
		return nil, err
	}
	d := &Dev{
		c:     c,
		isSPI: true,
	}
	err = d.makeDev(o)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// makeDev turns on with the expected sensitivity and verifies if it is a supported device.
func (d *Dev) makeDev(o *Opts) error {
	err := d.TurnOn()
	if err != nil {
		return err
	}
	if o.Sensitivity != S2G { // default
		err = d.setSensitivity(o.Sensitivity)
		if err != nil {
			return err
		}
	}
	// Verify that the device Id
	rx, err := d.Read(DeviceID)
	if err != nil {
		return fmt.Errorf("unable to read the deviceID \"%s\"", err.Error())
	}
	switch byte(rx & 0xff) {
	case Adxl345:
		d.name = "adxl345"
		return nil
	case o.ExpectedDeviceID:
		d.name = fmt.Sprintf("expected%#x", o.ExpectedDeviceID)
		return nil
	default:
		return fmt.Errorf("unrecognized device expected=\"%#02x\" or \"%#02x\"found=\"%#02x\" ", o.ExpectedDeviceID, Adxl345, rx)
	}
}

// SetSensitivity sets the sensitivity of the ADXL345.
// The sensitivity parameter should be one of 2, 4, 8, or 16, representing ±2g, ±4g, ±8g, or ±16g respectively.
func (d *Dev) setSensitivity(sensitivity Sensitivity) error {
	switch sensitivity {
	case S2G, S4G, S8G, S16G:
		// Write to the DataFormat register
		d.sensitivity = sensitivity
		return d.Write(DataFormat, byte(sensitivity))
	default:
		return fmt.Errorf("invalid sensitivity: %d. Valid values are 2, 4, 8, 16", sensitivity)
	}
}

// TurnOn turns on the measurement mode of the ADXL345.
// This is required before reading data from the device.
func (d *Dev) TurnOn() error {
	return d.Write(PowerCtl, 0x08)
}

// TurnOff turns off the measurement mode of the ADXL345.
func (d *Dev) TurnOff() error {
	return d.Write(PowerCtl, 0x00)
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
// The ADXL345 uses two 8-bit registers to store the output data for each axis.
// X := d.readAndCombine(DataX0, DataX1) where:
// `DataX0` is the address of the lower byte (LSB, least significant byte)
// `DataX1` is the address of the upper byte (MSB, most significant byte)
// The ADXL345 combines both registers to deliver 16-bit output for each acceleration axis.
// A similar approach is used for the Y and Z axes. This technique provides higher precision in the measurements.
func (d *Dev) readAndCombine(reg1, reg2 byte) int16 {
	low, _ := d.Read(reg1)
	high, _ := d.Read(reg2)
	return int16(uint16(high)<<8) | int16(low)
}

// Read reads a 16-bit value from the specified register address.
func (d *Dev) Read(regAddress byte) (int16, error) {
	// Send a two-byte sequence:
	// - The first byte contains the address with bit 7 set high to indicate read op
	// - The second byte is a "don't care" value, usually zero
	tx := []byte{regAddress | 0x80, 0x00}
	rx := make([]byte, len(tx))
	err := d.c.Tx(tx, rx)
	if err != nil {
		return 0, err
	}
	return int16(binary.LittleEndian.Uint16(rx)), nil
}

// Write writes a 1 byte value to the specified register address.
func (d *Dev) Write(regAddress byte, value byte) error {
	return d.c.Tx([]byte{regAddress, value}, nil)
}

// Acceleration represents the acceleration on the three axes X,Y,Z.
// The sensitivity can be set to different levels: ±2g, ±4g, ±8g, or ±16g. (S2G, S4G, S8G, S16G)
// The output are 16-bit integers, so the device measures between -32768 and +32767 for each axis.
// For example, if the sensitivity is set to ±2g and you're getting a reading of 16384 on the X axis, that would correspond to 1g of acceleration along the X axis.
// To convert the raw values to a physical unit (like g or m/s²), you would need to know the sensitivity setting of your device.
// For instance, if your sensitivity is set to ±2g, the conversion factor would be 2 / 32768 = 0.000061g per count.
// So, you would multiply the raw acceleration values by this factor to get the acceleration in `g`.
type Acceleration struct {
	X int16
	Y int16
	Z int16
}

// String returns a string representation of the Acceleration
func (a Acceleration) String() string {
	return fmt.Sprintf("X:%d Y:%d Z:%d", a.X, a.Y, a.Z)
}

// Sensitivity returns the sensitivity of the device as a human-readable string.
func (s Sensitivity) String() string {
	switch s {
	case S2G:
		return "+/-2g"
	case S4G:
		return "+/-4g"
	case S8G:
		return "+/-8g"
	case S16G:
		return "+/-16g"
	default:
		return fmt.Sprintf("unknown sensitivity: %#x", s)
	}
}
