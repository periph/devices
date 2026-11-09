// Copyright 2018 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Examples of SN3218 initialization, power, and LED channels.

package sn3218_test

import (
	"log"
	"time"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/sn3218"
	"periph.io/x/host/v3"
)

func Example() {
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	b, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if closeErr := b.Close(); closeErr != nil {
			log.Print(closeErr)
		}
	}()

	d, err := sn3218.New(b)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if closeErr := d.Halt(); closeErr != nil {
			log.Print(closeErr)
		}
	}()

	// By default, the device is disabled and brightness is 0 for all LEDs
	// So let's set the brightness to a low value and enable the device to
	// get started
	if err := d.BrightnessAll(1); err != nil {
		log.Fatal(err)
	}
	if err := d.Sleep(); err != nil {
		log.Fatal(err)
	}

	// Switch LED 7 on
	if err := d.Switch(7, true); err != nil {
		log.Fatal("Error while switching LED", err)
	}

	time.Sleep(1000 * time.Millisecond)
}
