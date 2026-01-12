// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package lps2x_test

import (
	"fmt"
	"log"
	"time"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/lps2x"
	"periph.io/x/host/v3"
)

func Example() {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Use i2creg I²C bus registry to find the first available I²C bus.
	b, err := i2creg.Open("")
	if err != nil {
		log.Fatalf("failed to open I²C: %v", err)
	}
	defer b.Close()

	// Initialize the device.
	// Use default address, 25Hz sample rate, and average over 16 readings.
	dev, err := lps2x.New(b, lps2x.DefaultAddress, lps2x.SampleRate25Hertz, lps2x.AverageReadings16)
	if err != nil {
		log.Fatalf("failed to initialize lps2x: %v", err)
	}
	time.Sleep(time.Second)

	// Read environment data.
	e := physic.Env{}
	if err := dev.Sense(&e); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%8s %10s\n", e.Temperature, e.Pressure)
}
