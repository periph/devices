// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package nxp74hc595

import (
	"log"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
)

func Example() {
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}
    // Open the SPI Bus
	pc, err := spireg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()
	conn, err := pc.Connect(physic.MegaHertz, spi.Mode1, 8)
	if err != nil {
		log.Fatal(err)
	}
    // Create a new 74HC595 device conn that bus.
	dev, err := New(conn)
	if err != nil {
		log.Fatal(err)
	}
	// Get a GPIO group, and write values to it.
	gr, _ := dev.Group(0, 1, 2, 3, 4, 5, 6, 7, 8)
	for i := range 256 {
		gr.Out(gpio.GPIOValue(i), 0)
	}
}
