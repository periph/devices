// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare1602_test

import (
	"log"
	"time"

	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/display/displaytest"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/waveshare1602"
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
	dev, err := waveshare1602.New(bus, waveshare1602.LCD1602RGBBacklight, 2, 16)
	if err != nil {
		log.Fatal(err)
	}
	_ = dev.Backlight(display.Intensity(0xff))
	_ = dev.Clear()
	time.Sleep(time.Second)

	_, _ = dev.WriteString("Hello")
	_ = dev.MoveTo(2, 2)
	time.Sleep(5 * time.Second)
	_, _ = dev.WriteString("1234567890")
	time.Sleep(10 * time.Second)
	displaytest.TestTextDisplay(dev, true)
	_ = dev.Halt()
}
