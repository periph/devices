// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This package provides a driver for the Texas Instruments HDC3021/3022
// I2C Temperature/Humidity Sensors. This is a high accuracy sensor with
// very good resolution.
//
// Datasheet
//
//	https://www.ti.com/lit/ds/symlink/hdc3022.pdf
package hdc302x

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
)

type SampleRate uint16

const (
	// Constants for the sample rate to use for the measurements. The datasheet
	// recommends not sampling more often than once per second to avoid self-heating
	// of the sensor.
	//
	// Every other second
	RateHalfHertz SampleRate = iota
	// Sample 1x Second.
	RateHertz
	RateTwoHertz
	RateFourHertz
	Rate10Hertz
)

// Dev represents a hdc302x sensor.
type Dev struct {
	d          *i2c.Dev
	shutdown   chan struct{}
	mu         sync.Mutex
	sampleRate SampleRate
	halted     bool
}

// The alert function works with pairs of values Temperature/Humidity. A
// Threshold is a set of humidity/temperature values defining an upper or
// lower limit for alerts.
type Threshold struct {
	Humidity    physic.RelativeHumidity
	Temperature physic.Temperature
}

// For alert and clear, there is a pair of temperatures. For example,
// a low alert value, and a clear low alert value. There's a value
// for each measurement parameter.
type ThresholdPair struct {
	Low  Threshold
	High Threshold
}

type StatusWord uint16

const (
	// Status flags returned by ReadStatus()
	StatusActiveAlerts  StatusWord = 1 << 15
	StatusHeaterEnabled StatusWord = 1 << 13
	// Mirrored on the alert pin.
	StatusRHTrackingAlert StatusWord = 1 << 11
	// Also reflected on alert pin
	StatusTempTrackingAlert     StatusWord = 1 << 10
	StatusRHHighTrackingAlert   StatusWord = 1 << 9
	StatusRHLowTrackingAlert    StatusWord = 1 << 8
	StatusTempHighTrackingAlert StatusWord = 1 << 7
	StatusTempLowTrackingAlert  StatusWord = 1 << 6
	StatusDeviceReset           StatusWord = 1 << 4
	// Set if there was a CRC error on the last write command.
	StatusLastWriteCRCFailure StatusWord = 1 << 0
)

// Configuration provides information about the running device's config.
type Configuration struct {
	// Device unique ID. Read-Only
	SerialNumber int64
	// Numeric vendor ID. Read-Only
	VendorID uint16
	// Status Word. Refer to the Status* constants above, and the datasheet for
	// usage.
	Status StatusWord
	// refer to the Rate constants. Read-Only
	SampleRate SampleRate
	// Offset for RH calculation. Note that these offsets are approximate,
	// so a request to set the offset to -5%rH may result in an offset of
	// -4.8%rH. This is an artifact of the device's offset implementation.
	// Refer to the datasheet for more information.
	HumidityOffset physic.RelativeHumidity
	// Offset for Temp result. Note that the data sheet states this is not
	// used in the RH calculation.
	TemperatureOffset physic.Temperature

	// High/Low thresholds for triggering alerts. As with the offsets,
	// written values are not precise.
	AlertThresholds ThresholdPair
	// High/Low threshold for clearing alerts.
	ClearThresholds ThresholdPair
}

const (
	// The default i2c bus address for this device.
	DefaultSensorAddress uint16 = 0x44
)

type HeaterPower uint16

const (
	// Constants for setting the heater's power setting.
	PowerFull    HeaterPower = 0x3fff
	PowerHalf    HeaterPower = 0x03ff
	PowerQuarter HeaterPower = 0x9f
	PowerOff     HeaterPower = 0
)

type devCommand []byte

// Sample Rate commands
var measure2Seconds = devCommand{0x20, 0x32}
var measureSecond = devCommand{0x21, 0x30}
var measure2xSecond = devCommand{0x22, 0x36}
var measure4xSecond = devCommand{0x23, 0x34}
var measure10xSecond = devCommand{0x27, 0x37}

var sampleRateCommands = []devCommand{measure2Seconds, measureSecond, measure2xSecond, measure4xSecond, measure10xSecond}
var sampleRateDurations = []time.Duration{2 * time.Second, time.Second, 500 * time.Millisecond, 250 * time.Millisecond, 100 * time.Millisecond}

// Other device commands
var clearStatus = devCommand{0x30, 0x41}
var disableHeater = devCommand{0x30, 0x66}
var enableHeater = devCommand{0x30, 0x6d}
var read = devCommand{0xe0, 0x0}
var readSetHeater = devCommand{0x30, 0x6e}
var readSetOffsets = devCommand{0xa0, 0x04}
var readStatus = devCommand{0xf3, 0x2d}
var readVendorID = devCommand{0x37, 0x81}
var reset = devCommand{0x30, 0xa2}
var stopContinuousReadings = devCommand{0x30, 0x93}

// read/write alert threshold commands.
var readLowAlertThresholds = devCommand{0xe1, 0x02}
var readHighAlertThresholds = devCommand{0xe1, 0x1f}
var readLowClearThresholds = devCommand{0xe1, 0x09}
var readHighClearThresholds = devCommand{0xe1, 0x14}
var writeLowAlertThresholds = devCommand{0x61, 0x00}
var writeHighAlertThresholds = devCommand{0x61, 0x1d}
var writeLowClearThresholds = devCommand{0x61, 0x0b}
var writeHighClearThresholds = devCommand{0x61, 0x16}

var errInvalidCRC = errors.New("hdc302x: invalid crc")

const (
	// Magic numbers for count to value conversions.
	temperatureOffset float64 = -45.0
	temperatureScalar float64 = 175.0
	humidityScalar    float64 = 100.0
	scaleDivisor      float64 = 65535.0
)

// NewI2C returns a new HDC302x sensor using the specified bus, address, and
// sample rate.
func NewI2C(b i2c.Bus, addr uint16, sampleRate SampleRate) (*Dev, error) {
	dev := &Dev{d: &i2c.Dev{Bus: b, Addr: addr}, shutdown: nil, sampleRate: sampleRate}
	return dev, dev.start()
}

// send continuous measurement start command.
func (dev *Dev) start() error {
	if err := dev.d.Tx(sampleRateCommands[dev.sampleRate], nil); err != nil {
		return fmt.Errorf("hdc302x: init %w", err)
	}
	// Sleep for a minimum of one sample acquisition period. If you
	// read before a sample has acquired, you get remote I/O error.
	time.Sleep(sampleRateDurations[dev.sampleRate])
	dev.halted = false
	return nil
}

// Convert the raw count to a temperature.
func countToTemperature(bytes []byte) physic.Temperature {
	count := (uint16(bytes[0]) << 8) | uint16(bytes[1])
	f := float64(count)/float64(scaleDivisor)*temperatureScalar + temperatureOffset
	t := physic.ZeroCelsius + physic.Temperature(f*float64(physic.Celsius))
	return t
}

// convert the raw count to a humidity value.
func countToHumidity(bytes []byte) physic.RelativeHumidity {
	count := (uint16(bytes[0]) << 8) | uint16(bytes[1])
	f := float64(count) / float64(scaleDivisor) * humidityScalar
	return physic.RelativeHumidity(f * float64(physic.PercentRH))
}

func crc8(bytes []byte) byte {
	var crc byte = 0xff
	for _, val := range bytes {
		crc ^= val
		for range 8 {
			if (crc & 0x80) == 0 {
				crc <<= 1
			} else {
				crc = (byte)((crc << 1) ^ 0x31)
			}
		}
	}
	return crc
}

// Halt shuts down the device. If a SenseContinuous operation is in progress,
// its aborted. Implements conn.Resource
func (dev *Dev) Halt() error {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	if dev.shutdown != nil {
		close(dev.shutdown)
	}
	var err error
	if !dev.halted {
		dev.halted = true
		err = dev.d.Tx(stopContinuousReadings, nil)
	}
	return err
}

// Sense reads temperature and humidity from the device and writes the value to
// the specified env variable. Implements physic.SenseEnv.
func (dev *Dev) Sense(env *physic.Env) error {
	env.Temperature = 0
	env.Pressure = 0
	env.Humidity = 0
	res := make([]byte, 6)
	dev.mu.Lock()
	defer dev.mu.Unlock()
	if dev.halted {
		if err := dev.start(); err != nil {
			return err
		}
	}
	if err := dev.d.Tx(read, res); err != nil {
		return fmt.Errorf("hdc302x: %w", err)
	}
	if crc8(res[:2]) != res[2] || crc8(res[3:5]) != res[5] {
		return errInvalidCRC
	}
	env.Temperature = countToTemperature(res)
	env.Humidity = countToHumidity(res[3:])
	return nil
}

func temperatureToFloat64(temp physic.Temperature) float64 {
	return float64(temp) / float64(physic.Celsius)
}

func humidityToFloat64(humidity physic.RelativeHumidity) float64 {
	return float64(humidity) / float64(physic.PercentRH)
}

// SenseContinuous continuously reads from the device and writes the value to
// the returned channel. Implements physic.SenseEnv. To terminate the
// continuous read, call Halt().
//
// If interval is less than the device sample period, an error is returned.
func (dev *Dev) SenseContinuous(interval time.Duration) (<-chan physic.Env, error) {

	if dev.shutdown != nil {
		return nil, errors.New("hdc302x: SenseContinuous already running")
	}

	if interval < sampleRateDurations[dev.sampleRate] {
		return nil, errors.New("hdc302x: sample interval is < device sample rate")
	}

	dev.shutdown = make(chan struct{})
	chResult := make(chan physic.Env, 16)
	go func(ch chan physic.Env) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(ch)
		for {
			select {
			case <-dev.shutdown:
				dev.shutdown = nil
				return
			case <-ticker.C:
				env := physic.Env{}
				if err := dev.Sense(&env); err == nil {
					ch <- env
				}
			}
		}
	}(chResult)
	return chResult, nil
}

// Precision returns the sensor's precision, or minimum value between steps the
// device can measure. Refer to the datasheet for information on limits and
// accuracy.
func (dev *Dev) Precision(env *physic.Env) {
	env.Temperature = physic.Temperature(math.Round(temperatureScalar / scaleDivisor * float64(physic.Celsius)))
	env.Humidity = physic.RelativeHumidity(math.Round((float64(physic.PercentRH) * humidityScalar) / float64(scaleDivisor)))
	env.Pressure = 0
}

func (dev *Dev) readSerialNumber() int64 {
	var result int64
	cmd := []byte{0x36, 0x83}
	r := make([]byte, 3)
	// this is a 6 byte value read in 3 parts
	for range 3 {
		err := dev.d.Tx(cmd, r)
		if err != nil || (crc8(r[:2]) != r[2]) {
			return result
		}
		result = result<<16 | (int64(r[0])<<8 | int64(r[1]))
		cmd[1] += 1 // Increment the register to the next one.
	}
	return result
}

// read the alert / clear threshold values from the device.
func (dev *Dev) readAlertValues(cfg *Configuration) error {

	// Pair
	// 	Low
	//		Temp
	//		Humidity
	//	High
	//		Temp
	//		Humidity

	var cmds = []devCommand{readLowAlertThresholds, readHighAlertThresholds, readLowClearThresholds, readHighClearThresholds}
	var pairs = [2]*ThresholdPair{&cfg.AlertThresholds, &cfg.ClearThresholds}
	var threshold *Threshold

	r := make([]byte, 3)

	for ix, cmd := range cmds {
		pair := pairs[ix>>1]
		if ix%2 == 0 {
			threshold = &pair.Low
		} else {
			threshold = &pair.High
		}

		err := dev.d.Tx(cmd, r)
		if err != nil {
			return err
		}
		if crc8(r[:2]) != r[2] {
			return errInvalidCRC
		}
		wValue := uint16(r[0])<<8 | uint16(r[1])
		// The alert value is returned as a 16 bit words, where bits 0-8 are the
		// Temperature value, and bits 9-15 are the Humidity. The temperature
		// bits correspond to bits 7-15 of the temperature, and bits 9-15 of the
		// humidity. Refer to the datasheet.
		temp := &threshold.Temperature
		humidity := &threshold.Humidity
		*temp = physic.Temperature(((float64(uint16(wValue<<7)) * temperatureScalar) / scaleDivisor) * float64(physic.Celsius))
		*humidity = physic.RelativeHumidity(((float64(wValue&0xfe00) * humidityScalar) / scaleDivisor) * float64(physic.PercentRH))
	}

	return nil
}

// readOffsets returns temperature/humidity offset values stored to the device.
func (dev *Dev) readOffsets(cfg *Configuration) error {
	r := make([]byte, 3)
	if err := dev.d.Tx(readSetOffsets, r); err != nil {
		return fmt.Errorf("hdc302x: %w", err)
	}
	if crc8(r[:2]) != r[2] {
		return errInvalidCRC
	}

	// The result comes back as the humidity offset, followed by
	// the temperature offset. The offsets are computed by summing
	// the bits and applying the partial algorithm.

	h := uint16((r[0] & 0x7f)) << 7
	t := uint16((r[1] & 0x7f)) << 6
	rh := float64(h) / scaleDivisor * humidityScalar
	temp := (float64(t) * temperatureScalar) / scaleDivisor

	if r[0]&0x80 == 0x00 {
		rh *= -1.0
	}
	cfg.HumidityOffset = physic.RelativeHumidity(rh * float64(physic.PercentRH))

	if r[1]&0x80 == 0x00 {
		temp *= -1.0
	}
	cfg.TemperatureOffset = physic.Temperature(temp * float64(physic.Celsius))

	return nil
}

func (dev *Dev) readVendorID() (uint16, error) {
	r := make([]byte, 3)
	err := dev.d.Tx(readVendorID, r)
	if err == nil {
		vid := uint16(r[0])<<8 | uint16(r[1])
		return vid, nil
	}
	return 0, err
}

// ReadStatus returns the device's status word, and if successful, clears the
// status. Refer to the Status* constants and the datasheet for interpretation.
func (dev *Dev) ReadStatus() (StatusWord, error) {
	r := make([]byte, 3)
	if err := dev.d.Tx(readStatus, r); err != nil {
		return 0, err
	}
	if crc8(r[:2]) != r[2] {
		return 0, errInvalidCRC
	}
	_ = dev.d.Tx(clearStatus, nil)
	return StatusWord(r[0])<<8 | StatusWord(r[1]), nil
}

// Return the device's configuration settings. Includes alert values, offset
// values, and other information about the device.
func (dev *Dev) Configuration() (*Configuration, error) {
	cfg := &Configuration{SampleRate: dev.sampleRate}
	cfg.SerialNumber = dev.readSerialNumber()
	err := dev.readOffsets(cfg)
	if err != nil {
		return cfg, err
	}
	if cfg.VendorID, err = dev.readVendorID(); err != nil {
		return cfg, err
	}
	if cfg.Status, err = dev.ReadStatus(); err != nil {
		return cfg, err
	}

	err = dev.readAlertValues(cfg)

	return cfg, err
}

// setOffsets writes temperature and humidity offsets to the device.
// Refer to the datasheet for information on offsets. The critical
// thing to know is that the smallest offsets are ~0.2%RH, and ~
// 0.2 degrees C.
func (dev *Dev) setOffsets(cfg *Configuration) error {
	var w = []byte{readSetOffsets[0],
		readSetOffsets[1],
		computeHumidityOffsetByte(cfg.HumidityOffset),
		computeTemperatureOffsetByte(cfg.TemperatureOffset),
		0,
	}
	w[4] = crc8(w[2:4])
	return dev.d.Tx(w, nil)
}

// Refer to the datasheet. Essentially, the offsets are only a specific set of
// bit ranges.
func computeTemperatureOffsetByte(temp physic.Temperature) byte {
	var res byte
	fTemp := temperatureToFloat64(temp)
	if fTemp >= 0 {
		res |= 0x80
	} else {
		fTemp *= -1.0
	}
	for bit := 12; bit > 5; bit-- {
		offset := (float64(int64(1)<<bit) * temperatureScalar) / scaleDivisor
		if fTemp >= offset {
			fTemp -= offset
			res |= (1 << (bit - 6))
		}
	}
	return res
}

func computeHumidityOffsetByte(humidity physic.RelativeHumidity) byte {
	var res byte
	fHumidity := humidityToFloat64(humidity)
	if fHumidity >= 0 {
		res |= 0x80
	} else {
		fHumidity *= -1.0
	}
	for bit := 13; bit > 6; bit-- {
		offset := (float64(int64(1)<<bit) * humidityScalar) / scaleDivisor
		if fHumidity >= offset {
			fHumidity -= offset
			res |= (1 << (bit - 7))
		}
	}
	return res
}

// Reset performs a soft-reset of the device.
func (dev *Dev) Reset() error {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	err := dev.d.Tx(reset, nil)
	time.Sleep(time.Second)
	return err
}

// setThreshold sets a threshold pair for either alert, or clear alert.
// if typeAlert is true, it indicates the pair type is alert, otherwise
// it's clear alert.
func (dev *Dev) setThresholds(typeAlert bool, tp *ThresholdPair) error {
	var cmds = [][]devCommand{{writeLowAlertThresholds, writeHighAlertThresholds},
		{writeLowClearThresholds, writeHighClearThresholds}}

	pair := 1
	if typeAlert {
		pair = 0
	}
	var th *Threshold
	for ix := range 2 {
		if ix == 0 {
			th = &tp.Low
		} else {
			th = &tp.High
		}
		temp := temperatureToFloat64(th.Temperature)
		tempBits := uint16(0)
		for bit := 15; bit >= 0; bit-- {
			bitVal := (float64(uint16(1<<bit)) * temperatureScalar) / scaleDivisor
			if temp >= bitVal {
				temp -= bitVal
				tempBits |= (1 << bit)
			}
		}
		humidity := humidityToFloat64(th.Humidity)
		humBits := uint16(0)
		for bit := 15; bit >= 0; bit-- {
			bitVal := (float64(uint16(1<<bit)) * humidityScalar) / scaleDivisor
			if humidity >= bitVal {
				humidity -= bitVal
				humBits |= (1 << bit)
			}
		}
		wval := uint16(0)
		wval = (humBits & 0xfe00) | tempBits>>7
		w := []byte{cmds[pair][ix][0], cmds[pair][ix][1], byte(wval >> 8), byte(wval & 0xff), 0}
		w[4] = crc8(w[2:4])
		err := dev.d.Tx(w, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetConfiguration takes a modified configuration struct and
// applies it to the device.
func (dev *Dev) SetConfiguration(cfg *Configuration) error {
	_ = dev.Halt()
	dev.mu.Lock()
	defer dev.mu.Unlock()
	current, err := dev.Configuration()
	if err != nil {
		return err
	}
	if current.HumidityOffset != cfg.HumidityOffset || current.TemperatureOffset != cfg.TemperatureOffset {
		if err := dev.setOffsets(cfg); err != nil {
			return err
		}
	}

	if !current.AlertThresholds.Equals(&cfg.AlertThresholds) {
		if err := dev.setThresholds(true, &cfg.AlertThresholds); err != nil {
			return err
		}
	}

	if !current.ClearThresholds.Equals(&cfg.ClearThresholds) {
		if err := dev.setThresholds(false, &cfg.ClearThresholds); err != nil {
			return err
		}
	}
	return nil
}

// The hdc302x sensors have a built in heater element for operating in environments
// where the humidity/temperature level is condensing. SetHeater allows you to turn
// the heater element on and off at specified power levels.  Refer to the datasheet
// for instructions on how the heater can be used in those environments.
func (dev *Dev) SetHeater(powerLevel HeaterPower) error {
	if powerLevel > PowerFull {
		return fmt.Errorf("hdc302x: invalid value for powerLevel: 0x%x", powerLevel)
	}
	if powerLevel == PowerOff {
		return dev.d.Tx(disableHeater, nil)
	}
	var setValue = []byte{readSetHeater[0],
		readSetHeater[1],
		byte((powerLevel >> 8) & 0xff),
		byte(powerLevel & 0xff),
		0}
	setValue[4] = crc8(setValue[2:4])
	err := dev.d.Tx(setValue, nil)
	if err != nil {
		return err
	}
	return dev.d.Tx(enableHeater, nil)
}

func (dev *Dev) String() string {
	return fmt.Sprintf("hdc302x: %s", dev.d.String())
}

func (cfg *Configuration) String() string {
	return fmt.Sprintf(`{
		SerialNumber: 0x%x, 
		VendorID: 0x%x,
		Status: 0x%x,
		SampleRate: %d,
		HumidityOffset: %s,
		TemperatureOffset: %s,
		AlertThresholds: %s,
		ClearThresholds: %s
		}`,
		cfg.SerialNumber,
		cfg.VendorID,
		cfg.Status,
		cfg.SampleRate,
		cfg.HumidityOffset,
		cfg.TemperatureOffset+physic.ZeroCelsius,
		&cfg.AlertThresholds,
		&cfg.ClearThresholds)
}

func (t *Threshold) String() string {
	return fmt.Sprintf("{ Humidity: %s, Temperature: %s }", t.Humidity, t.Temperature+physic.ZeroCelsius)
}

func (tp *ThresholdPair) String() string {
	return fmt.Sprintf("{ Low: %s, High: %s }",
		&tp.Low,
		&tp.High)
}

func (t *Threshold) Equals(tCompare *Threshold) bool {
	return t.Temperature == tCompare.Temperature && t.Humidity == tCompare.Humidity
}

// For thresholds, you can only set a truncated value. For temperature, that means
// the 9 high bits, and for humidity, the 7 high bits. This means a comparison of
// a written value with the resulting value can be off. This method encapsulates
// the comparison of a threshold pair to make sure they're approximately equal.
func (t *Threshold) ApproximatelyEquals(tCompare *Threshold) bool {
	t1 := temperatureToFloat64(t.Temperature)
	h1 := humidityToFloat64(t.Humidity)
	t2 := temperatureToFloat64(tCompare.Temperature)
	h2 := humidityToFloat64(tCompare.Humidity)
	tLimit := float64(uint16(1<<8)) * temperatureScalar / scaleDivisor
	hLimit := float64(uint16(1<<9)) * humidityScalar / scaleDivisor
	return math.Abs(t1-t2) < tLimit &&
		math.Abs(h1-h2) < hLimit
}

func (tp *ThresholdPair) Equals(tpCompare *ThresholdPair) bool {
	return tp.Low.Equals(&tpCompare.Low) && tp.High.Equals(&tpCompare.High)
}

var _ conn.Resource = &Dev{}
var _ physic.SenseEnv = &Dev{}
