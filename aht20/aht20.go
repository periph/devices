// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package aht20

import (
	"errors"
	"fmt"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
	"sync"
	"time"
)

const deviceAddress = 0x38

const (
	cmdStatus     byte = 0x71
	cmdInitialize byte = 0xBE
	cmdMeasure    byte = 0xAC
	cmdSoftReset  byte = 0xBA
)

const (
	bitBusy        byte = 1 << 7
	bitInitialized byte = 1 << 3
)

var (
	argsInitialize = []byte{cmdInitialize, 0x08, 0x00}
	argsMeasure    = []byte{cmdMeasure, 0x33, 0x00}
)

const crc8Polynomial = uint8(0b00110001) // p(x) = x^8 + x^5 + x^4 + 1. x^8 is omitted due to byte size

type Dev struct {
	opts Opts
	d    *i2c.Dev
	mu   sync.Mutex
	stop chan struct{}
	wg   sync.WaitGroup
}

// Opts holds the configuration options for the device.
type Opts struct {
	// MeasurementReadTimeout is the timeout for reading a single measurement. The timeout only applies after the measurement triggering which itself takes 80ms. Default is 150ms. 0 means no timeout.
	MeasurementReadTimeout time.Duration
	// MeasurementWaitInterval is the interval between subsequent sensor value reads. This applies only if the measurement is not finished after the initial 80ms wait. Do not confuse this interval with SenseContinuous. Default is 10ms. Leave 0 to use default.
	MeasurementWaitInterval time.Duration
	// ValidateData enables data validation using CRC8. If enabled, the sensor will return an error if the data is corrupt. Default is true.
	ValidateData bool
}

// DefaultOpts holds the default configuration options for the device.
var DefaultOpts = Opts{
	MeasurementReadTimeout:  150 * time.Millisecond,
	MeasurementWaitInterval: 10 * time.Millisecond,
	ValidateData:            true,
}

// NewI2C returns an object that communicates over I²C to AHT20 environmental sensor. The sensor
// will be calibrated and initialized if it is not already. The Opts can be nil.
func NewI2C(b i2c.Bus, opts *Opts) (*Dev, error) {
	if opts == nil {
		opts = &DefaultOpts
	}
	if opts.MeasurementWaitInterval <= 0 {
		opts.MeasurementWaitInterval = 10 * time.Millisecond
	}

	d := &Dev{d: &i2c.Dev{Bus: b, Addr: deviceAddress}, opts: *opts}
	if err, initialized := d.isInitialized(); err != nil {
		return nil, errors.Join(fmt.Errorf("could read sensor status"), err)
	} else if !initialized {
		if err := d.initialize(); err != nil {
			return nil, errors.Join(fmt.Errorf("could not calibrate sensor"), err)
		}
	}
	return d, nil
}

// Sense implements physic.SenseEnv. It returns the current temperature and humidity, the pressure
// is always 0 since the AH20 does not measure pressure. The measurement takes at least 80ms. If the
// configured timeout is reached, a ReadTimeoutError is returned. If the data is corrupt, a
// DataCorruptionError is returned. If the sensor is not initialized, a NotInitializedError is
// returned.
func (d *Dev) Sense(e *physic.Env) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// trigger measurement
	if err := d.d.Tx(argsMeasure, nil); err != nil {
		return err
	}
	time.Sleep(80 * time.Millisecond) // wait for 80ms according to datasheet

	end := time.Now().Add(d.opts.MeasurementReadTimeout)
	data := make([]byte, 7)
	for d.opts.MeasurementReadTimeout <= 0 || time.Now().Before(end) {

		// read measurement
		if err := d.d.Tx(nil, data); err != nil {
			return err
		} else if d.opts.ValidateData && calculateCRC8(data[0:6]) != data[6] {
			return &DataCorruptionError{}
		}

		if data[0]&bitInitialized == 0 {
			return &NotInitializedError{}
		} else if data[0]&bitBusy == 0 {
			hRaw := uint32(data[1])<<12 | uint32(data[2])<<4 | uint32(data[3])>>4
			tRaw := (uint32(data[3])&0xF)<<16 | uint32(data[4])<<8 | uint32(data[5])

			humidityRH := float64(hRaw) / 1048576.0 * 100.0
			temperatureC := (float64(tRaw)/1048576.0)*200 - 50.0

			e.Humidity = physic.RelativeHumidity(humidityRH * float64(physic.PercentRH))
			e.Temperature = physic.Temperature(temperatureC*float64(physic.Kelvin)) + physic.ZeroCelsius
			return nil
		}
		time.Sleep(d.opts.MeasurementWaitInterval) // wait until measurement is ready
	}

	return &ReadTimeoutError{}
}

// SenseContinuous implements physic.SenseEnv. It returns a channel that will
// receive a measurement every interval. It is the caller's responsibility to call Halt() when done.
// The sensor tries to read the measurement at the given interval however it may take longer if the
// sensor is busy.
func (d *Dev) SenseContinuous(interval time.Duration) (<-chan physic.Env, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.wg.Add(1)

	sensing := make(chan physic.Env)
	d.stop = make(chan struct{})
	go func() {
		defer d.wg.Done()
		defer close(sensing)
		dMeasurement := 100 * time.Millisecond // duration of last measurement
		for {
			select {
			case <-d.stop:
				return
			case <-time.After(interval - dMeasurement):
				var e physic.Env
				now := time.Now()
				if err := d.Sense(&e); err == nil {
					sensing <- e
				}
				dMeasurement = time.Since(now)
			}
		}
	}()
	return sensing, nil
}

// Precision implements physic.SenseEnv.
func (d *Dev) Precision(e *physic.Env) {
	e.Temperature = 10 * physic.MilliKelvin
	e.Humidity = 24 * physic.MilliRH
}

// SoftReset resets the sensor. It includes a reboot and a re-calibration.
func (d *Dev) SoftReset() error {
	if err := d.d.Tx([]byte{cmdSoftReset}, nil); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond) // wait for 20ms according to datasheet
	return nil
}

// Halt stops the AHT20 from acquiring measurements as initiated by SenseContinuous().
func (d *Dev) Halt() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stop == nil {
		return nil
	}
	close(d.stop)
	d.wg.Wait()
	d.stop = nil
	return nil
}

func (d *Dev) isInitialized() (error, bool) {
	var data byte
	if err := d.d.Tx([]byte{cmdStatus}, []byte{data}); err != nil {
		return err, false
	}
	return nil, data&bitInitialized == 1
}

func (d *Dev) initialize() error {
	if err := d.d.Tx(argsInitialize, nil); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond) // wait for 10ms according to datasheet
	return nil
}

func calculateCRC8(data []byte) uint8 {
	var crc uint8 = 0xFF // initial value according to datasheet

	for _, b := range data {
		crc ^= b
		for i := 0; i < 8; i++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ crc8Polynomial // 0x07 is the polynomial
			} else {
				crc <<= 1
			}
		}
	}

	return crc
}
