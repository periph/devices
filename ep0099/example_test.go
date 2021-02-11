// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package ep0099_test

import (
	"log"
	"time"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/ep0099"
	"periph.io/x/host/v3"
)

func Example() {
	// Initializes host to manage bus and devices
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Opens default bus
	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer bus.Close()

	// Initializes device using current I2C bus and device address.
	// Address should be provided as configured on the board's DIP switches.
	dev, err := ep0099.New(bus, 0x10)
	if err != nil {
		log.Fatal(err)
	}
	defer dev.Halt()

	// Run device demo
	for _, channel := range dev.AvailableChannels() {
		state, _ := dev.State(channel)
		log.Printf("[channel %#x] Initial state: %s", channel, state)

		dev.On(channel)
		state, _ = dev.State(channel)
		log.Printf("[channel %#x] State after .On: %s", channel, state)

		dev.Off(channel)
		state, _ = dev.State(channel)
		log.Printf("[channel %#x] State after .Off: %s", channel, state)

		time.Sleep(2 * time.Second)
	}
}
