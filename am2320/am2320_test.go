// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package am2320

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
var liveDevice bool

// Playback values for a single sense operation.
var pbSense = []i2ctest.IO{
	{Addr: SensorAddress, W: []uint8{0x0}},
	{Addr: SensorAddress, W: []uint8{0x3, 0x0, 0x4}, R: []uint8{0x3, 0x4, 0x1, 0x5c, 0x0, 0xef, 0x71, 0x8a}}}

func init() {
	var err error

	liveDevice = os.Getenv("AM2320") != ""

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

func TestBasic(t *testing.T) {
	dev := Dev{}
	env := &physic.Env{}
	dev.Precision(env)
	if env.Pressure != 0 {
		t.Error("this device doesn't measure pressure")
	}
	if 10*env.Temperature != physic.Celsius {
		t.Error("incorrect temperature precision value")
	}
	if env.Humidity != physic.MilliRH {
		t.Error("incorrect humidity precision")
	}

	s := dev.String()
	if len(s) == 0 {
		t.Error("invalid value for String()")
	}

	// Check the CRC Calculation algorithm using the data supplied by the vendor.
	crcTest := []byte{0x03, 0x04, 0x01, 0xf4, 0x00, 0xfa, 0x31, 0xa5}
	if !checkCRC(crcTest) {
		t.Error("crc error")
	}
	// ensure a corruption is detected.
	crcTest[0] = crcTest[0] ^ 0xff
	if checkCRC(crcTest) {
		t.Error("crc error")
	}
}

func TestSense(t *testing.T) {
	d, err := getDev(t, pbSense)
	if err != nil {
		t.Fatalf("failed to initialize am2320: %v", err)
	}
	defer shutdown(t)

	// Read temperature and humidity from the sensor
	e := physic.Env{}

	if err := d.Sense(&e); err != nil {
		t.Fatal(err)
	}
	t.Logf("%8s %9s", e.Temperature, e.Humidity)

	if !liveDevice {
		// The playback temp is 23.9C Ensure that's what we got.
		expected := physic.ZeroCelsius + 23_900*physic.MilliKelvin
		if e.Temperature != expected {
			t.Errorf("incorrect temperature value read. Expected: %s (%d) Found: %s (%d)",
				e.Temperature.String(),
				e.Temperature,
				expected.String(),
				expected)
		}

		// 34.8% expected.
		expectedRH := 34*physic.PercentRH + 8*physic.MilliRH
		if e.Humidity != expectedRH {
			t.Errorf("incorrect humidity value read. Expected: %s (%d) Found: %s (%d)",
				e.Humidity.String(),
				e.Humidity,
				expectedRH.String(),
				expectedRH)
		}
	}
}

func TestSenseContinuous(t *testing.T) {
	readCount := 10

	// make 10 copies of the single reading playback data.
	pb := make([]i2ctest.IO, 0, len(pbSense)*10)
	for range readCount {
		pb = append(pb, pbSense...)
	}

	d, err := getDev(t, pb)
	if err != nil {
		t.Fatalf("failed to initialize am2320: %v", err)
	}
	defer shutdown(t)

	_, err = d.SenseContinuous(time.Second)
	if err == nil {
		t.Error("SenseContinuous() accepted invalid reading interval")
	}
	ch, err := d.SenseContinuous(3 * time.Second)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(3 * time.Duration(readCount) * time.Second)
		err := d.Halt()
		if err != nil {
			t.Error(err)
		}
	}()

	count := 0
	for e := range ch {
		count += 1
		t.Log(time.Now(), e)
	}
	if count < (readCount-1) || count > (readCount+1) {
		t.Errorf("expected %d readings. received %d", readCount, count)
	}
}
