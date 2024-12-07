// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package hdc302x

import (
	"fmt"
	"math"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/host/v3"
)

var bus i2c.Bus
var liveDevice bool

// Playback values for a single sense operation.
var pbSense = []i2ctest.IO{
	{Addr: DefaultSensorAddress, W: []uint8{0x23, 0x34}},
	// 26.621 C, 23.2%RH
	{Addr: DefaultSensorAddress, W: []uint8{0xe0, 0x0}, R: []uint8{0x68, 0xc5, 0x51, 0x3b, 0x82, 0x31}},
}

// Playback for heater testing.
var pbHeater = []i2ctest.IO{
	{Addr: DefaultSensorAddress, W: []uint8{0x23, 0x34}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe0, 0x0}, R: []uint8{0x64, 0x93, 0x3d, 0x45, 0x3a, 0x61}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x6e, 0x3f, 0xff, 0x6}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x6d}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe0, 0x0}, R: []uint8{0x9d, 0xb1, 0x2, 0x9, 0xc6, 0xa3}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x66}}}

// Playback for modifying configuration.
var pbConfiguration = []i2ctest.IO{
	{Addr: DefaultSensorAddress, W: []uint8{0x23, 0x34}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x83}, R: []uint8{0xc2, 0x95, 0x3e}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x84}, R: []uint8{0xb1, 0x49, 0x51}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x85}, R: []uint8{0x15, 0x21, 0x2f}},
	{Addr: DefaultSensorAddress, W: []uint8{0xa0, 0x4}, R: []uint8{0x80, 0x80, 0xd8}},
	{Addr: DefaultSensorAddress, W: []uint8{0x37, 0x81}, R: []uint8{0x30, 0x0, 0x33}},
	{Addr: DefaultSensorAddress, W: []uint8{0xf3, 0x2d}, R: []uint8{0x0, 0x0, 0x81}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x41}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x2}, R: []uint8{0x34, 0x66, 0xad}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x1f}, R: []uint8{0xcd, 0x33, 0xfd}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x9}, R: []uint8{0x38, 0x69, 0x37}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x14}, R: []uint8{0xc9, 0x2d, 0x22}}}

// playback for offset modification
var pbOffsets = []i2ctest.IO{
	{Addr: DefaultSensorAddress, W: []uint8{0x23, 0x34}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe0, 0x0}, R: []uint8{0x70, 0x90, 0x83, 0x42, 0x10, 0x92}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x83}, R: []uint8{0xc2, 0x95, 0x3e}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x84}, R: []uint8{0xb1, 0x49, 0x51}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x85}, R: []uint8{0x15, 0x21, 0x2f}},
	{Addr: DefaultSensorAddress, W: []uint8{0xa0, 0x4}, R: []uint8{0x19, 0xba, 0x48}},
	{Addr: DefaultSensorAddress, W: []uint8{0x37, 0x81}, R: []uint8{0x30, 0x0, 0x33}},
	{Addr: DefaultSensorAddress, W: []uint8{0xf3, 0x2d}, R: []uint8{0x0, 0x0, 0x81}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x41}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x2}, R: []uint8{0x34, 0x66, 0xad}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x1f}, R: []uint8{0xcd, 0x33, 0xfd}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x9}, R: []uint8{0x38, 0x69, 0x37}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x14}, R: []uint8{0xc9, 0x2d, 0x22}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x93}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x83}, R: []uint8{0xc2, 0x95, 0x3e}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x84}, R: []uint8{0xb1, 0x49, 0x51}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x85}, R: []uint8{0x15, 0x21, 0x2f}},
	{Addr: DefaultSensorAddress, W: []uint8{0xa0, 0x4}, R: []uint8{0x19, 0xba, 0x48}},
	{Addr: DefaultSensorAddress, W: []uint8{0x37, 0x81}, R: []uint8{0x30, 0x0, 0x33}},
	{Addr: DefaultSensorAddress, W: []uint8{0xf3, 0x2d}, R: []uint8{0x0, 0x0, 0x81}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x41}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x2}, R: []uint8{0x34, 0x66, 0xad}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x1f}, R: []uint8{0xcd, 0x33, 0xfd}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x9}, R: []uint8{0x38, 0x69, 0x37}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x14}, R: []uint8{0xc9, 0x2d, 0x22}},
	{Addr: DefaultSensorAddress, W: []uint8{0xa0, 0x4, 0x32, 0xf4, 0xac}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x83}, R: []uint8{0xc2, 0x95, 0x3e}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x84}, R: []uint8{0xb1, 0x49, 0x51}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x85}, R: []uint8{0x15, 0x21, 0x2f}},
	{Addr: DefaultSensorAddress, W: []uint8{0xa0, 0x4}, R: []uint8{0x32, 0xf4, 0xac}},
	{Addr: DefaultSensorAddress, W: []uint8{0x37, 0x81}, R: []uint8{0x30, 0x0, 0x33}},
	{Addr: DefaultSensorAddress, W: []uint8{0xf3, 0x2d}, R: []uint8{0x0, 0x0, 0x81}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x41}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x2}, R: []uint8{0x34, 0x66, 0xad}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x1f}, R: []uint8{0xcd, 0x33, 0xfd}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x9}, R: []uint8{0x38, 0x69, 0x37}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x14}, R: []uint8{0xc9, 0x2d, 0x22}},
	{Addr: DefaultSensorAddress, W: []uint8{0x23, 0x34}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe0, 0x0}, R: []uint8{0x7f, 0x28, 0x1c, 0x37, 0x8d, 0xab}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0xa2}}}

var pbAlerts = []i2ctest.IO{
	{Addr: DefaultSensorAddress, W: []uint8{0x23, 0x34}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe0, 0x0}, R: []uint8{0x70, 0xe0, 0x7b, 0x3f, 0xf7, 0xbf}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x83}, R: []uint8{0xc2, 0x95, 0x3e}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x84}, R: []uint8{0xb1, 0x49, 0x51}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x85}, R: []uint8{0x15, 0x21, 0x2f}},
	{Addr: DefaultSensorAddress, W: []uint8{0xa0, 0x4}, R: []uint8{0x19, 0xba, 0x48}},
	{Addr: DefaultSensorAddress, W: []uint8{0x37, 0x81}, R: []uint8{0x30, 0x0, 0x33}},
	{Addr: DefaultSensorAddress, W: []uint8{0xf3, 0x2d}, R: []uint8{0x80, 0x10, 0xe1}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x41}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x2}, R: []uint8{0x34, 0x66, 0xad}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x1f}, R: []uint8{0xcd, 0x33, 0xfd}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x9}, R: []uint8{0x38, 0x69, 0x37}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x14}, R: []uint8{0xc9, 0x2d, 0x22}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x93}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x83}, R: []uint8{0xc2, 0x95, 0x3e}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x84}, R: []uint8{0xb1, 0x49, 0x51}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x85}, R: []uint8{0x15, 0x21, 0x2f}},
	{Addr: DefaultSensorAddress, W: []uint8{0xa0, 0x4}, R: []uint8{0x19, 0xba, 0x48}},
	{Addr: DefaultSensorAddress, W: []uint8{0x37, 0x81}, R: []uint8{0x30, 0x0, 0x33}},
	{Addr: DefaultSensorAddress, W: []uint8{0xf3, 0x2d}, R: []uint8{0x0, 0x0, 0x81}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x41}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x2}, R: []uint8{0x34, 0x66, 0xad}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x1f}, R: []uint8{0xcd, 0x33, 0xfd}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x9}, R: []uint8{0x38, 0x69, 0x37}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x14}, R: []uint8{0xc9, 0x2d, 0x22}},
	{Addr: DefaultSensorAddress, W: []uint8{0xa0, 0x4, 0x80, 0x80, 0xd8}},
	{Addr: DefaultSensorAddress, W: []uint8{0x61, 0x0, 0x4c, 0x1d, 0xb3}},
	{Addr: DefaultSensorAddress, W: []uint8{0x61, 0x1d, 0xbe, 0xdb, 0x93}},
	{Addr: DefaultSensorAddress, W: []uint8{0x61, 0xb, 0x58, 0x2b, 0x3d}},
	{Addr: DefaultSensorAddress, W: []uint8{0x61, 0x16, 0xb2, 0xcc, 0xf3}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x83}, R: []uint8{0xc2, 0x95, 0x3e}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x84}, R: []uint8{0xb1, 0x49, 0x51}},
	{Addr: DefaultSensorAddress, W: []uint8{0x36, 0x85}, R: []uint8{0x15, 0x21, 0x2f}},
	{Addr: DefaultSensorAddress, W: []uint8{0xa0, 0x4}, R: []uint8{0x80, 0x80, 0xd8}},
	{Addr: DefaultSensorAddress, W: []uint8{0x37, 0x81}, R: []uint8{0x30, 0x0, 0x33}},
	{Addr: DefaultSensorAddress, W: []uint8{0xf3, 0x2d}, R: []uint8{0x0, 0x0, 0x81}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x41}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x2}, R: []uint8{0x4c, 0x1d, 0xb3}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x1f}, R: []uint8{0xbe, 0xdb, 0x93}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x9}, R: []uint8{0x58, 0x2b, 0x3d}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe1, 0x14}, R: []uint8{0xb2, 0xcc, 0xf3}},
	{Addr: DefaultSensorAddress, W: []uint8{0x23, 0x34}},
	{Addr: DefaultSensorAddress, W: []uint8{0xe0, 0x0}, R: []uint8{0x62, 0x77, 0x62, 0x4a, 0x94, 0x1b}},
	{Addr: DefaultSensorAddress, W: []uint8{0xf3, 0x2d}, R: []uint8{0x89, 0x0, 0x61}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x41}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0x93}},
	{Addr: DefaultSensorAddress, W: []uint8{0x30, 0xa2}}}

func init() {
	var err error

	liveDevice = os.Getenv("HDC302X") != ""

	// Make sure periph is initialized.
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

// getDev returns a configured device using either an i2c bus, or a playback bus.
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
	dev, err := NewI2C(bus, DefaultSensorAddress, RateFourHertz)

	if err != nil {
		t.Log("error constructing dev")
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
	var tests = []struct {
		bytes  []byte
		result byte
	}{
		{bytes: []byte{0xbe, 0xef}, result: 0x92},
		{bytes: []byte{0xab, 0xcd}, result: 0x6f},
	}
	for _, test := range tests {
		res := crc8(test.bytes)
		if res != test.result {
			t.Errorf("crc8(%#v)!=0x%d receieved 0x%d", test.bytes, test.result, res)
		}
	}
}

// TestConversions tests the various temperature/humidity functions
// for correct operation.
func TestConversions(t *testing.T) {
	envPrecision := physic.Env{}
	dev := Dev{}
	dev.Precision(&envPrecision)
	t.Logf("Precision: Temperature: %d nanoKelvin Humidity: %d TenthMicroPercent RH", envPrecision.Temperature, envPrecision.Humidity)

	temp := countToTemperature([]byte{0x00, 0x00})
	expected := physic.ZeroCelsius - 45*physic.Celsius
	if temp != expected {
		t.Errorf("unexpected countToTemperature. expected %s, received %s %d", expected, temp, temp)
	}
	temp = countToTemperature([]byte{0xd4, 0x1c})
	expected = physic.ZeroCelsius + 100*physic.Celsius
	if math.Abs(float64(expected-temp)) > float64(envPrecision.Temperature) {
		t.Errorf("unexpected countToTemperature. expected %s (%d), received %s (%d)", expected, expected, temp, temp)
	}

	var hTests = []struct {
		bytes  []byte
		result physic.RelativeHumidity
	}{
		{bytes: []byte{0x0, 0x0}, result: physic.RelativeHumidity(0)},
		{bytes: []byte{0x80, 0x0}, result: 50 * physic.PercentRH},
		{bytes: []byte{0xff, 0xff}, result: 100 * physic.PercentRH},
	}
	for _, hTest := range hTests {
		humidity := countToHumidity(hTest.bytes)
		if (humidity - hTest.result) > envPrecision.Humidity {
			t.Errorf("unexpected humidity got %s (%d) expected %s (%d)", humidity, humidity, hTest.result, hTest.result)
		}
	}

}

// Tests the conversions of offsets to binary offset values used by the device.
func TestOffsetConversions(t *testing.T) {
	// These are a subtly off from the datasheet because it's using floating point representation,
	var tempOffsets = []struct {
		temperature    physic.Temperature
		expectedResult byte
	}{
		{temperature: 170_902 * physic.MicroKelvin, expectedResult: 0x81},
		{temperature: -170_902 * physic.MicroKelvin, expectedResult: 0x01}, // Test sign handling
		{temperature: 7_178_000 * physic.MicroKelvin, expectedResult: 0xaa},
		{temperature: 10_937_700 * physic.MicroKelvin, expectedResult: 0xc0},
		{temperature: 21_704_500 * physic.MicroKelvin, expectedResult: 0xff},
	}
	for _, test := range tempOffsets {
		res := computeTemperatureOffsetByte(test.temperature)
		if res != test.expectedResult {
			t.Errorf("computeTemperatureOffsetByte() Offset: %s Expected Value: 0x%x Received: 0x%x", test.temperature, test.expectedResult, res)
		}
	}

	// Now repeat for humidity
	var humidityOffsets = []struct {
		humidity       physic.RelativeHumidity
		expectedResult byte
	}{
		{humidity: 19532 * physic.TenthMicroRH, expectedResult: 0x81},
		{humidity: -19532 * physic.TenthMicroRH, expectedResult: 0x01},
		{humidity: 820326 * physic.TenthMicroRH, expectedResult: 0xaa},
		{humidity: -24_805_1 * physic.MicroRH, expectedResult: 0x7f},
	}
	for _, test := range humidityOffsets {
		res := computeHumidityOffsetByte(test.humidity)
		if res != test.expectedResult {
			t.Errorf("computeHumidityOffsetByte() Offset: %s Expected Value: 0x%x Received: 0x%x", test.humidity, test.expectedResult, res)
		}
	}

}

func TestBasic(t *testing.T) {
	dev, err := getDev(t, []i2ctest.IO{pbSense[0]})
	if err != nil {
		t.Fatal(err)
	}
	env := &physic.Env{}
	dev.Precision(env)
	if env.Pressure != 0 {
		t.Error("this device doesn't measure pressure")
	}
	if env.Temperature != (2670329 * physic.NanoKelvin) {
		t.Errorf("incorrect temperature precision value got %d expected %d", env.Temperature, 2670329*physic.NanoKelvin)
	}
	if env.Humidity != 153*physic.TenthMicroRH {
		t.Errorf("incorrect humidity precision got %d expected %d", env.Humidity, 153*physic.TenthMicroRH)
	}

	s := dev.String()
	if len(s) == 0 {
		t.Error("invalid value for String()")
	}
}

func TestSense(t *testing.T) {
	d, err := getDev(t, pbSense)

	if err != nil {
		t.Fatalf("failed to initialize hdc302x: %v", err)
	}
	defer shutdown(t)

	// Read temperature and humidity from the sensor
	e := physic.Env{}
	if err := d.Sense(&e); err != nil {
		t.Error(err)
	}
	t.Logf("%8s %9s", e.Temperature, e.Humidity)

	if !liveDevice {
		// The playback temp is 26.621C Ensure that's what we got.
		expected := physic.Temperature(299770889600)
		if e.Temperature != expected {
			t.Errorf("incorrect temperature value read. Expected: %s (%d) Found: %s (%d)",
				expected.String(),
				expected,
				e.Temperature.String(),
				e.Temperature,
			)
		}

		// 23.2% expected.
		expectedRH := 2324559 * physic.TenthMicroRH
		if e.Humidity != expectedRH {
			t.Errorf("incorrect humidity value read. Expected: %s (%d) Found: %s (%d)",
				expectedRH.String(),
				expectedRH,
				e.Humidity.String(),
				e.Humidity,
			)
		}
	}

}

func TestSenseContinuous(t *testing.T) {

	readCount := int32(10)

	// make 10 copies of the single reading playback data.
	pb := make([]i2ctest.IO, 0, readCount+1)
	pb = append(pb, pbSense[0])
	for range readCount {
		pb = append(pb, pbSense[1])
	}
	// Add in the halt
	pb = append(pb, i2ctest.IO{Addr: DefaultSensorAddress,
		W: []uint8{stopContinuousReadings[0], stopContinuousReadings[1]}})

	dev, err := getDev(t, pb)
	if err != nil {
		t.Error(fmt.Errorf("failed to initialize hd302x: %w", err))
	}
	defer shutdown(t)

	_, err = dev.SenseContinuous(100 * time.Millisecond)
	if err == nil {
		t.Error("expected error for sense continuous interval < sample interval")
	}

	ch, err := dev.SenseContinuous(time.Second)
	if err != nil {
		t.Fatal(err)
	}

	_, err = dev.SenseContinuous(time.Second)
	if err == nil {
		t.Error("expected an error for attempting concurrent SenseContinuous")
	}

	counter := atomic.Int32{}
	tEnd := time.Now().UnixMilli() + int64(readCount+2)*1000
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			// Stay here until we get the expected number of reads, or the time
			// has expired.
			if counter.Load() == readCount || time.Now().UnixMilli() > tEnd {
				err := dev.Halt()
				if err != nil {
					t.Error(err)
				}
				return
			}
		}
	}()

	for e := range ch {
		counter.Add(1)
		t.Log(time.Now(), e)
	}
	if counter.Load() != readCount {
		t.Errorf("expected %d readings. received %d", readCount, counter.Load())
	}
}

func TestConfiguration(t *testing.T) {
	dev, err := getDev(t, pbConfiguration)
	if err != nil {
		t.Fatalf("failed to initialize hd302x: %v", err)
	}
	defer shutdown(t)

	cfg, err := dev.Configuration()
	if err != nil {
		t.Error(err)
	}
	s := cfg.String()
	t.Log("configuration: ", s)
	if len(s) == 0 {
		t.Errorf("invalid Configuration.String()")
	}
	if cfg.SerialNumber == 0 {
		t.Error("invalid serial number")
	}
	if cfg.VendorID != 0x3000 {
		t.Errorf("invalid manufacturer id 0x%x", cfg.VendorID)
	}
}

// Tests applying offsets for temperature and humidity and checking that
// the values are applied during a subsequent read.
func TestOffsetModification(t *testing.T) {
	dev, err := getDev(t, pbOffsets)
	if err != nil {
		t.Fatalf("failed to initialize hd302x: %v", err)
	}
	defer shutdown(t)
	env := physic.Env{}
	err = dev.Sense(&env)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial Readings: %s", env)
	cfg, err := dev.Configuration()
	if err != nil {
		t.Fatal(err)
	}
	cfg.TemperatureOffset += 10 * physic.Celsius
	cfg.HumidityOffset -= 5 * physic.PercentRH
	t.Log("writing configuration: ", cfg)
	err = dev.SetConfiguration(cfg)

	if err != nil {
		t.Fatal(err)
	}
	cfg2, _ := dev.Configuration()
	t.Logf("read configuration=%s", cfg2)
	env2 := physic.Env{}
	err = dev.Sense(&env2)
	if err != nil {
		t.Error(err)
	}
	t.Log("Second Readings (post offset): ", env2)
	if env2.Temperature < (env.Temperature+(9_500*physic.MilliKelvin)) ||
		env2.Temperature > (env.Temperature+(10_500*physic.MilliKelvin)) {
		t.Errorf("offset temperature invalid. Expected ~ %s + 10C Got: %s", env.Temperature, env2.Temperature)
	}

	lLow := env.Humidity - 6*physic.PercentRH
	lHigh := env.Humidity - 4*physic.PercentRH
	if (env2.Humidity < lLow) ||
		(env2.Humidity > lHigh) {
		t.Errorf("offset humidity invalid. Expected ~ %s - 5%% Got: %s Lower Limit: %s, Upper Limit: %s", env.Humidity, env2.Humidity, lLow, lHigh)
	}

	// Issue a soft reset to make sure any alterations are undone.
	_ = dev.Reset()
}

// Tests using alert values.
func TestAlerts(t *testing.T) {

	dev, err := getDev(t, pbAlerts)
	if err != nil {
		t.Fatalf("failed to initialize hd302x: %v", err)
	}
	defer shutdown(t)
	env := physic.Env{}
	err = dev.Sense(&env)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Initial Temperature/Humidity Readings: %s", env)
	cfg, err := dev.Configuration()
	if err != nil {
		t.Fatal(err)
	}

	cfg.TemperatureOffset = 0
	cfg.HumidityOffset = 0
	cfg.AlertThresholds.Low.Temperature = 10 * physic.Celsius
	cfg.AlertThresholds.High.Temperature = 75 * physic.Celsius
	// Purposely set the low limit so it will trigger an alert on the status bit.
	cfg.AlertThresholds.Low.Humidity = env.Humidity + 5*physic.PercentRH
	cfg.AlertThresholds.High.Humidity = 75 * physic.PercentRH

	cfg.ClearThresholds.Low.Temperature = 15 * physic.Celsius
	cfg.ClearThresholds.High.Temperature = 70 * physic.Celsius
	cfg.ClearThresholds.Low.Humidity = cfg.AlertThresholds.Low.Humidity + 5*physic.PercentRH
	cfg.ClearThresholds.High.Humidity = 70 * physic.PercentRH
	t.Logf("Writing Configuration:\n%s", cfg)
	// write the Alert levels
	err = dev.SetConfiguration(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Re-read the configuration.
	cfg2, err := dev.Configuration()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("re-read configuration = \n%s", cfg2)
	// Trigger a read and then get the status word. We should have a humidity alert set...
	_ = dev.Sense(&env)
	status, err := dev.ReadStatus()
	if err != nil {
		t.Error(err)
	}
	_ = dev.Halt()

	// Verify things are within one lsb
	if !cfg.AlertThresholds.Low.ApproximatelyEquals(&cfg2.AlertThresholds.Low) {
		t.Errorf("error in low alert thresholds set: %s read: %s", cfg.AlertThresholds.Low.String(), cfg2.AlertThresholds.Low.String())
	}
	if !cfg.AlertThresholds.High.ApproximatelyEquals(&cfg2.AlertThresholds.High) {
		t.Errorf("error in high alert thresholds set: %s read: %s", cfg.AlertThresholds.High.String(), cfg2.AlertThresholds.High.String())
	}
	if !cfg.ClearThresholds.Low.ApproximatelyEquals(&cfg2.ClearThresholds.Low) {
		t.Errorf("error in low clear thresholds set: %s read: %s", cfg.ClearThresholds.Low.String(), cfg2.ClearThresholds.Low.String())
	}
	if !cfg.ClearThresholds.High.ApproximatelyEquals(&cfg2.ClearThresholds.High) {
		t.Errorf("error in high clear thresholds set: %s read: %s", cfg.ClearThresholds.High.String(), cfg2.ClearThresholds.High.String())
	}

	if status&StatusActiveAlerts != StatusActiveAlerts {
		t.Error("expected status active alerts to be set")
	}
	if status&StatusRHTrackingAlert != StatusRHTrackingAlert {
		t.Error("expected rh tracking alerts to be set")
	}
	if status&StatusRHLowTrackingAlert != StatusRHLowTrackingAlert {
		t.Error("expected RH Low Tracking alert status bit to be set")
	}

	// Issue a soft reset to make sure any alterations are undone.
	_ = dev.Reset()
}

// TestHeater turns on the sensor's integrated heater. The heater
// can be used to remove condensation from the sensor.
func TestHeater(t *testing.T) {
	dev, err := getDev(t, pbHeater)
	if err != nil {
		t.Fatalf("failed to initialize hd302x: %v", err)
	}
	defer shutdown(t)

	err = dev.SetHeater(PowerFull + 1)
	if err == nil {
		t.Error("expected error with invalid power value")
	}
	env := physic.Env{}
	env2 := physic.Env{}
	err = dev.Sense(&env)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("initial temperature: %s Humidity: %s", env.Temperature, env.Humidity)
	err = dev.SetHeater(PowerFull)
	defer func() {
		if err := dev.SetHeater(PowerOff); err != nil {
			t.Error(err)
		}
	}()
	if err != nil {
		t.Fatal(err)
	}
	if liveDevice {
		for range 5 {
			t.Log("Sleeping 5 seconds to test heater...")
			time.Sleep(time.Second)
		}
	}
	err = dev.Sense(&env2)

	if err != nil {
		t.Error(err)
	}
	t.Logf("final temperature after heater enabled: %s Humidity: %s", env2.Temperature, env2.Humidity)
	if env2.Temperature <= env.Temperature {
		t.Errorf("expected heater to increase sensor temperature. Initial: %s Final: %s", env.Temperature, env2.Temperature)
	}
}
