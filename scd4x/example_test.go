//go:build examples
// +build examples

// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package scd4x_test

import (
	"fmt"
	"log"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/scd4x"
	"periph.io/x/host/v3"
)

// basic example program for scd4x sensors using this library.
//
// To execute this as a stand-alone program:
//
// Copy the file example_test.go to a new directory.
// rename the file to main.go
// rename the Example() function to main, and the package to main
//
// execute:
//
//	go mod init mydomain.com/scd4x
//	go mod tidy
//	go build -o main main.go
//	./main
func Example() {
	fmt.Println("scd4x example program")
	if _, err := host.Init(); err != nil {
		fmt.Println(err)
	}
	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	dev, err := scd4x.NewI2C(bus, scd4x.SensorAddress)
	if err != nil {
		log.Fatal(err)
	}

	env := scd4x.Env{}
	err = dev.Sense(&env)
	if err == nil {
		fmt.Println(env.String())
	} else {
		fmt.Println(err)
	}

	cfg, err := dev.GetConfiguration()
	if err == nil {
		fmt.Printf("Configuration: %#v\n", cfg)
	} else {
		fmt.Println(err)
	}
	// Output: Temperature: 24.845°C Humidity: 32.3%rH CO2: 581 PPM
	// Configuration: &scd4x.DevConfig{AmbientPressure:0, ASCEnabled:true, ASCInitialPeriod:158400000000000, ASCStandardPeriod:561600000000000, ASCTarget:400, SensorAltitude:0, SerialNumber:127207989525260, TemperatureOffset:4, SensorType:0}
}
