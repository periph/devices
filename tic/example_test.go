// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package tic_test

import (
	"log"
	"time"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/tic"
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

	// Create a new motor controller.
	dev, err := tic.NewI2C(bus, tic.Tic36v4, tic.I2CAddr)
	if err != nil {
		log.Fatal(err)
	}

	// Set the current limit with respect to the motor.
	if err := dev.SetCurrentLimit(1000 * physic.MilliAmpere); err != nil {
		log.Fatalf("failed to set current limit: %v", err)
	}

	// The "Exit safe start" command is required before the motor can move.
	if err := dev.ExitSafeStart(); err != nil {
		log.Fatalf("failed to exit safe start: err %v", err)
	}

	// Set the target velocity to 200 microsteps per second.
	if err := dev.SetTargetVelocity(2000000); err != nil {
		log.Fatalf("failed to set target velocity: err %v", err)
	}

	// Use a ticker to frequently send commands before the timeout period
	// elapses (1000ms default).
	ticker := time.NewTicker(900 * time.Millisecond)
	defer ticker.Stop()

	// Stop after 3 seconds.
	stop := time.After(3 * time.Second)

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			// Any command sent to the Tic will reset the timeout. However,
			// this can be done explicitly using ResetCommandTimeout().
			if err := dev.ResetCommandTimeout(); err != nil {
				log.Fatalf("failed to reset command timeout: %v", err)
			}
		}
	}
}
