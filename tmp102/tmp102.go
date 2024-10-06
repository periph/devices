// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package tmp102

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
)

type ConversionRate byte

type AlertMode byte

// Dev represents a TMP102 sensor.
type Dev struct {
	d        *i2c.Dev
	shutdown chan bool
	mu       sync.Mutex
	opts     *Opts
}

const (
	// Conversion (sample) Rates. The device default is 4 readings/second.
	RateQuarterHertz ConversionRate = iota
	RateOneHertz
	RateFourHertz
	RateEightHertz

	// ModeComparator sets the device to operate in Comparator mode.
	// Refer to section 6.4.5.1 of the TMP102 datasheet. When used, the
	// ALERT pin of the TMP102 will trigger.
	ModeComparator AlertMode = 0
	// ModeInterrupt sets the device to operate in Interrupt mode.
	// Note that reading the temperature will clear the alert, so be aware
	// if you're using SenseContinuous.
	ModeInterrupt AlertMode = 1

	// Addresses of registers to read/write.
	_REGISTER_TEMPERATURE   byte = 0
	_REGISTER_CONFIGURATION byte = 1
	_REGISTER_RANGE_LOW     byte = 2
	_REGISTER_RANGE_HIGH    byte = 3

	// Bit numbers for various configuration operations.
	_SHUTDOWN_BIT        int = 8
	_THERMOSTAT_MODE     int = 9
	_CONVERSION_RATE_POS int = 6

	_DEGREES_RESOLUTION physic.Temperature = 62_500 * physic.MicroKelvin

	// The minimum temperature in StandardMode the device can read.
	MinimumTemperature physic.Temperature = physic.ZeroCelsius - 40*physic.Kelvin
	// The maximum temperature in StandardMode the device can read.
	MaximumTemperature physic.Temperature = physic.ZeroCelsius + 125*physic.Kelvin
)

// Opts represents configurable options for the TMP102.
type Opts struct {
	SampleRate   ConversionRate
	AlertSetting AlertMode
	AlertLow     physic.Temperature
	AlertHigh    physic.Temperature
}

func (dev *Dev) isShutdown() bool {
	return dev.shutdown == nil
}

// start initializes the device to a known state and ensures its
// not in shutdown mode.
func (dev *Dev) start() error {
	config := dev.ReadConfiguration()
	mask := uint16(0xffff) ^ (uint16(1<<_SHUTDOWN_BIT) | uint16(1<<_THERMOSTAT_MODE))
	config &= mask

	config |= uint16(dev.opts.AlertSetting) << _THERMOSTAT_MODE

	cr := ConversionRate((config >> _CONVERSION_RATE_POS) & 0x03)
	if cr != dev.opts.SampleRate {
		// Turn off the sample rate bits.
		config &= 0xffff ^ uint16(0x03<<_CONVERSION_RATE_POS)
		// Now set the new value.
		config |= uint16(dev.opts.SampleRate) << _CONVERSION_RATE_POS
	}

	var bits []byte
	w := make([]byte, 3)
	w[0] = _REGISTER_CONFIGURATION
	w[1] = byte(config>>8) & 0xff
	w[2] = byte(config & 0xff)

	err := dev.d.Tx(w, nil)
	if err != nil {
		return err
	}
	dev.shutdown = make(chan bool)
	if dev.opts.AlertLow != 0 {
		bits, err = temperatureToCount(dev.opts.AlertLow)
		if err != nil {
			return err
		}
		w[0] = _REGISTER_RANGE_LOW
		w[1] = bits[0]
		w[2] = bits[1]
		err = dev.d.Tx(w, nil)
		if err != nil {
			return err
		}
	}

	if dev.opts.AlertHigh != 0 {
		bits, err = temperatureToCount(dev.opts.AlertHigh)
		if err != nil {
			return err
		}
		w[0] = _REGISTER_RANGE_HIGH
		w[1] = bits[0]
		w[2] = bits[1]
		err = dev.d.Tx(w, nil)
	}
	return err
}

// temperatureToCount converts a temperature into the count that the device
// uses. Required to set the Low/High Range registers for alerts.
func temperatureToCount(temp physic.Temperature) ([]byte, error) {
	result := make([]byte, 2)
	if temp == physic.ZeroCelsius {
		return result, nil
	}

	negative := temp < physic.ZeroCelsius
	var count uint16
	if negative {
		temp = physic.ZeroCelsius + physic.Temperature(-1*temp.Celsius())*physic.Kelvin
		count = uint16((temp - physic.ZeroCelsius) / _DEGREES_RESOLUTION)
		count = ((twosComplement(count) | (1 << 11)) + 1)

	} else {
		count = uint16((temp - physic.ZeroCelsius) / _DEGREES_RESOLUTION)

	}
	count = count << 4
	result[0] = byte(count >> 8 & 0xff)
	result[1] = byte(count & 0xf0)
	return result, nil
}

func twosComplement(value uint16) uint16 {
	var result uint16
	for iter := 0; iter < 11; iter++ {
		bitVal := uint16(1 << iter)
		if (value & bitVal) == 0 {
			result |= bitVal
		}
	}
	return result
}

// countToTemperature returns the temperature from the raw device count.
func countToTemperature(bytes []byte) physic.Temperature {
	count := (uint16(bytes[0]) << 4) | (uint16(bytes[1]) >> 4)
	negative := (count & (1 << 11)) > 0
	if negative {
		count = twosComplement(count) + 1
	}
	var t physic.Temperature
	if negative {
		t = physic.ZeroCelsius - (physic.Temperature(count) * _DEGREES_RESOLUTION)
	} else {
		t = physic.ZeroCelsius + (physic.Temperature(count) * _DEGREES_RESOLUTION)
	}
	return t
}

// readConfiguration returns the device's configuration registers as a 16 bit
// unsigned integer. Refer to the datasheet for interpretation.
func (dev *Dev) ReadConfiguration() uint16 {
	w := make([]byte, 1)
	w[0] = _REGISTER_CONFIGURATION
	r := make([]byte, 2)
	_ = dev.d.Tx(w, r)
	result := uint16(r[0])<<8 | uint16(r[1])

	return result
}

// readTemperature returns the raw counts from the device temperature registers.
func (dev *Dev) readTemperature() (physic.Temperature, error) {
	var err error
	if dev.isShutdown() {
		err = dev.start()
		if err != nil {
			return MinimumTemperature, err
		}
	}
	r := make([]byte, 2)
	err = dev.d.Tx([]byte{_REGISTER_TEMPERATURE}, r)
	if err != nil {
		return MinimumTemperature, err
	}
	return countToTemperature(r), nil
}

// NewI2C returns a new TMP102 sensor using the specified bus and address.
// If opts is not supplied, the configuration of the sensor is set to the
// default on startup.
func NewI2C(b i2c.Bus, addr uint16, opts *Opts) (*Dev, error) {
	if opts == nil {
		opts = &Opts{SampleRate: RateFourHertz, AlertSetting: ModeComparator}
	}
	d := &Dev{d: &i2c.Dev{Bus: b, Addr: addr}, opts: opts, shutdown: nil}
	return d, d.start()
}

// GetAlertMode returns the current alert settings for the device.
func (dev *Dev) GetAlertMode() (mode AlertMode, rangeLow, rangeHigh physic.Temperature, err error) {
	dev.mu.Lock()
	defer dev.mu.Unlock()

	mode = AlertMode((dev.ReadConfiguration() >> _THERMOSTAT_MODE) & 0x01)
	rangeLow = MinimumTemperature
	rangeHigh = MaximumTemperature

	w := make([]byte, 1)
	r := make([]byte, 2)

	w[0] = _REGISTER_RANGE_LOW
	err = dev.d.Tx(w, r)
	if err != nil {
		return
	}
	rangeLow = countToTemperature(r)

	w[0] = _REGISTER_RANGE_HIGH
	err = dev.d.Tx(w, r)
	if err != nil {
		return
	}
	rangeHigh = countToTemperature(r)

	return
}

// Halt shuts down the device. If a SenseContinuous operation is in progress,
// its aborted. Implements conn.Resource
func (dev *Dev) Halt() error {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	var err error

	if dev.shutdown != nil {
		close(dev.shutdown)
		dev.shutdown = nil
	}
	current := dev.ReadConfiguration()
	mask := uint16(0xffff ^ (1 << _SHUTDOWN_BIT))
	new := current & mask
	if current != new {
		w := make([]byte, 3)
		w[0] = _REGISTER_CONFIGURATION
		w[1] = byte(new >> 8)
		w[2] = byte(new & 0xff)
		err = dev.d.Tx(w, nil)
	}

	return err
}

// Sense reads temperature from the device and writes the value to the specified
// env variable. Implements physic.SenseEnv.
func (dev *Dev) Sense(env *physic.Env) error {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	t, err := dev.readTemperature()
	if err == nil {
		env.Temperature = t
	}
	return err
}

// SenseContinuous continuously reads from the device and writes the value to
// the returned channel. Implements physic.SenseEnv. To terminate the
// continuous read, call Halt().
func (dev *Dev) SenseContinuous(interval time.Duration) (<-chan physic.Env, error) {
	channelSize := 16
	if interval < (125 * time.Millisecond) {
		return nil, errors.New("invalid duration. minimum 125ms")
	}
	channel := make(chan physic.Env, channelSize)
	go func(channel chan physic.Env, shutdown <-chan bool) {
		ticker := time.NewTicker(interval)
		for {
			select {
			case <-shutdown:
				close(channel)
				return
			case <-ticker.C:
				// do the reading and write to the channel.
				e := physic.Env{}
				err := dev.Sense(&e)
				if err == nil && len(channel) < channelSize {
					channel <- e
				}
			}
		}
	}(channel, dev.shutdown)

	return channel, nil
}

// SetAlertMode sets the device to operate in alert (thermostat) mode. Alert
// mode will set the Alert pin on the device to active mode when the conditions
// apply. Refer to section 6.4.5 and section 6.5.4 of the TMP102 datasheet.
//
// To detect the alert trigger, you will need to connect the device ALERT pin
// to a GPIO pin on your SBC and configure that GPIO pin with edge detection,
// or continuously poll the GPIO pin state. If you choose polling, care should
// be taken if you're also using SenseContinuous.
func (dev *Dev) SetAlertMode(mode AlertMode, rangeLow, rangeHigh physic.Temperature) error {
	if rangeLow >= rangeHigh ||
		rangeLow < MinimumTemperature ||
		rangeHigh > MaximumTemperature {
		return errors.New("invalid temperature range")
	}
	dev.opts.AlertSetting = mode
	dev.opts.AlertLow = rangeLow
	dev.opts.AlertHigh = rangeHigh
	var err error

	dev.mu.Lock()
	defer dev.mu.Unlock()

	// Write the low range temperature
	rangeBytes, _ := temperatureToCount(rangeLow)
	w := make([]byte, 3)
	w[0] = _REGISTER_RANGE_LOW
	w[1] = rangeBytes[0]
	w[2] = rangeBytes[1]
	err = dev.d.Tx(w, nil)
	if err != nil {
		return err
	}
	// Write the High Range Temperature
	rangeBytes, _ = temperatureToCount(rangeHigh)
	w[0] = _REGISTER_RANGE_HIGH
	w[1] = rangeBytes[0]
	w[2] = rangeBytes[1]
	err = dev.d.Tx(w, nil)
	if err != nil {
		return err
	}
	// Check if the device is in shutdown, or if the mode has
	// changed, and update the device running configuration
	running := dev.ReadConfiguration()
	mask := uint16(0xffff ^ ((1 << _SHUTDOWN_BIT) | (1 << _THERMOSTAT_MODE)))
	new := (running & mask) | uint16(mode)<<_THERMOSTAT_MODE
	if new != running {
		w[0] = _REGISTER_CONFIGURATION
		w[1] = byte(new >> 8)
		w[2] = byte(new & 0xff)
		err = dev.d.Tx(w, nil)
	}

	return err
}

// Precision returns the sensor's precision, or minimum value between steps the
// device can make. The specified precision is 0.0625 degrees Celsius. Note
// that the accuracy of the device is +/- 0.5 degrees Celsius.
func (dev *Dev) Precision(env *physic.Env) {
	env.Temperature = _DEGREES_RESOLUTION
	env.Pressure = 0
	env.Humidity = 0
}

func (dev *Dev) String() string {
	return fmt.Sprintf("tmp102: %s", dev.d.String())
}

var _ conn.Resource = &Dev{}
var _ physic.SenseEnv = &Dev{}
