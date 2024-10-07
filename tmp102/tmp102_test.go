// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package tmp102

import (
	"testing"
	"time"

	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/conn/v3/physic"
)

const (
	// Default value for alert low range value.
	DefaultLow physic.Temperature = physic.ZeroCelsius + 75*physic.Kelvin
	// Default value for alert high range value.
	DefaultHigh physic.Temperature = physic.ZeroCelsius + 80*physic.Kelvin

	addr uint16 = 0x48
)

func defaultOps() []i2ctest.IO {
	ops := []i2ctest.IO{
		{Addr: addr, W: []byte{_REGISTER_CONFIGURATION}},
		{Addr: addr, W: []byte{_REGISTER_CONFIGURATION, 0x00, 0x80}}, // Write the config
		{Addr: addr, W: []byte{_REGISTER_RANGE_LOW, 0x4b, 0x00}},     // Write the low alert temp
		{Addr: addr, W: []byte{_REGISTER_RANGE_HIGH, 0x50, 0x00}},    // Write the high alert temp
	}
	return ops
}

// TestSenseContinous test the sense continuous function, which
// implicitly tests Sense() and countToTemperature().
func TestSenseContinuous(t *testing.T) {
	// A set of counts, and the expected temperature value.
	tests := []struct {
		bits     []byte
		expected physic.Temperature
	}{
		{[]byte{0x64, 0x00}, physic.ZeroCelsius + 100*physic.Kelvin},
		{[]byte{0x50, 0x00}, physic.ZeroCelsius + 80*physic.Kelvin},
		{[]byte{0x32, 0x00}, physic.ZeroCelsius + 50*physic.Kelvin},
		{[]byte{0x19, 0x00}, physic.ZeroCelsius + 25*physic.Kelvin},
		{[]byte{0x00, 0x00}, physic.ZeroCelsius},
		{[]byte{0xe7, 0x00}, physic.ZeroCelsius - 25*physic.Kelvin},
		{[]byte{0xc9, 0x00}, physic.ZeroCelsius - 55*physic.Kelvin},
	}

	opts := Opts{
		SampleRate:   RateFourHertz,
		AlertSetting: ModeComparator,
		AlertLow:     DefaultLow,
		AlertHigh:    DefaultHigh,
	}

	ops := defaultOps()
	// Add the test values to our playback bus.
	for _, test := range tests {
		ops = append(ops, i2ctest.IO{Addr: addr, W: []byte{_REGISTER_TEMPERATURE}, R: test.bits})
	}
	pb := &i2ctest.Playback{Ops: ops, DontPanic: true, Count: 1}
	defer pb.Close()
	record := &i2ctest.Record{Bus: pb}

	tmp102, err := NewI2C(record, addr, &opts)
	if err != nil {
		t.Error(err)
		return
	}

	ch, err := tmp102.SenseContinuous(250 * time.Millisecond)
	if err != nil {
		t.Error(err)
		return
	}
	for count := 0; count < len(tests); count++ {
		env := <-ch
		t.Logf("Temperature = %.4f", env.Temperature.Celsius())
		if env.Temperature != tests[count].expected {
			t.Errorf("Error testing. Read: %.4f Expected %.4f", env.Temperature.Celsius(), tests[count].expected.Celsius())
		}

	}
	err = tmp102.Halt()
	if err != nil {
		t.Error(err)
	}
	t.Logf("record.ops=%#v", record.Ops)
}

func TestString(t *testing.T) {
	ops := defaultOps()
	pb := &i2ctest.Playback{Ops: ops, DontPanic: true, Count: 1}
	defer pb.Close()
	record := &i2ctest.Record{Bus: pb}
	tmp102, err := NewI2C(record, addr, nil)
	if err != nil {
		t.Error(err)
		return
	}

	s := tmp102.String()
	t.Log(s)
	if len(s) == 0 {
		t.Error("invalid String() result")
	}
}

func TestSetAlertMode(t *testing.T) {
	ops := make([]i2ctest.IO, 0)
	ops = append(ops, []i2ctest.IO{
		{Addr: addr, W: []byte{_REGISTER_CONFIGURATION}, R: []byte{0x00, 0x00}}, // Read the device config.
		{Addr: addr, W: []byte{_REGISTER_CONFIGURATION, 0x00, 0x80}},            // Set the device config.
		{Addr: addr, W: []byte{_REGISTER_RANGE_LOW}, R: []byte{0x4b, 0}},        // Read the low limit register
		{Addr: addr, W: []byte{_REGISTER_RANGE_HIGH}, R: []byte{0x50, 0}},       // Read the High Limit Register
		{Addr: addr, W: []byte{_REGISTER_RANGE_LOW, 0x4b, 0x80}},                // Set the read of the low limit to 75C
		{Addr: addr, W: []byte{_REGISTER_RANGE_HIGH, 0x4f, 0x80}},               // Set the read of the high limit to 80C
		{Addr: addr, W: []byte{_REGISTER_CONFIGURATION, 0x02, 0x00}},            // Add 1/2 Degree C to the range low
		{Addr: addr, W: []byte{_REGISTER_CONFIGURATION}, R: []byte{0x02, 0}},    // Read the confugration register.
		{Addr: addr, W: []byte{_REGISTER_RANGE_LOW}, R: []byte{0x4b, 0x80}},     // Read the low temp register
		{Addr: addr, W: []byte{_REGISTER_RANGE_HIGH}, R: []byte{0x4f, 0x80}},    // Read the high temp register
		{Addr: addr, W: []byte{_REGISTER_RANGE_LOW, 0x4b, 0x00}},                // write it back to 75C
		{Addr: addr, W: []byte{_REGISTER_RANGE_HIGH, 0x50, 0x00}},               // set it back to 80C
	}...)
	pb := &i2ctest.Playback{Ops: ops, DontPanic: true, Count: 1}
	defer pb.Close()
	record := &i2ctest.Record{Bus: pb}
	defer t.Logf("record=%#v", record)
	tmp102, err := NewI2C(record, addr, nil)
	if err != nil {
		t.Error(err)
		return
	}
	mode, low, high, err := tmp102.GetAlertMode()
	t.Logf("newMode=%d, newLow=%.4f, newHigh=%.4f", mode, low.Celsius(), high.Celsius())

	if err != nil {
		t.Error(err)
	}
	var newMode AlertMode
	if mode == ModeComparator {
		newMode = ModeInterrupt
	} else {
		newMode = ModeComparator
	}
	newLow := low + 500*physic.MilliKelvin
	newHigh := high - 500*physic.MilliKelvin
	t.Logf("newMode=%d, newLow=%.4f, newHigh=%.4f", newMode, newLow.Celsius(), newHigh.Celsius())
	err = tmp102.SetAlertMode(newMode, newLow, newHigh)

	if err != nil {
		t.Error(err)
	}

	checkMode, checkLow, checkHigh, err := tmp102.GetAlertMode()
	t.Logf("checkMode=%d checkLow=%.4f checkHigh=%.4f", checkMode, checkLow.Celsius(), checkHigh.Celsius())
	if err != nil {
		t.Error(err)
	}
	if checkMode != newMode || checkLow != newLow || checkHigh != newHigh {
		t.Errorf("Error setting/reading alert mode. Received: Mode=%d, Low=%.4f, High=%.4f. Expected: Mode:%d, Low=%.4f, High=%.4f",
			checkMode, checkLow.Celsius(), checkHigh.Celsius(),
			newMode, newLow.Celsius(), newHigh.Celsius())
	}

	err = tmp102.SetAlertMode(mode, low, high)
	if err != nil {
		t.Error(err)
	}
	checkMode, _, _, _ = tmp102.GetAlertMode()
	if checkMode != mode {
		t.Errorf("Error resetting mode. Got %d Expected %d", checkMode, mode)
	}
}
