// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// sht4x is a package for interfacing with the Sensirion SHT-40, SHT-41, and
// SHT-45 sensors.
//
// # Datasheet
//
// https://sensirion.com/media/documents/33FD6951/67EB9032/HT_DS_Datasheet_SHT4x_5.pdf
//
// # Temperature Accuracy
//
// SHT-40 & SHT-41
//
//	Typical accuracy: ±0.2 °C
//
//	Response time τ₆₃% ≈ 2 s
//
// SHT-45
//
//	Typical accuracy: ±0.1 °C
//
//	Response time τ₆₃% ≈ 2 s
//
// # Humidity Accuracy
//
// SHT-40 (Base‑class)
//
//	Typical accuracy at 25 °C: ±1.8 % RH
//
//	Maximum accuracy (at 25 °C): up to ±3.5 % RH
//
// SHT-41 (Intermediate‑class)
//
//	Typical accuracy at 25 °C: ±1.8 % RH
//
//	Maximum accuracy (at 25 °C): up to ±2.5 % RH
//
// SHT-45 (High‑accuracy‑class)
//
//	Typical accuracy at 25 °C: ±1.0 % RH
//
//	Maximum accuracy (at 25 °C): up to ≈±1.75 % RH
//
// All three share a resolution of 0.01 % RH, a response time τ₆₃% ≈ 4 s, and long‑term drift < 0.2 % RH/year .
//
// All devices have a resolution of 0.01 °C and specified range –40…+125 °C .
package sht4x

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/common"
)

// HeaterPower represents a type for the heater power setting.
type HeaterPower int

// HeaterDuration represents a duration for turning the heater on.
type HeaterDuration time.Duration

const (
	// Power settings for the heater element.
	Power20mW HeaterPower = iota
	Power110mW
	Power200mW

	// Durations that you can turn the heater on for.
	Duration100ms HeaterDuration = HeaterDuration(time.Duration(100 * time.Millisecond))
	Duration1s    HeaterDuration = HeaterDuration(time.Second)

	// Default I2C Address
	DefaultAddress i2c.Addr = 0x44
)

const (
	// byte commands for device.
	cmdHeater200mW1s    byte = 0x39
	cmdHeater200mW100ms byte = 0x32
	cmdHeater110mW1s    byte = 0x2f
	cmdHeater110mW100ms byte = 0x24
	cmdHeater20mW1s     byte = 0x1e
	cmdHeater20mW100ms  byte = 0x15

	cmdSoftReset byte = 0x94
	// Read at highest precision and repeatability
	cmdMeasure          byte = 0xfd
	cmdReadSerialNumber byte = 0x89

	countDivisor = float64(65535)

	minTemperature = -40*physic.Kelvin + physic.ZeroCelsius
	maxTemperature = 125*physic.Kelvin + physic.ZeroCelsius

	minRH = 0 * physic.PercentRH
	maxRH = 100 * physic.PercentRH

	minSampleDuration = 10 * time.Millisecond
)

// Dev represents a SHT-4X series temperature/humidity sensor
type Dev struct {
	d        *i2c.Dev
	shutdown chan struct{}
	mu       sync.Mutex
}

func New(bus i2c.Bus, addr i2c.Addr) (*Dev, error) {
	dev := &Dev{d: &i2c.Dev{Bus: bus, Addr: uint16(addr)}}
	return dev, nil
}

// If you try to read immediately after a write with this device, you'll get an
// io error. This just wraps the write and adds a delay before attempting the
// read.
func (dev *Dev) txWithDelay(w, r *[]byte, delay time.Duration) (err error) {
	if w != nil {
		err = dev.d.Tx(*w, nil)
		if err != nil {
			err = fmt.Errorf("sht4x: error transmitting %w", err)
			return
		}
	}
	time.Sleep(delay)
	if r != nil {
		err = dev.d.Tx(nil, *r)
		if err != nil {
			err = fmt.Errorf("sht4x: error reading %w", err)
		}
		// All calls that return bytes return the same format. 2 bytes
		// of data, a CRC, 2 bytes of data, and
		// a CRC. Verify them
		if common.CRC8((*r)[:2]) != (*r)[2] {
			err = errors.New("sht4x: bytes[:2] read crc error")
		}
		if err == nil && common.CRC8((*r)[3:5]) != (*r)[5] {
			err = errors.New("sht4x: bytes[3:5] read crc error")
		}
	}
	return
}

// convert the count to a temperature value.
func countToTemp(count uint16) physic.Temperature {
	// T=-45+175*(count/countDivisor)
	val := physic.Temperature(float64(physic.Kelvin)*(-45.0+175.0*(float64(count)/countDivisor))) + physic.ZeroCelsius
	if val < minTemperature {
		val = minTemperature
	} else if val > maxTemperature {
		val = maxTemperature
	}
	return val
}

func countToHumidity(count uint16) physic.RelativeHumidity {
	// RH=-6 + 125*(count/countDivisor)
	val := physic.RelativeHumidity((-6.0 + 125.0*(float64(count)/countDivisor)) * float64(physic.PercentRH))
	if val < minRH {
		val = minRH
	} else if val > maxRH {
		val = maxRH
	}
	return val
}

// Precision returns the smallest change in readings the device can produce.
// Implements physic.SenseEnv.
func (dev *Dev) Precision(e *physic.Env) {
	e.Temperature = physic.Kelvin / 100
	e.Humidity = physic.PercentRH / 100
	e.Pressure = 0
}

// Halt shuts down the device and terminates a SenseContinuous
// command if running. Implements conn.Resource
func (dev *Dev) Halt() error {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	if dev.shutdown != nil {
		close(dev.shutdown)
	}
	return nil
}

// Reset issues a soft-reset to the device
func (dev *Dev) Reset() error {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	err := dev.d.Tx([]byte{cmdSoftReset}, nil)
	if err != nil {
		err = fmt.Errorf("sht4x: error resetting %w", err)
	}
	time.Sleep(2 * time.Millisecond)
	return err
}

// Sense reads temperature and humidity from the device.
func (dev *Dev) Sense(e *physic.Env) error {
	e.Pressure = 0
	r := make([]byte, 6)
	w := []byte{cmdMeasure}
	err := dev.txWithDelay(&w, &r, 10*time.Millisecond)
	if err != nil {
		e.Temperature = minTemperature
		e.Humidity = minRH
		return fmt.Errorf("sht4x: error reading device %w", err)
	}
	e.Temperature = countToTemp(uint16(r[0])<<8 | uint16(r[1]))
	e.Humidity = countToHumidity(uint16(r[3])<<8 | uint16(r[4]))
	return nil
}

// SenseContinuous continuously reads from the device and sends the output
// to the returned channel. To terminate the read, call Dev.Halt()
func (dev *Dev) SenseContinuous(duration time.Duration) (<-chan physic.Env, error) {

	if dev.shutdown != nil {
		return nil, errors.New("sht4x: SenseContinuous already running")
	}

	if duration < minSampleDuration {
		return nil, errors.New("sht4x: sample interval is < device sample rate")
	}
	dev.shutdown = make(chan struct{})
	ch := make(chan (physic.Env), 16)
	go func(ch chan<- physic.Env) {
		ticker := time.NewTicker(duration)
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

// SerialNumber returns the device serial number set at the factory.
func (dev *Dev) SerialNumber() (uint32, error) {
	r := make([]byte, 6)
	w := []byte{cmdReadSerialNumber}
	dev.mu.Lock()
	defer dev.mu.Unlock()
	err := dev.txWithDelay(&w, &r, 10*time.Millisecond)
	if err != nil {
		return 0, err
	}
	result := uint32(r[0])<<24 | uint32(r[1])<<16 | uint32(r[3])<<8 | uint32(r[4])
	return result, nil
}

// SetHeater enables the sensor's heater. You can specify the power level, and
// the duration. After duration has passed, the heater will be turned off
// automatically. Enabling the heater can allow operation in condensing
// environments.
//
// powerLevel is one of the HeaterPower constants, and duration is one of the
// heaterDuration constants, either 100ms, or 1000ms.
//
// Returns the temperature and humidity after the period has completed. Refer to
// section 4.9 of the datasheet.
func (dev *Dev) SetHeater(powerLevel HeaterPower, duration HeaterDuration) (physic.Env, error) {
	env := physic.Env{Temperature: minTemperature, Humidity: minRH}
	var cmd byte
	switch duration {
	case Duration100ms:
		switch powerLevel {
		case Power20mW:
			cmd = cmdHeater20mW100ms
		case Power110mW:
			cmd = cmdHeater110mW100ms
		case Power200mW:
			cmd = cmdHeater200mW100ms
		default:
			return env, errors.New("sht4x: invalid heater power")
		}
	case Duration1s:
		switch powerLevel {
		case Power20mW:
			cmd = cmdHeater20mW1s
		case Power110mW:
			cmd = cmdHeater110mW1s
		case Power200mW:
			cmd = cmdHeater200mW1s
		default:
			return env, errors.New("sht4x: invalid heater power")
		}
	default:
		return env, errors.New("sht4x: invalid heater duration")
	}
	r := make([]byte, 6)
	w := []byte{cmd}
	dev.mu.Lock()
	defer dev.mu.Unlock()
	err := dev.txWithDelay(&w, &r, time.Duration(duration)+10*time.Millisecond)
	if err != nil {
		return env, fmt.Errorf("sht4x: error setting heater %w", err)
	}
	env.Temperature = countToTemp(uint16(r[0])<<8 | uint16(r[1]))
	env.Humidity = countToHumidity(uint16(r[3])<<8 | uint16(r[4]))

	return env, nil
}

// String returns a string representation of the device.
func (dev *Dev) String() string {
	return "sht4x"
}

var _ conn.Resource = &Dev{}
var _ physic.SenseEnv = &Dev{}
