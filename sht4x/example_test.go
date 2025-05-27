// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sht4x_test

import (
	"log"
	"time"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/sht4x"
	"periph.io/x/host/v3"
)

// Example shows creating an SHT-4X sensor and reading from it.
func Example() {
	if _, err := host.Init(); err != nil {
		log.Fatal("Error calling host.init()")
	}
	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer bus.Close()

	dev, err := sht4x.New(bus, sht4x.DefaultAddress)
	if err != nil {
		log.Fatal(err)
	}

	env := &physic.Env{}

	for range 10 {
		err = dev.Sense(env)
		if err != nil {
			log.Println(err)
		} else {
			log.Printf("Temperature: %s   Humidity: %s\n", env.Temperature, env.Humidity)
		}
		time.Sleep(time.Second)
	}
}
