// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This package provides a driver for the AOSONG AM2320 Temperature/Humidity
// Sensor. This sensor is a basic, inexpensive i2c sensor with reasonably good
// accuracy for both temperature and humidity.
//
// # Datasheet
//
// https://cdn-shop.adafruit.com/product-files/3721/AM2320.pdf
package am2320

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
)

// Dev represents an am2320 temperature/humidity sensor.
type Dev struct {
	d        *i2c.Dev
	mu       sync.Mutex
	shutdown chan struct{}
}

const (
	// The address of this device is fixed. Note that the datasheet states
	// the value is 0xb8, which is incorrect.
	SensorAddress uint16 = 0x5c

	humidityRegisters byte = 0x00
)

// Create a new am2320 device and return it.
func NewI2C(b i2c.Bus, addr uint16) (*Dev, error) {
	d := &Dev{d: &i2c.Dev{Bus: b, Addr: addr}}
	return d, nil
}

// Halt interrupts a running SenseContinuous() operation.
func (dev *Dev) Halt() error {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	if dev.shutdown != nil {
		close(dev.shutdown)
	}
	return nil
}

// Algorithm from the datasheet. Returns true if CRC matches check value.
func checkCRC(bytes []byte) bool {
	crc := uint16(0xffff)
	for ix := range len(bytes) - 2 {
		b := uint16(bytes[ix])
		crc ^= b
		for range 8 {
			if (crc & 0x01) == 0x01 {
				crc = crc >> 1
				crc ^= 0xa001
			} else {
				crc = crc >> 1
			}
		}
	}
	chk := uint16(bytes[len(bytes)-2]) | uint16(bytes[len(bytes)-1])<<8
	return chk == crc
}

// readCommand provides the logic of communicating with the sensor. According
// to the datasheet, it tries to stay in low-power as much as possible to
// avoid self-heating the sensors. This makes it finicky to talk to. On success,
// returns a slice of registerCount bytes starting from registerAddress.
func (dev *Dev) readCommand(registerAddress, registerCount byte) ([]byte, error) {
	// Send a wake-up call to the device.
	var err error
	for range 5 {
		err = dev.d.Tx([]byte{0}, nil)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	w := []byte{0x3, registerAddress, registerCount}
	// The read return format is:
	//
	// {operation,registerCount,requested registers...,crc low, crc high}
	r := make([]byte, registerCount+4)

	for range 10 {
		err = dev.d.Tx(w, r)
		if err == nil &&
			w[0] == r[0] && w[2] == r[1] &&
			checkCRC(r) {

			return r[2 : 2+registerCount], nil
		}
		time.Sleep(2 * time.Second)
	}
	if err == nil {
		err = errors.New("invalid return values or crc from sensor")
	}
	return nil, fmt.Errorf("am2320 error sending read command: %w", err)
}

// Sense queries the sensor for the current temperature and humidity. Note that
// the sensor reports a sample rate of 1/2 hz. It's recommended to not poll
// the sensor more frequently than once every 3 seconds.
func (dev *Dev) Sense(env *physic.Env) error {
	env.Temperature = 0
	env.Pressure = 0
	env.Humidity = 0

	dev.mu.Lock()
	defer dev.mu.Unlock()

	r, err := dev.readCommand(humidityRegisters, 4)
	if err != nil {
		return err
	}

	h := int16(r[0])<<8 | int16(r[1])
	env.Humidity = physic.RelativeHumidity(h) * physic.MilliRH
	t := int16(r[2])<<8 | int16(r[3])
	env.Temperature = physic.ZeroCelsius + (physic.Celsius/10)*physic.Temperature(t)

	return nil
}

// SenseContinuous returns a channel that can be read to return values from
// the sensor. The minimum value for interval is 3 seconds. To end the read,
// call Halt()
func (dev *Dev) SenseContinuous(interval time.Duration) (<-chan physic.Env, error) {
	if interval < (3 * time.Second) {
		return nil, errors.New("am2320: invalid duration. minimum 3 seconds")
	}
	if dev.shutdown != nil {
		return nil, errors.New("am2320: sense continuous already running")
	}

	dev.shutdown = make(chan struct{})
	ch := make(chan physic.Env, 16)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-dev.shutdown:
				close(ch)
				dev.shutdown = nil
				return
			case <-ticker.C:
				e := physic.Env{}
				err := dev.Sense(&e)
				if err == nil {
					ch <- e
				}
			}
		}
	}()
	return ch, nil
}

func (dev *Dev) String() string {
	return fmt.Sprintf("am2320: %s", dev.d)
}

// Precision returns the resolution of the device for it's measured parameters.
func (dev *Dev) Precision(env *physic.Env) {
	env.Temperature = physic.Celsius / 10
	env.Pressure = 0
	env.Humidity = physic.MilliRH
}

var _ conn.Resource = &Dev{}
var _ physic.SenseEnv = &Dev{}
