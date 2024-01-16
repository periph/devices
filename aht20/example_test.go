// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package aht20_test

import (
	"fmt"
	"log"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/aht20"
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

	// Create a new AHT20 device using I²C bus.
	d, err := aht20.NewI2C(b, nil) // nil for default options or &aht20.DefaultOpts
	if err != nil {
		log.Fatalf("failed to initialize AHT20: %v", err)
	}

	// Read temperature and humidity from the sensor
	e := physic.Env{}
	if err := d.Sense(&e); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%8s %9s\n", e.Temperature, e.Humidity)
}
