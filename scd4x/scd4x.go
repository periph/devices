// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package scd4x

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
)

// PPM=Parts Per Million. Units of measure for CO2 concentration.
type PPM int

// Sensor Variant type
type Variant int

const (
	SCD40 Variant = iota
	SCD41
)

// Type of reset to perform.
type ResetMode int

const (
	ResetFactory ResetMode = iota
	// Reset to last values stored in EEPROM
	ResetEEPROM
)

const (
	// These devices only support this i2c address.
	SensorAddress uint16 = 0x62
)

type cmd uint16

// Structure to simplify sending commands to the device.
type command struct {
	// The 16-bit command words.
	cmdWord cmd
	// The expected number of bytes returned. 0, 3, or 9.
	responseSize int
	// True if this command is permitted while the sensor is running in
	// acquisition mode.
	whileSensing bool
}

// The various implemented commands.

var cmdStartMeasurement = command{
	cmdWord: 0x21b1,
}

var cmdReadMeasurement = command{
	cmdWord:      0xec05,
	responseSize: 9,
	whileSensing: true,
}

var cmdStopMeasurement = command{
	cmdWord:      0x3f86,
	whileSensing: true,
}
var cmdGetTemperatureOffset = command{
	cmdWord:      0x2318,
	responseSize: 3,
}
var cmdSetTemperatureOffset = command{
	cmdWord: 0x241d,
}
var cmdGetSensorAltitude = command{
	cmdWord:      0x2322,
	responseSize: 3,
}
var cmdSetSensorAltitude = command{
	cmdWord: 0x2427,
}
var cmdGetAmbientPressure = command{
	cmdWord:      0xe000,
	responseSize: 3,
	whileSensing: true,
}
var cmdSetAmbientPressure = command{
	cmdWord:      0xe000,
	whileSensing: true,
}
var cmdSetASCEnabled = command{
	cmdWord: 0x2416,
}
var cmdGetASCEnabled = command{
	cmdWord:      0x2313,
	responseSize: 3,
}
var cmdGetASCTarget = command{
	cmdWord:      0x233f,
	responseSize: 3,
}
var cmdSetASCTarget = command{
	cmdWord: 0x243a,
}
var cmdGetDataReadyStatus = command{
	cmdWord:      0xe4b8,
	responseSize: 3,
	whileSensing: true,
}
var cmdPersistSettings = command{
	cmdWord: 0x3615,
}
var cmdGetSerialNumber = command{
	cmdWord:      0x3682,
	responseSize: 9,
}
var cmdPerformFactoryReset = command{
	cmdWord: 0x3632,
}
var cmdReinit = command{
	cmdWord: 0x3646,
}
var cmdGetSensorVariant = command{
	cmdWord:      0x202f,
	responseSize: 3,
}
var cmdGetASCInitialPeriod = command{
	cmdWord:      0x2340,
	responseSize: 3,
}
var cmdSetASCInitialPeriod = command{
	cmdWord: 0x2445,
}
var cmdGetASCStandardPeriod = command{
	cmdWord:      0x234b,
	responseSize: 3,
}
var cmdSetASCStandardPeriod = command{
	cmdWord: 0x244e,
}
var cmdWakeUp = command{
	cmdWord: 0x36f6,
}

// DevConfig is the current running configuration of the device. Values prefixed
// with ASC refer to Auto-Self-Calibration. Use Dev.GetConfiguration() to read
// the value, and Dev.SetConfiguration() to apply changes.
//
// Refer to the datasheet for more information on settings.
type DevConfig struct {
	// Ambient pressure value. Used to adjust operation of sensor.
	AmbientPressure physic.Pressure
	// Automatic-Self-Calibration enabled. True or false.
	ASCEnabled bool
	// Refer to datasheet for usage.
	ASCInitialPeriod time.Duration
	// Refer to datasheet for usage.
	ASCStandardPeriod time.Duration
	// Target CO2 concentration for automatic self calibration. To obtain the
	// current value, visit:
	//
	// https://www.co2.earth/daily-co2
	ASCTarget PPM
	// Sensor altitude in metres. Alternative method to adjust ambient pressure
	// for sensor correction.
	SensorAltitude physic.Distance
	// The 48 bit unique serial number of the device. Read-Only
	SerialNumber int64
	// Offset temperature added to reading. Refer to the datasheet for usage.
	TemperatureOffset physic.Temperature
	// The Type of sensor. SCD40 or SCD41. Read-Only
	SensorType Variant
}

// Dev represents an SCD4x device.
type Dev struct {
	// The i2c bus device.
	d *i2c.Dev
	// channel to halt SenseContinuous
	chHalt chan bool
	mu     sync.Mutex
	// True if the device is in continuous sense mode.
	sensing bool
}

func (ppm *PPM) String() string {
	return fmt.Sprintf("%d PPM", *ppm)
}

// The sensor reading. Returns CO2 PPM, Temperature, and Humidity.
type Env struct {
	physic.Env
	CO2 PPM
}

// Return the sensor readings in string format.
func (e *Env) String() string {
	return fmt.Sprintf("Temperature: %s Humidity: %s CO2: %s", e.Temperature.String(), e.Humidity.String(), e.CO2.String())
}

// NewI2c creates a new SCD4x sensor using the supplied bus and address.
// The constant value SensorAddress should be supplied as the value for
// addr.
func NewI2C(b i2c.Bus, addr uint16) (*Dev, error) {
	d := &Dev{d: &i2c.Dev{Bus: b, Addr: addr}, chHalt: nil}
	return d, d.start()
}

// GetConfiguration returns a structure containing all of the scd4x configuration
// variables. You can then alter settings and call SetConfiguration with it.
//
// To examine the device use:
//
//	cfg, _ :=dev.GetConfiguration()
//	fmt.Printf("Configuration=%#v\n", cfg)
func (d *Dev) GetConfiguration() (*DevConfig, error) {

	cfg := &DevConfig{}
	var words []uint16
	var err error

	if words, err = d.sendCommand(cmdGetAmbientPressure, nil); err != nil {
		return nil, err
	}
	cfg.AmbientPressure = physic.Pascal * 100 * physic.Pressure(words[0])

	if words, err = d.sendCommand(cmdGetASCEnabled, nil); err != nil {
		return nil, err
	}
	cfg.ASCEnabled = words[0] != 0

	if words, err = d.sendCommand(cmdGetASCInitialPeriod, nil); err != nil {
		return nil, err
	}
	cfg.ASCInitialPeriod = time.Hour * time.Duration(words[0])

	if words, err = d.sendCommand(cmdGetASCStandardPeriod, nil); err != nil {
		return nil, err
	}
	cfg.ASCStandardPeriod = time.Hour * time.Duration(words[0])

	if words, err = d.sendCommand(cmdGetASCTarget, nil); err != nil {
		return nil, err
	}
	cfg.ASCTarget = PPM(words[0])

	if words, err = d.sendCommand(cmdGetSerialNumber, nil); err != nil {
		return nil, err
	}
	cfg.SerialNumber = int64(words[0])<<32 | int64(words[1])<<16 | int64(words[2])

	if words, err = d.sendCommand(cmdGetSensorVariant, nil); err != nil {
		return nil, err
	}
	if (words[0]>>11)&0x07 == 0 {
		cfg.SensorType = SCD40
	} else {
		cfg.SensorType = SCD41
	}

	if words, err = d.sendCommand(cmdGetSensorAltitude, nil); err != nil {
		return nil, err
	}
	cfg.SensorAltitude = physic.Distance(words[0]) * physic.Metre

	if words, err = d.sendCommand(cmdGetTemperatureOffset, nil); err != nil {
		return nil, err
	}
	cfg.TemperatureOffset = countToOffset(words[0])

	return cfg, nil
}

// SetConfiguration alters the configuration of the sensor. Note that this call
// does not persist the settings to EEPROM. You need to call Persist() to
// commit the writes to EEPROM. If you do not persist changes, then those settings
// will be lost when the unit is power-cycled.
func (d *Dev) SetConfiguration(newCfg *DevConfig) error {

	_ = d.Halt()
	d.mu.Lock()
	defer d.mu.Unlock()

	w := make([]uint16, 1)
	currentConfig, err := d.GetConfiguration()
	if err != nil {
		return fmt.Errorf("scd4x GetConfiguration(): %w", err)
	}

	if currentConfig.AmbientPressure != newCfg.AmbientPressure {
		w[0] = uint16(newCfg.AmbientPressure / (100 * physic.Pascal))
		_, err := d.sendCommand(cmdSetAmbientPressure, w)
		if err != nil {
			return err
		}
	}

	if currentConfig.ASCEnabled != newCfg.ASCEnabled {

		if newCfg.ASCEnabled {
			w[0] = 1
		} else {
			w[0] = 0
		}
		_, err := d.sendCommand(cmdSetASCEnabled, w)
		if err != nil {
			return err
		}
	}

	if currentConfig.ASCInitialPeriod != newCfg.ASCInitialPeriod {
		if newCfg.ASCInitialPeriod%4 != 0 {
			return fmt.Errorf("scd4x: invalid initial period %d. must be a mulitple of 4", newCfg.ASCInitialPeriod)
		}
		w[0] = uint16(newCfg.ASCInitialPeriod / time.Hour)
		_, err := d.sendCommand(cmdSetASCInitialPeriod, w)
		if err != nil {
			return err
		}
	}

	if currentConfig.ASCStandardPeriod != newCfg.ASCStandardPeriod {
		if newCfg.ASCStandardPeriod%4 != 0 {
			return fmt.Errorf("scd4x: invalid standard period %d. must be a mulitple of 4", newCfg.ASCStandardPeriod)
		}
		w[0] = uint16(newCfg.ASCStandardPeriod / time.Hour)
		_, err := d.sendCommand(cmdSetASCStandardPeriod, w)
		if err != nil {
			return err
		}
	}

	if currentConfig.ASCTarget != newCfg.ASCTarget {
		w[0] = uint16(newCfg.ASCTarget)
		_, err := d.sendCommand(cmdSetASCTarget, w)
		if err != nil {
			return err
		}
	}

	if currentConfig.SensorAltitude != newCfg.SensorAltitude {
		w[0] = uint16(newCfg.SensorAltitude / physic.Metre)
		_, err := d.sendCommand(cmdSetSensorAltitude, w)
		if err != nil {
			return err
		}
	}

	if currentConfig.TemperatureOffset != newCfg.TemperatureOffset {
		val := float64(newCfg.TemperatureOffset.Celsius()) * (float64(65535) / float64(175))
		w[0] = uint16(val)
		_, err := d.sendCommand(cmdSetTemperatureOffset, w)
		if err != nil {
			return err
		}
	}

	return nil
}

// Halt stops continuous sensing if enabled, and if a SenseContinuous operation
// is in progress, it too is halted.
func (d *Dev) Halt() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.sensing {
		if d.chHalt != nil {
			close(d.chHalt)
		}
		d.sensing = false
		_, err := d.sendCommand(cmdStopMeasurement, nil)
		time.Sleep(550 * time.Millisecond)
		if err != nil {
			return err
		}
	}
	return nil
}

// Persist writes the current running configuration to the sensor EEPROM for
// use on the next power-up.
func (d *Dev) Persist() error {
	_, err := d.sendCommand(cmdPersistSettings, nil)
	return err
}

// Reset performs either a factory reset, or a re-load of settings from EEPROM
// depending on the value of mode. During development, it was noticed that
// ResetFactory DOES NOT reset AmbientPressure to 0.
func (d *Dev) Reset(mode ResetMode) error {
	var err error
	if mode == ResetFactory {
		_, err = d.sendCommand(cmdPerformFactoryReset, nil)
	} else if mode == ResetEEPROM {
		_, err = d.sendCommand(cmdReinit, nil)
	} else {
		err = fmt.Errorf("scd4x: invalid reset mode 0x%x", mode)
	}
	return err
}

func calcCRC(bytes []byte) byte {
	polynomial := byte(0x31)
	crc := byte(0xff)
	for ix := range len(bytes) {
		crc ^= bytes[ix]
		for crc_bit := byte(8); crc_bit > 0; crc_bit-- {
			if (crc & 0x80) == 0x80 {
				crc = (crc << 1) ^ polynomial
			} else {
				crc = (crc << 1)
			}
		}
	}
	return crc
}

// makeWriteData converts the slice of word values into byte values with the
// CRC following.
func makeWriteData(data []uint16) []byte {
	bytes := make([]byte, len(data)*3)
	for ix, val := range data {
		bytes[ix*3] = byte((val >> 8) & 0xff)
		bytes[ix*3+1] = byte(val & 0xff)
		bytes[ix*3+2] = calcCRC(bytes[ix*3 : ix*3+2])
	}
	return bytes
}

// All commands to read or write to the sensor go through this function.
func (d *Dev) sendCommand(cmd command, writeData []uint16) ([]uint16, error) {

	if d.sensing && !cmd.whileSensing {
		// We're in sense mode and this command isn't compatible. Stop sensing.
		if err := d.Halt(); err != nil {
			return nil, err
		}
	}

	w := make([]byte, 2)
	w[0] = byte((cmd.cmdWord >> 8) & 0xff)
	w[1] = byte(cmd.cmdWord & 0xff)
	if writeData != nil {
		writeBytes := makeWriteData(writeData)
		w = append(w, writeBytes...)
	}
	var r []byte
	if cmd.responseSize > 0 {
		r = make([]byte, cmd.responseSize)
	}

	err := d.d.Tx(w, r)
	if err != nil {
		return nil, fmt.Errorf("scd4x cmd 0x%x: %w", cmd.cmdWord, err)
	}
	if cmd.responseSize == 0 {
		return nil, nil
	}

	// OK, we need to convert the bytes into a slice of words and
	// verify the CRC as we go.
	result := make([]uint16, cmd.responseSize/3)
	for ix := range len(result) {
		crc := calcCRC(r[ix*3 : ix*3+2])
		if r[ix*3+2] != crc {
			return nil, fmt.Errorf("scd4x cmd 0x%x: invalid crc", cmd.cmdWord)
		}

		word := uint16(r[ix*3])<<8 | uint16(r[ix*3+1])

		result[ix] = word
	}

	return result, nil
}

// start continuous sensing.
func (d *Dev) start() error {
	if d.sensing {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.sendCommand(cmdWakeUp, nil)
	if err != nil {
		// If an SCD4x is in measurement mode, then any non-measurement mode
		// command will return an error. In that case, send a stop measurement
		// command, wait the specified time and try sending a re-init.
		_, _ = d.sendCommand(cmdStopMeasurement, nil)
		time.Sleep(550 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)

	_, err = d.sendCommand(cmdStartMeasurement, nil)
	if err == nil {
		d.sensing = true
	}
	return err
}

// Formula used for temperature offset calculation.
func countToOffset(count uint16) physic.Temperature {
	frac := 175.0 / 65535.0
	return physic.Temperature(frac * float64(count))
}

// countToTemp converts a device count to Temperature
func countToTemp(count uint16) physic.Temperature {
	frac := float64(count) / 65535.0
	result := -45 + 175*frac
	return physic.ZeroCelsius + physic.Temperature(float64(physic.Celsius)*result)
}

func countToHumidity(count uint16) physic.RelativeHumidity {
	frac := float64(count) / 65535.0
	return physic.RelativeHumidity(frac * 100.0 * float64(physic.PercentRH))
}

// Sense returns readings (Temperature, Humidity, and CO2 concentration in PPM)
// from the device. Note that in normal acquisition mode, the minimum reading
// period is 5 seconds. If you call this function more frequently than this,
// it will block until data is ready.
func (d *Dev) Sense(env *Env) error {
	env.Temperature = 0
	env.Humidity = 0
	env.CO2 = 0
	env.Pressure = 0

	if !d.sensing {
		err := d.start()
		if err != nil {
			return err
		}
		time.Sleep(5 * time.Second)
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	ready := false
	mask := uint16(1<<11 - 1)
	tCutoff := time.Now().Unix() + 6
	for !ready && time.Now().Unix() < tCutoff {
		words, err := d.sendCommand(cmdGetDataReadyStatus, nil)
		ready = err == nil && (words[0]&mask) > 0
		if !ready {
			time.Sleep(time.Second)
		}
	}
	if !ready {
		return errors.New("scd4x: timeout waiting for data ready status")
	}
	words, err := d.sendCommand(cmdReadMeasurement, nil)
	if err != nil {
		return err
	}
	env.CO2 = PPM(words[0])
	env.Temperature = countToTemp(words[1])
	env.Humidity = countToHumidity(words[2])
	return nil
}

// SenseContinuous continuously reads the sensor on the specified duration, and
// writes readings to the returned channel. The sense time for the scd4x device
// is 5 seconds in normal acquisition mode. If you specify a shorter period than
// that, the routine will spin until the device indicates a reading is ready. To
// terminate a continuous sense, call Halt().
func (d *Dev) SenseContinuous(interval time.Duration) (<-chan Env, error) {
	if d.chHalt != nil {
		return nil, errors.New("scd4x: SenseContinuous() running already")
	}
	if !d.sensing {
		if err := d.start(); err != nil {
			return nil, err
		}
	}
	channelSize := 16
	channel := make(chan Env, channelSize)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(channel)
		if d.chHalt == nil {
			d.chHalt = make(chan bool)
		}

		defer func() { d.chHalt = nil }()

		for {
			select {
			case <-d.chHalt:
				return
			case <-ticker.C:
				// do the reading and write to the channel.
				e := Env{}
				err := d.Sense(&e)
				if err == nil && len(channel) < channelSize {
					channel <- e
				}
			}
		}
	}()
	return channel, nil
}

// Precision returns the sensor's resolution, or minimum value between steps the
// device can make. The specified precision is 1 PPM for CO2, 1/65535 for temperature
// and humidity.
func (d *Dev) Precision(env *Env) {
	countIncrement := float64(1.0) / float64((1<<16)-1)
	env.Temperature = physic.Temperature(countIncrement * float64(physic.Celsius))
	env.Pressure = 0
	env.Humidity = physic.RelativeHumidity(float64(physic.PercentRH) * countIncrement)
	env.CO2 = 1
}

func (d *Dev) String() string {
	return fmt.Sprintf("scd4x: %s", d.d.String())
}
