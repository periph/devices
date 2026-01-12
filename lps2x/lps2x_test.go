// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package lps2x

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/conn/v3/physic"
)

var recordingData = map[string][]i2ctest.IO{
	"TestCountToPressure": []i2ctest.IO{
		{Addr: 0x5c, W: []uint8{0xf}, R: []uint8{0xb4}},
		{Addr: 0x5c, W: []uint8{0x10, 0x10}}},
	"TestBasic": []i2ctest.IO{
		{Addr: 0x5c, W: []uint8{0xf}, R: []uint8{0xb4}},
		{Addr: 0x5c, W: []uint8{0x10, 0x10}}},
	"TestSense": []i2ctest.IO{
		{Addr: 0x5c, W: []uint8{0xf}, R: []uint8{0xb4}},
		{Addr: 0x5c, W: []uint8{0x10, 0x10}},
		{Addr: 0x5c, W: []uint8{0x27}, R: []uint8{0x33, 0xbf, 0x19, 0x34, 0x2f, 0x9}}},
	"TestSenseContinuous": []i2ctest.IO{
		{Addr: 0x5c, W: []uint8{0xf}, R: []uint8{0xb4}},
		{Addr: 0x5c, W: []uint8{0x10, 0x10}},
		{Addr: 0x5c, W: []uint8{0x27}, R: []uint8{0x33, 0x72, 0x1a, 0x34, 0x2f, 0x9}},
		{Addr: 0x5c, W: []uint8{0x27}, R: []uint8{0x33, 0xe7, 0x1a, 0x34, 0x2f, 0x9}},
		{Addr: 0x5c, W: []uint8{0x27}, R: []uint8{0x33, 0x9e, 0x19, 0x34, 0x2f, 0x9}},
		{Addr: 0x5c, W: []uint8{0x27}, R: []uint8{0x33, 0xd4, 0x18, 0x34, 0x2f, 0x9}},
		{Addr: 0x5c, W: []uint8{0x27}, R: []uint8{0x33, 0x3e, 0x1a, 0x34, 0x2f, 0x9}},
		{Addr: 0x5c, W: []uint8{0x27}, R: []uint8{0x33, 0x93, 0x1a, 0x34, 0x2f, 0x9}},
		{Addr: 0x5c, W: []uint8{0x27}, R: []uint8{0x33, 0x51, 0x1a, 0x34, 0x2f, 0x9}},
		{Addr: 0x5c, W: []uint8{0x27}, R: []uint8{0x33, 0xc9, 0x1a, 0x34, 0x2f, 0x9}},
		{Addr: 0x5c, W: []uint8{0x27}, R: []uint8{0x33, 0xa, 0x1a, 0x34, 0x2f, 0x9}},
		{Addr: 0x5c, W: []uint8{0x27}, R: []uint8{0x33, 0x72, 0x1a, 0x34, 0x2f, 0x9}}},
	"TestCountToTemp": []i2ctest.IO{
		{Addr: 0x5c, W: []uint8{0xf}, R: []uint8{0xb4}},
		{Addr: 0x5c, W: []uint8{0x10, 0x10}}},
}

var liveDevice bool
var timeDurationMultiplier time.Duration

func getDev(testName string) (*Dev, error) {
	ops := recordingData[testName]
	dev, err := New(&i2ctest.Playback{Ops: ops, DontPanic: true}, DefaultAddress, SampleRate4Hertz, AverageNone)
	return dev, err
}

func TestInt24ToInt64(t *testing.T) {
	if convert24BitTo64Bit([]byte{0xff, 0xff, 0xff}) != 0xffffffff {
		t.Error("Error converting -1 to 32bits")
		t.Errorf("Error converting -1 to 64 bits, got 0x%x", convert24BitTo64Bit([]byte{0xff, 0xff, 0xff}))
	}
	if convert24BitTo64Bit([]byte{0xf0, 0xff, 0xff}) != 0xfffffff0 {
		t.Errorf("Error converting -16 to 64 bits, got 0x%x", convert24BitTo64Bit([]byte{0xf0, 0xff, 0xff}))
	}
	if convert24BitTo64Bit([]byte{0x10, 0, 0}) != 16 {
		t.Error("Error converting 16 to 32bits")
	}
}

func TestCountToTemp(t *testing.T) {
	dev, _ := getDev(t.Name())
	c := dev.countToTemp(0)
	if c != physic.ZeroCelsius {
		t.Error("expected zero celsius for zero count!")
	}
	c = dev.countToTemp(5000)
	if c != (physic.ZeroCelsius + 50*physic.Kelvin) {
		t.Errorf("expected 50 celsius received %s", c.String())
	}
}

func TestCountToPressure(t *testing.T) {
	dev, _ := getDev(t.Name())
	p := dev.countToPressure(0)
	if p != 0 {
		t.Errorf("expected 0 Pa received %s", p.String())
	}

	p = dev.countToPressure(4096 * 10)
	if p != (10 * physic.Pascal * 100) {
		t.Errorf("expected 1000 Pa received %s", p.String())
	}
	dev.fsMode = 1
	p = dev.countToPressure(4096 * 10)
	if p != (20 * physic.Pascal * 100) {
		t.Errorf("expected 2000pa received %s", p.String())
	}

}

func TestBasic(t *testing.T) {
	// Test String()
	dev, _ := getDev(t.Name())
	s := dev.String()
	if len(s) == 0 {
		t.Errorf("String() returned empty")
	}
	if s != lps28dfw {
		t.Errorf("received model %s, expected %s", s, lps28dfw)
	}
	// Test Precision()
	env := physic.Env{}
	dev.Precision(&env)
	if env.Humidity != 0 {
		t.Error("expected 0% RH")
	}
	if env.Temperature != (physic.Kelvin / 100) {
		t.Errorf("expected precision of 1/100 kelvin got %s", env.Temperature.String())
	}
	if env.Pressure != HectoPascal {
		t.Errorf("expected pressure precision of 1 HectoPascal got %s", env.Pressure.String())
	}
}

func TestSense(t *testing.T) {
	dev, err := getDev(t.Name())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * timeDurationMultiplier * time.Second)
	env := physic.Env{}
	err = dev.Sense(&env)
	if err != nil {
		t.Error(err)
	}
	t.Logf("dev=%s", dev.String())
	t.Logf("Temperature: %s Pressure: %s (PSI=%f)", env.Temperature.String(), env.Pressure.String(), float64(env.Pressure/physic.Pascal)*float64(0.000145038))
}

func TestSenseContinuous(t *testing.T) {
	dev, err := getDev(t.Name())
	var d time.Duration
	if liveDevice {
		d = time.Second
	} else {
		d = 250 * time.Millisecond
	}
	if err != nil {
		t.Fatal(err)
	}
	// So the default is 4hz, average none, so the min reading rate is 250ms
	_, err = dev.SenseContinuous(100 * time.Millisecond)
	if err == nil {
		t.Error("expected error on insufficient sense continuous duration")
	}

	counter := atomic.Int32{}
	chEnd := make(chan struct{})
	chRead, err := dev.SenseContinuous(d)
	if err != nil {
		t.Fatal(err)
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(ch <-chan physic.Env, chEnd chan struct{}) {
		var env physic.Env
		for {
			select {
			case env = <-ch:
				t.Logf("received %#v", env)
				counter.Add(1)
			case <-chEnd:
				wg.Done()
				break
			}
		}
	}(chRead, chEnd)
	time.Sleep(10 * d)
	chEnd <- struct{}{}
	wg.Wait()
	if counter.Load() != 10 {
		t.Errorf("Expected reading count of 10, received %d", counter.Load())
	}
}
