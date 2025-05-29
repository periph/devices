// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sht4x

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/conn/v3/physic"
)

var liveDevice bool

func getDev(testName string) (*Dev, error) {
	return New(&i2ctest.Playback{Ops: recordingData[testName], DontPanic: true}, DefaultAddress)
}

func TestBasic(t *testing.T) {
	t.Logf("liveDevice=%t", liveDevice)
	dev, err := getDev(t.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Test String
	s := dev.String()
	if len(s) == 0 {
		t.Error("string returned empty")
	}

	// Test Serial Number
	sn, err := dev.SerialNumber()
	if err == nil {
		if sn == 0 {
			t.Error("invalid serial number")
		} else {
			t.Logf("SerialNumber=0x%x", sn)
		}
	} else {
		t.Error(err)
	}
}

func TestCountToTemp(t *testing.T) {
	temp := countToTemp(0)
	if temp != minTemperature {
		t.Errorf("invalid temperature %s. Expected -40", temp)
	}
	temp = countToTemp(0xffff)
	if temp != maxTemperature {
		t.Errorf("invalid temperature %s. Expected 125", temp)
	}
	temp = countToTemp(0x8000)
	tTest := 42.5 + physic.ZeroCelsius.Celsius()
	diff := physic.Temperature(math.Abs(tTest-float64(temp.Celsius()))) * physic.Kelvin
	if diff > 2*physic.MilliKelvin {
		t.Errorf("invalid temperature expected %f. got %s diff=%s", tTest, temp, diff)
	}
}

func TestCountToHumidity(t *testing.T) {
	rh := countToHumidity(0)
	if rh != minRH {
		t.Errorf("received RH %s expected %s", rh, minRH)
	}
	rh = countToHumidity(0xffff)
	if rh != maxRH {
		t.Errorf("received RH %s expected %s", rh, maxRH)
	}
	rh = countToHumidity(0x8000)
	expected := physic.RelativeHumidity(56.5 * float64(physic.PercentRH))
	diff := rh - expected
	if diff > 2*physic.MilliRH {
		t.Errorf("received rh %s expected %s diff=%v", rh, expected, diff)
	}
}

// Test turning on the heater at various power levels and durations.
func TestHeater(t *testing.T) {
	dev, err := getDev(t.Name())
	if err != nil {
		t.Fatal(err)
	}
	// Test Invalid parameters
	_, err = dev.SetHeater(Power20mW, HeaterDuration(10*time.Second))
	if err == nil {
		t.Error("SetHeater() invalid duration did not generate error.")
	}
	_, err = dev.SetHeater(HeaterPower(500), Duration100ms)
	if err == nil {
		t.Error("SetHeater() invalid power level did not generate error.")
	}
	initEnv := &physic.Env{}
	// Iterate over the allowed durations
	for _, duration := range []HeaterDuration{Duration100ms, Duration1s} {
		// Iterate over the supported heater power levels
		for _, power := range []HeaterPower{Power20mW, Power110mW, Power200mW} {
			// Read the initial temperature at the test start.
			err := dev.Sense(initEnv)
			if err != nil {
				t.Error(err)
				continue
			}
			// Turn the heater on 3 times
			var diffLast float64
			for range 3 {
				env, err := dev.SetHeater(power, duration)
				if err != nil {
					t.Error(err)
					break
				}
				// Confirm that the difference between the initial temperature and
				// the temperature after the heater was turned on is > 0
				diff := env.Temperature.Celsius() - initEnv.Temperature.Celsius()
				t.Logf("initTemp=%s heaterTemp=%s, diff=%f", initEnv.Temperature, env.Temperature, diff)
				if diff <= 0 || diff <= diffLast {
					t.Errorf("heater error power=%d, Duration=%v diff=%f expected > 0", power, duration, diff)
				}
				diffLast = diff
			}
			if liveDevice {
				// Give the thermometer core time to cool off
				time.Sleep(10 * time.Second)
			}
		}
	}
}

func TestReset(t *testing.T) {
	dev, err := getDev(t.Name())
	if err != nil {
		t.Fatal(err)
	}
	err = dev.Reset()
	if err != nil {
		t.Error(err)
	}
}

func TestSense(t *testing.T) {
	dev, err := getDev(t.Name())
	if err != nil {
		t.Fatal(err)
	}
	env := &physic.Env{}
	err = dev.Sense(env)
	if err != nil {
		t.Error(err)
	}
	t.Logf("env=%#v temperature=%s humidity=%s", *env, env.Temperature.String(), env.Humidity.String())
}

func TestSenseContinuous(t *testing.T) {
	readCount := int32(10)
	dev, err := getDev(t.Name())
	if err != nil {
		t.Fatal(err)
	}
	_, err = dev.SenseContinuous(time.Millisecond)
	if err == nil {
		t.Error("SenseContinuous() doesn't return an error on too short a duration.")
	}
	ch, err := dev.SenseContinuous(100 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	_, err = dev.SenseContinuous(time.Second)
	if err == nil {
		t.Error("expected an error for attempting concurrent SenseContinuous")
	}

	counter := atomic.Int32{}
	tEnd := time.Now().UnixMilli() + int64(readCount+2)*100
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			// Stay here until we get the expected number of reads, or the time
			// has expired and when we do, Halt the SenseContinuous.
			if counter.Load() >= readCount || time.Now().UnixMilli() >= tEnd {
				t.Logf("calling halt!")
				err := dev.Halt()
				t.Logf("halt() returned")
				if err != nil {
					t.Error(err)
				}
				wg.Done()
				return
			}
		}
	}()
	// Iterate over the channel until it's closed.
	for e := range ch {
		counter.Add(1)
		t.Log(time.Now(), e, "count=", counter.Load())
	}
	if counter.Load() < readCount || counter.Load() > (readCount+1) {
		t.Errorf("expected %d readings. received %d", readCount, counter.Load())
	}
	wg.Wait()
}
