// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.
//
// Unit tests for the package. Note that this supports running on a live
// sensor, or using playback mode to simulate a live device.
//
// To use a live device, define the environment variable SCD4X and run go test.

package scd4x

import (
	"fmt"
	"os"
	"testing"
	"time"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/host/v3"
)

var bus i2c.Bus
var liveDevice bool = false

// playback values for TestSense
var sensePlayback = []i2ctest.IO{
	{Addr: SensorAddress, W: []uint8{0x36, 0xf6}},
	{Addr: SensorAddress, W: []uint8{0x21, 0xb1}},
	{Addr: SensorAddress, W: []uint8{0xe4, 0xb8}, R: []uint8{0x80, 0x0, 0xa2}},
	{Addr: SensorAddress, W: []uint8{0xe4, 0xb8}, R: []uint8{0x80, 0x0, 0xa2}},
	{Addr: SensorAddress, W: []uint8{0xe4, 0xb8}, R: []uint8{0x80, 0x6, 0x4}},
	{Addr: SensorAddress, W: []uint8{0xec, 0x5}, R: []uint8{0x2, 0x2c, 0xa3, 0x67, 0xd, 0x36, 0x4d, 0x8, 0xf1}}}

var senseContinuousPlayback = []i2ctest.IO{
	{Addr: SensorAddress, W: []uint8{0x36, 0xf6}},
	{Addr: SensorAddress, W: []uint8{0x21, 0xb1}},
	{Addr: SensorAddress, W: []uint8{0xe4, 0xb8}, R: []uint8{0x80, 0x6, 0x4}},
	{Addr: SensorAddress, W: []uint8{0xec, 0x5}, R: []uint8{0x2, 0x1f, 0x35, 0x65, 0x82, 0xbb, 0x53, 0x5e, 0x2a}},
	{Addr: SensorAddress, W: []uint8{0xe4, 0xb8}, R: []uint8{0x80, 0x6, 0x4}},
	{Addr: SensorAddress, W: []uint8{0xec, 0x5}, R: []uint8{0x2, 0x22, 0xbc, 0x65, 0x39, 0xee, 0x55, 0x4b, 0xc6}},
	{Addr: SensorAddress, W: []uint8{0xe4, 0xb8}, R: []uint8{0x80, 0x6, 0x4}},
	{Addr: SensorAddress, W: []uint8{0xec, 0x5}, R: []uint8{0x2, 0x1c, 0x66, 0x64, 0xeb, 0x7c, 0x56, 0xd1, 0x9}},
	{Addr: SensorAddress, W: []uint8{0xe4, 0xb8}, R: []uint8{0x80, 0x6, 0x4}},
	{Addr: SensorAddress, W: []uint8{0xec, 0x5}, R: []uint8{0x2, 0x1f, 0x35, 0x64, 0xad, 0xe7, 0x58, 0x2f, 0xf9}},
	{Addr: SensorAddress, W: []uint8{0xe4, 0xb8}, R: []uint8{0x80, 0x6, 0x4}},
	{Addr: SensorAddress, W: []uint8{0xec, 0x5}, R: []uint8{0x2, 0x1f, 0x35, 0x64, 0x79, 0x27, 0x59, 0x71, 0x6c}},
	{Addr: SensorAddress, W: []uint8{0xe4, 0xb8}, R: []uint8{0x80, 0x6, 0x4}},
	{Addr: SensorAddress, W: []uint8{0xec, 0x5}, R: []uint8{0x2, 0x1f, 0x35, 0x64, 0x46, 0xcc, 0x5a, 0x8d, 0xbe}}}

var getSetTestPlayback = []i2ctest.IO{
	{Addr: SensorAddress, W: []uint8{0x36, 0xf6}},
	{Addr: SensorAddress, W: []uint8{0x21, 0xb1}},
	{Addr: SensorAddress, W: []uint8{0x3f, 0x86}},
	{Addr: SensorAddress, W: []uint8{0x36, 0x46}},
	{Addr: SensorAddress, W: []uint8{0xe0, 0x0}, R: []uint8{0x0, 0x5, 0x74}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x13}, R: []uint8{0x0, 0x1, 0xb0}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x40}, R: []uint8{0x0, 0x2c, 0x7a}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x4b}, R: []uint8{0x0, 0x9c, 0xc5}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x3f}, R: []uint8{0x1, 0x90, 0x4c}},
	{Addr: SensorAddress, W: []uint8{0x36, 0x82}, R: []uint8{0x73, 0xb1, 0x19, 0xeb, 0x7, 0x7a, 0x3b, 0xc, 0x54}},
	{Addr: SensorAddress, W: []uint8{0x20, 0x2f}, R: []uint8{0x4, 0x41, 0xe}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x22}, R: []uint8{0x0, 0x0, 0x81}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x18}, R: []uint8{0x5, 0xda, 0x29}},
	{Addr: SensorAddress, W: []uint8{0xe0, 0x0}, R: []uint8{0x0, 0x5, 0x74}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x13}, R: []uint8{0x0, 0x1, 0xb0}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x40}, R: []uint8{0x0, 0x2c, 0x7a}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x4b}, R: []uint8{0x0, 0x9c, 0xc5}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x3f}, R: []uint8{0x1, 0x90, 0x4c}},
	{Addr: SensorAddress, W: []uint8{0x36, 0x82}, R: []uint8{0x73, 0xb1, 0x19, 0xeb, 0x7, 0x7a, 0x3b, 0xc, 0x54}},
	{Addr: SensorAddress, W: []uint8{0x20, 0x2f}, R: []uint8{0x4, 0x41, 0xe}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x22}, R: []uint8{0x0, 0x0, 0x81}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x18}, R: []uint8{0x5, 0xda, 0x29}},
	{Addr: SensorAddress, W: []uint8{0xe0, 0x0, 0x0, 0xa, 0x5a}},
	{Addr: SensorAddress, W: []uint8{0x24, 0x16, 0x0, 0x0, 0x81}},
	{Addr: SensorAddress, W: []uint8{0x24, 0x45, 0x0, 0x30, 0x44}},
	{Addr: SensorAddress, W: []uint8{0x24, 0x4e, 0x0, 0xa0, 0x7d}},
	{Addr: SensorAddress, W: []uint8{0x24, 0x3a, 0x1, 0xa4, 0x4d}},
	{Addr: SensorAddress, W: []uint8{0x24, 0x27, 0x6, 0x44, 0x22}},
	{Addr: SensorAddress, W: []uint8{0xe0, 0x0}, R: []uint8{0x0, 0xa, 0x5a}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x13}, R: []uint8{0x0, 0x0, 0x81}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x40}, R: []uint8{0x0, 0x30, 0x44}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x4b}, R: []uint8{0x0, 0xa0, 0x7d}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x3f}, R: []uint8{0x1, 0xa4, 0x4d}},
	{Addr: SensorAddress, W: []uint8{0x36, 0x82}, R: []uint8{0x73, 0xb1, 0x19, 0xeb, 0x7, 0x7a, 0x3b, 0xc, 0x54}},
	{Addr: SensorAddress, W: []uint8{0x20, 0x2f}, R: []uint8{0x4, 0x41, 0xe}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x22}, R: []uint8{0x6, 0x44, 0x22}},
	{Addr: SensorAddress, W: []uint8{0x23, 0x18}, R: []uint8{0x5, 0xda, 0x29}},
	{Addr: SensorAddress, W: []uint8{0x36, 0x46}}}

var basicStartup = []i2ctest.IO{
	{Addr: SensorAddress, W: []uint8{0x36, 0xf6}},
	{Addr: SensorAddress, W: []uint8{0x21, 0xb1}}}

func init() {
	var err error
	// If the environment variable is set, assume we have a live device on
	// the default i2c bus and use it for testing. If the variable is not
	// present, then use the playback/read values.
	if os.Getenv("SCD4X") != "" {
		liveDevice = true
	}
	if _, err = host.Init(); err != nil {
		fmt.Println(err)
	}

	if liveDevice {
		bus, err = i2creg.Open("")
		if err != nil {
			fmt.Println(err)
		}
		// Add the recorder to dump the data stream when we're using a live device.
		bus = &i2ctest.Record{Bus: bus}
	} else {
		bus = &i2ctest.Playback{DontPanic: true}
	}

}

// getDev returns an scd4x device for testing connected to either a live
// bus, or a playback bus. playbackOps is a slice of i2ctest.IO
// operations to be used for playback mode. Ignored for live device
// testing.
func getDev(t *testing.T, playbackOps ...[]i2ctest.IO) (*Dev, error) {
	if liveDevice {
		if recorder, ok := bus.(*i2ctest.Record); ok {
			// Clear the operations buffer.
			recorder.Ops = make([]i2ctest.IO, 0, 32)
		}
	} else {
		if len(playbackOps) == 1 {
			pb := bus.(*i2ctest.Playback)
			pb.Ops = playbackOps[0]
			pb.Count = 0
		}
	}
	dev, err := NewI2C(bus, SensorAddress)

	if err != nil {
		t.Fatal(err)
	}

	return dev, err
}

// shutdown dumps the recorder values if we we're running a live device.
func shutdown(t *testing.T) {
	if recorder, ok := bus.(*i2ctest.Record); ok {
		t.Logf("%#v", recorder.Ops)
	}
}

func TestCRC(t *testing.T) {
	tests := []struct {
		bytes []byte
		crc   byte
	}{
		{bytes: []byte{0xbe, 0xef}, crc: 0x92},
		{bytes: []byte{0x01, 0xa4}, crc: 0x4d},
	}
	for _, test := range tests {
		res := calcCRC(test.bytes)
		if res != test.crc {
			t.Error(fmt.Errorf("crc calculation error bytes: %#v, result: 0x%x expected: 0x%x", test.bytes, res, test.crc))
		}
	}
}

func TestCountToTemperature(t *testing.T) {
	tests := []struct {
		count    uint16
		expected physic.Temperature
	}{
		{count: 0x6667, expected: physic.ZeroCelsius + 25*physic.Celsius},
	}
	for _, test := range tests {
		result := countToTemp(test.count)
		// round to 2 sig figs for the floating point comparison.
		result -= result % (10 * physic.MilliKelvin)
		if result != test.expected {
			t.Errorf("received: %.8f expected %.8f", result.Celsius(), test.expected.Celsius())
		}
	}
}

func TestCountToHumidity(t *testing.T) {
	result := countToHumidity(0x5eb9) // from the datasheet
	// Truncate to 2 decimals for comparison.
	result -= result % physic.MilliRH
	expected := physic.RelativeHumidity(37 * physic.PercentRH)
	if result != expected {
		t.Errorf("unexpected value: %d expected %d", result, expected)
	}
}

// Non-device basic functionality.
func TestBasic(t *testing.T) {
	dev, err := getDev(t, basicStartup)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dev.Halt() }()
	defer shutdown(t)

	env := Env{}
	dev.Precision(&env)
	t.Logf("scd4x.Precision()=%#v\n", env)
	if env.CO2 != 1 || env.Humidity != physic.TenthMicroRH || env.Temperature != (15259*physic.NanoKelvin) {
		t.Error(fmt.Errorf("incorrect value for Precision(): %#v", env))
	}

	s := dev.String()
	t.Logf("dev.String()=%s", s)
	if len(s) == 0 {
		t.Error("Dev.String() returned empty value.")
	}
}

func TestSense(t *testing.T) {
	dev, err := getDev(t, sensePlayback)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dev.Halt() }()
	defer shutdown(t)
	env := Env{}
	err = dev.Sense(&env)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(env.String())
	}
}

func TestSenseContinuous(t *testing.T) {
	readings := 6
	timeBase := time.Second
	if liveDevice {
		timeBase *= 10
	}
	dev, err := getDev(t, senseContinuousPlayback)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dev.Halt() }()
	defer shutdown(t)
	t.Log("dev.sensing=", dev.sensing)
	ch, err := dev.SenseContinuous(timeBase)
	if err != nil {
		t.Error(err)
	}

	go func() {
		time.Sleep(time.Duration(readings) * timeBase)
		_ = dev.Halt()
	}()
	received := 0
	for env := range ch {
		t.Log(env.String())
		received += 1
	}
	if received < (readings-1) || received > readings {
		t.Errorf("SenseContinuous() expected at least %d readings, got %d", readings-1, received)
	}

}

func TestGetSetConfiguration(t *testing.T) {
	dev, err := getDev(t, getSetTestPlayback)
	if err != nil {
		t.Fatal(err)
	}
	err = dev.Halt()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	// Baseline our settings
	err = dev.Reset(ResetEEPROM)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	defer shutdown(t)
	cfg, err := dev.GetConfiguration()

	if err != nil {
		t.Error(err)
	}
	t.Logf("existing configuration: %#v", cfg)
	cfg.AmbientPressure += 500 * physic.Pascal
	cfg.ASCEnabled = !cfg.ASCEnabled
	cfg.ASCInitialPeriod += 4 * time.Hour
	cfg.ASCStandardPeriod += 4 * time.Hour
	cfg.ASCTarget += 20
	cfg.SensorAltitude = 1604 * physic.Metre

	err = dev.SetConfiguration(cfg)
	if err != nil {
		t.Error(err)
	}
	read, err := dev.GetConfiguration()
	if err != nil {
		t.Error(err)
	}
	t.Logf("new configuration: %#v", read)

	if read.AmbientPressure != cfg.AmbientPressure {
		t.Errorf("scd4x: error setting ambient pressure. found: %s (%d) expected: %s (%d)", read.AmbientPressure.String(), read.AmbientPressure, cfg.AmbientPressure.String(), cfg.AmbientPressure)
	}
	if read.ASCEnabled != cfg.ASCEnabled {
		t.Errorf("scd4x: error setting asc enabled. Found %t expected %t", read.ASCEnabled, cfg.ASCEnabled)
	}
	if read.ASCInitialPeriod != cfg.ASCInitialPeriod {
		t.Errorf("scd4x: error setting initial period. found: %d expected %d", read.ASCInitialPeriod, cfg.ASCInitialPeriod)
	}
	if read.ASCStandardPeriod != cfg.ASCStandardPeriod {
		t.Errorf("scd4x: error setting standard period. found: %d expected %d", read.ASCStandardPeriod, cfg.ASCStandardPeriod)
	}
	if read.ASCTarget != cfg.ASCTarget {
		t.Errorf("scd4x: error setting asc target. found %d expected %d", read.ASCTarget, cfg.ASCTarget)
	}
	if read.SensorAltitude != cfg.SensorAltitude {
		t.Errorf("scd4x: error setting sensor altitude. found %d expected %d", read.SensorAltitude/physic.Metre, cfg.SensorAltitude/physic.Metre)
	}

	_ = dev.Reset(ResetEEPROM) // and go back to our known state.
}

// Since there are limited read/write cycles, by default DO NOT test persist
// and reset factory. To perform the tests, define the environment variable
// SCDRESET. Running this test will destructively clear customized values
// previously programmed into the device.
func TestPersistAndResetFactory(t *testing.T) {
	if !liveDevice || os.Getenv("SCDRESET") == "" {
		t.Skip("using live device and SCDRESET not defined. skipping")
	}
	dev, err := getDev(t)
	if err != nil {
		t.Fatal(err)
	}
	err = dev.Halt()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	// Read the current running configuration.
	cfg, err := dev.GetConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown(t)

	// Set the altitude to the current Altitude+1000M and write it to the device.
	if cfg.SensorAltitude < (2000 * physic.Metre) {
		cfg.SensorAltitude += 1000 * physic.Metre
	} else {
		cfg.SensorAltitude -= (500 * physic.Metre)
	}
	t.Logf("updating sensor altitude to %s", cfg.SensorAltitude)

	err = dev.SetConfiguration(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Now, re-read the configuration to verify the write worked.
	updatedCfg, err := dev.GetConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if updatedCfg.SensorAltitude != cfg.SensorAltitude {
		t.Fatalf("scd41x: Change sensor altitude failed. Read: %s Expected: %s", updatedCfg.SensorAltitude.String(), cfg.SensorAltitude)
	}

	// OK, now Persist()
	err = dev.Persist()
	if err != nil {
		t.Error(err)
	}
	_ = dev.Reset(ResetEEPROM)
	time.Sleep(time.Second)

	// OK, now write 0
	cfg.SensorAltitude = 0
	err = dev.SetConfiguration(cfg)
	if err != nil {
		t.Error(err)
	}
	// Reset Settings to EEPROM
	err = dev.Reset(ResetEEPROM)
	if err != nil {
		t.Fatal(err)
	}

	// Sometimes you have to wait for it to come to the party...
	for range 5 {
		_ = dev.Halt()
		// Now, re-read the configuration
		cfg, err = dev.GetConfiguration()
		if err != nil {
			t.Logf("GetConfiguration Failed: %s Sleeping before retry.", err)
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	if err != nil {
		t.Fatal(err)
	}

	// The expected value is the original value +1000M
	if cfg.SensorAltitude != updatedCfg.SensorAltitude {
		t.Errorf("Error using reset to eeprom. Expected SensorAltitude: %s Found: %s", updatedCfg.SensorAltitude, cfg.SensorAltitude)
	}

	t.Logf("current configuration: %#v", cfg)
	// Almost there. Now, reset to factory and read sensor-altitude.
	t.Logf("calling reset factory")
	err = dev.Reset(ResetFactory)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second)

	cfg, err = dev.GetConfiguration()

	if err != nil {
		t.Error(err)
	}
	t.Logf("Reset to factory configuration is now: %#v", cfg)

	if cfg.SensorAltitude != 0 {
		t.Errorf("Error resetting to factory. Sensor Altitude: %s expected 0m", cfg.SensorAltitude)
	}
}
