// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sgp30

import (
	"context"
	"encoding/binary"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
)

const (
	initAirQuality       uint16 = 0x2003
	measureAirQuality    uint16 = 0x2008
	getIAQBaseline       uint16 = 0x2015
	setIAQBaseline       uint16 = 0x201e
	setHumidity          uint16 = 0x2061
	measureTest          uint16 = 0x2032
	getFeatureSetVersion uint16 = 0x202f
	measureRawSignals    uint16 = 0x2050
	getTVOCBaseline      uint16 = 0x20b3
	setTVOCBaseline      uint16 = 0x2077

	i2CAddress uint16 = 0x58
)

// commandDuration maps the defined maximum measurement duration from the sensor
var commandDuration = map[uint16]time.Duration{
	initAirQuality:       time.Millisecond * 10,
	measureAirQuality:    time.Millisecond * 12,
	getIAQBaseline:       time.Millisecond * 10,
	setIAQBaseline:       time.Millisecond * 10,
	setHumidity:          time.Millisecond * 10,
	measureTest:          time.Millisecond * 220,
	getFeatureSetVersion: time.Millisecond * 10,
	measureRawSignals:    time.Millisecond * 25,
	getTVOCBaseline:      time.Millisecond * 10,
	setTVOCBaseline:      time.Millisecond * 10,
}

// commandResponseLength maps the defined response length including the CRC
var commandResponseLength = map[uint16]int{
	measureAirQuality:    6,
	getIAQBaseline:       6,
	measureTest:          3,
	getFeatureSetVersion: 3,
	measureRawSignals:    6,
	getTVOCBaseline:      3,
}

// CO2 represents the current carbon dioxide value in ppm
type CO2 uint16

func (c CO2) String() string {
	return strconv.Itoa(int(c)) + "ppm"
}

func (c *CO2) set(b []byte) {
	*c = (CO2)(binary.BigEndian.Uint16(b))
}

// TVOC represents the current total volatile organic compounds value in ppb
type TVOC uint16

func (t TVOC) String() string {
	return strconv.Itoa(int(t)) + "ppb"
}

func (t *TVOC) set(b []byte) {
	*t = (TVOC)(binary.BigEndian.Uint16(b))
}

// Env represents measurements from an environmental sensor.
type Env struct {
	CO2  CO2
	TVOC TVOC
}

// NewI2C returns an object that communicates over I2C to SGP30 environmental sensor.
//
// The address must be 0x58.
func NewI2C(b i2c.Bus, ctx context.Context) (*Dev, error) {
	d := &Dev{
		d: &i2c.Dev{Bus: b, Addr: i2CAddress},
		env: Env{
			CO2:  400,
			TVOC: 0,
		},
	}
	if err := d.makeDev(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

// Dev is a handle to an initialized SGP30 device.
type Dev struct {
	d   conn.Conn
	mu  sync.Mutex
	env Env
}

// AirQuality return the value struct for the sensor
func (d *Dev) AirQuality() Env {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.env
}

func (d *Dev) makeDev(ctx context.Context) error {
	// Sending  a "sgp30_iaq_init" command starts the air quality measurement
	if err := d.initAirQuality(); err != nil {
		return err
	}

	// After the "sgp30_iaq_init" command, a "sgp30_measure_iaq" command has to be sent in regular
	// intervals of 1s to ensure proper operation of the dynamic baseline compensation algorithm.
	if err := d.measure(); err != nil {
		log.Print(err)
	}

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := d.measure(); err != nil {
					log.Print(err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (d *Dev) initAirQuality() error {
	err := d.writeCommand(initAirQuality)
	if err == nil {
		time.Sleep(time.Second * 20)
	}
	return err
}

func (d *Dev) measure() error {
	buf := make([]byte, commandResponseLength[measureAirQuality])
	if err := d.readCommand(measureAirQuality, buf); err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.env.CO2.set(buf[0:2])
	d.env.TVOC.set(buf[3:5])

	return nil
}

func (d *Dev) readCommand(cmd uint16, b []byte) error {
	if len(b) != commandResponseLength[cmd] {
		return errors.New("response length mismatch")
	}

	regAddr := []byte{byte(cmd >> 8), byte(cmd & 0xFF)}
	if err := d.d.Tx(regAddr, nil); err != nil {
		return err
	}
	time.Sleep(commandDuration[cmd])

	return d.d.Tx(nil, b)
}

func (d *Dev) writeCommand(cmd uint16) error {
	regAddr := []byte{byte(cmd >> 8), byte(cmd & 0xFF)}
	if err := d.d.Tx(regAddr, nil); err != nil {
		return err
	}
	time.Sleep(commandDuration[cmd])
	return nil
}
