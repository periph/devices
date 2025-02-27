// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package pcf857x_test

import (
	"fmt"
	"log"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/pcf857x"
	"periph.io/x/host/v3"
)

func Example() {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Open default I²C bus.
	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatalf("failed to open I²C: %v", err)
	}
	defer bus.Close()

	// Create a new I2C IO extender
	extender, err := pcf857x.New(bus, pcf857x.DefaultAddress, pcf857x.PCF8574)
	if err != nil {
		log.Fatalln(err)
	}

	for _, pin := range extender.Pins {
		err = pin.In(gpio.Float, gpio.NoEdge)
		if err != nil {
			log.Fatalln(err)
		}
		level := pin.Read()
		fmt.Printf("%s\t%s\n", pin.Name(), level.String())
	}

	if err != nil {
		log.Fatalln(err)
	}
}
