// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This package is driver for the STMicroelectronics LPS series of pressure
// sensors. It supports the LPS22HB, LPS25HB, and LPS28DFW sensors.
//
// # Datasheets
//
// LPS22HB
// https://www.st.com/resource/en/datasheet/lps22hb.pdf
//
// LPS25HB
// https://www.st.com/resource/en/datasheet/lps25hb.pdf
//
// LPS28DFW
// https://www.st.com/resource/en/datasheet/lps28dfw.pdf
package lps2x

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
)

const (
	DefaultAddress i2c.Addr = 0x5c

	// The default measuring scale for these devices is HectoPascal, which is
	// 100 Pa.
	HectoPascal physic.Pressure = 100 * physic.Pascal

	// These devices implement an identify command that returns the model ID.
	LPS22HB  byte = 0xb1
	LPS25HB  byte = 0xbd
	LPS28DFW byte = 0xb4
)

type SampleRate byte
type AverageRate byte

const (
	lps22hb  = "LPS22HB"
	lps25hb  = "LPS25HB"
	lps28dfw = "LPS28DFW"

	cmdWhoAmI                 = 0x0f
	cmdStatus                 = 0x27
	cmdSampleRate             = 0x10
	cmdResConfLPS25HB         = 0x10
	cmdSampleRateLPS25HB      = 0x20
	dataReady            byte = 0x03
	minTemperature            = physic.ZeroCelsius - 40*physic.Kelvin
	maxTemperature            = physic.ZeroCelsius + 85*physic.Kelvin

	minPressure = 260 * HectoPascal

	minSampleDuration = time.Microsecond
)
const (
	SampleRateOneShot SampleRate = iota
	SampleRateHertz
	SampleRate4Hertz
	SampleRate10Hertz
	SampleRate25Hertz
	SampleRate50Hertz
	SampleRate75Hertz
	SampleRate100Hertz
	SampleRate200Hertz
)

const (
	SampleRateLPS25HBHertz = iota
	SampleRateLPS25HB7Hertz
	SampleRateLPS25HB12_5Hertz
	SampleRateLPS25HB25Hertz
)

const (
	AverageNone AverageRate = iota
	AverageReadings4
	AverageReadings8
	AverageReadings16
	AverageReadings32
	AverageReadings64
	AverageReadings128
	AverageReadings512
)

var (
	sampleRateTimes = []time.Duration{
		0,
		time.Second,
		time.Second / 4,
		time.Second / 10,
		time.Second / 25,
		time.Second / 50,
		time.Second / 75,
		time.Second / 100,
		time.Second / 200,
	}
	averageMultiple = []int{
		1,
		4,
		8,
		16,
		32,
		64,
		128,
		512,
	}
)

type Dev struct {
	conn            conn.Conn
	mu              sync.Mutex
	shutdown        chan struct{}
	deviceID        byte
	fsMode          byte
	sampleRate      SampleRate
	averageReadings AverageRate
}

// New creates a new LPS2x device on the specified I²C bus.
// addr is the I²C address (typically DefaultAddress or AlternateAddress).
// sampleRate controls measurement frequency, averageReadings controls internal averaging.
func New(bus i2c.Bus, address i2c.Addr, sampleRate SampleRate, averageRate AverageRate) (*Dev, error) {
	dev := &Dev{conn: &i2c.Dev{Bus: bus, Addr: uint16(address)}, sampleRate: sampleRate, averageReadings: averageRate}

	return dev, dev.start()
}

// start does an i2c transaction to read the device id and returns the error
// if any.
func (dev *Dev) start() error {

	r := []byte{0}
	err := dev.conn.Tx([]byte{cmdWhoAmI}, r)
	if err != nil {
		return err
	}

	dev.deviceID = r[0]
	if err == nil {
		if dev.deviceID == LPS25HB {
			// There are some key differences for this model. In this case, the Average Rate
			// is in the 0x10 register, and the sample rate is in the 0x20 register.
			// Also, the lps25hb supports different sample rates than other members of the
			// family.
			if dev.sampleRate > SampleRate25Hertz {
				return fmt.Errorf("lps2x: invalid sample rate %d, max: %d", dev.sampleRate, SampleRate25Hertz)
			}
			var tAvg, pAvg byte
			switch dev.averageReadings {
			case AverageReadings4:
			case AverageReadings8:
				// the default 0 value is correct.
			case AverageReadings16:
				tAvg = 1
				pAvg = 1
			case AverageReadings32:
				tAvg = 1
				pAvg = 2
			case AverageReadings64:
				tAvg = 1
				pAvg = 3
			case AverageReadings128:
				tAvg = 2
				pAvg = 3
			case AverageReadings512:
				tAvg = 3
				pAvg = 3
			}

			err = dev.conn.Tx([]byte{cmdResConfLPS25HB, tAvg<<2 | pAvg}, nil)
			if err != nil {
				err = fmt.Errorf("lps2x: error setting average rates %w", err)
			} else {
				odr := byte(0x80 | (dev.sampleRate << 4))
				err = dev.conn.Tx([]byte{cmdSampleRateLPS25HB, odr}, nil)
			}

		} else {
			err = dev.conn.Tx([]byte{cmdSampleRate, byte(dev.sampleRate<<3) | byte(dev.averageReadings)}, nil)
		}
	}
	return err
}

func (dev *Dev) Halt() error {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	if dev.shutdown != nil {
		close(dev.shutdown)
	}
	return nil
}

func (dev *Dev) Precision(env *physic.Env) {
	env.Humidity = 0
	env.Temperature = physic.Kelvin / 100
	env.Pressure = HectoPascal
}

func (dev *Dev) Sense(env *physic.Env) error {
	env.Humidity = 0

	// We're reading the status byte, and the following 5 bytes: 3 bytes of
	// pressure data, and 2 temperature bytes.
	w := []byte{cmdStatus}
	r := make([]byte, 6)

	err := dev.conn.Tx(w, r)
	if err != nil {
		env.Temperature = minTemperature
		env.Pressure = minPressure
		return fmt.Errorf("lps2x: error reading device %w", err)
	}
	if r[0]&dataReady != dataReady {
		env.Temperature = minTemperature
		env.Pressure = minPressure
		return errors.New("lps2x: data not ready, was sampling started?")
	}

	env.Temperature = dev.countToTemp(int16(r[5])<<8 | int16(r[4]))
	env.Pressure = dev.countToPressure(convert24BitTo64Bit(r[1:4]))
	return nil
}

func (dev *Dev) SenseContinuous(interval time.Duration) (<-chan physic.Env, error) {
	d := sampleRateTimes[dev.sampleRate]
	d *= time.Duration(averageMultiple[dev.averageReadings])
	if interval < d {
		return nil, fmt.Errorf("invalid duration, minimum duration: %v", d)
	}
	dev.mu.Lock()
	if dev.shutdown != nil {
		dev.mu.Unlock()
		return nil, errors.New("lps2x: SenseContinuous already running")
	}
	dev.mu.Unlock()

	if interval < minSampleDuration {
		// TODO: Verify
		return nil, errors.New("lps2x: sample interval is < device sample rate")
	}
	dev.shutdown = make(chan struct{})
	ch := make(chan physic.Env, 16)
	go func(ch chan<- physic.Env) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(ch)
		for {
			select {
			case <-dev.shutdown:
				dev.mu.Lock()
				defer dev.mu.Unlock()
				dev.shutdown = nil
				return
			case <-ticker.C:
				env := physic.Env{}
				if err := dev.Sense(&env); err == nil {
					ch <- env
				}
			}
		}
	}(ch)
	return ch, nil
}

// String returns the device model name.
func (dev *Dev) String() string {
	switch dev.deviceID {
	case LPS22HB:
		return lps22hb
	case LPS25HB:
		return lps25hb
	case LPS28DFW:
		return lps28dfw
	default:
		return "unknown"
	}
}

func convert24BitTo64Bit(bytes []byte) int64 {
	// Mask to isolate the lower 24 bits (0x00FFFFFF)
	// This ensures we only consider the 24-bit value if it was derived from a larger type
	val := uint32(bytes[0]) | uint32(bytes[1])<<8 | uint32(bytes[2])<<16

	// Check if the 24th bit (the sign bit) is set (0x00800000)
	if (val & 0x00800000) != 0 {
		// If the sign bit is set, it's a negative number.
		// Sign-extend by filling the upper 8 bits with ones (0xFF000000).
		val |= 0xFF000000
	}

	return int64(val)
}

func (dev *Dev) countToTemp(count int16) physic.Temperature {
	temp := physic.Temperature(count)*10*physic.MilliKelvin + physic.ZeroCelsius
	if temp < minTemperature {
		temp = minTemperature
	} else if temp > maxTemperature {
		temp = maxTemperature
	}
	return temp
}

func (dev *Dev) countToPressure(count int64) physic.Pressure {
	if dev.fsMode == 0 {
		return (physic.Pressure(count) * HectoPascal) / 4096
	}
	return (physic.Pressure(count) * HectoPascal) / 2048
}

var _ conn.Resource = &Dev{}
var _ physic.SenseEnv = &Dev{}
