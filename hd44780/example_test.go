// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package hd44780_test

import (
	"errors"
	"fmt"
	"log"
	"time"

	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/display/displaytest"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/hd44780"
	"periph.io/x/devices/v3/pcf857x"
	"periph.io/x/host/v3"
	"periph.io/x/host/v3/gpioioctl"
)

// This example shows using a gpio.Group with an HD44780 display. For this
// example its using the periph.io/x/host/gpioioctl package to obtain
// the gpio.Group and pins. You can use an I/O device that implements
// gpio.Group and gpio.PinOut to drive a display.
func Example() {
	var err error
	if _, err = host.Init(); err != nil {
		log.Fatal(err)
	}
	chip := gpioioctl.Chips[0]
	var ls gpio.Group
	// Using a group to obtain the pins. The first 4 pins in the group are the
	// data pins, and the remaining ones are reset, enable, and backlight.
	// For 8 bit mode, specify additional data pins.
	ls, err = chip.LineSet(gpioioctl.LineOutput, gpio.NoEdge, gpio.PullNoChange,
		"GPIO27", "GPIO22", "GPIO23", "GPIO24", "GPIO17", "GPIO18", "GPIO25")
	if err != nil {
		log.Fatal(err)
	}
	pins := ls.Pins()
	reset := pins[4].(gpio.PinOut)
	enable := pins[5].(gpio.PinOut)
	bl := hd44780.NewBacklight(pins[6].(gpio.PinOut))
	lcd, err := hd44780.NewHD44780(ls, reset, enable, bl, 2, 16)
	if err != nil {
		log.Fatal(err)
	}
	n, err := lcd.WriteString("Hello")
	time.Sleep(5 * time.Second)
	fmt.Printf("n=%d, err=%s\n", n, err)
	fmt.Println("lcd=", lcd.String())

	_ = lcd.Home()
	_ = lcd.MoveTo(1, 1)
	_, _ = lcd.WriteString("Line 1")
	_ = lcd.MoveTo(2, 2)
	_, _ = lcd.WriteString("Line 2")
	time.Sleep(5 * time.Second)
	_ = lcd.Clear()

	fmt.Println("calling TestTextDisplay")

	errs := displaytest.TestTextDisplay(lcd, true)
	fmt.Println("back from TestTextDisplay")
	for _, e := range errs {
		if !errors.Is(e, display.ErrNotImplemented) {
			log.Println(e)
		}
	}
}

// Create a new HD44780 that uses the Adafruit I2C/SPI Backpack.
func ExampleNewAdafruitI2CBackpack() {
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
	dev, err := hd44780.NewAdafruitI2CBackpack(bus, 0x20, 4, 20)
	fmt.Println(dev.String())
	if err != nil {
		log.Fatal(err)
	}
	_ = dev.Clear()
	_, _ = dev.WriteString("Hello")
	fmt.Println("wrote hello")
	time.Sleep(5 * time.Second)
	fmt.Println("calling test text display")
	_ = displaytest.TestTextDisplay(dev, true)
}

func ExampleNewAdafruitSPIBackpack() {
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	pc, err := spireg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()
	conn, err := pc.Connect(physic.MegaHertz, spi.Mode1, 8)
	if err != nil {
		log.Fatal(err)
	}
	display, err := hd44780.NewAdafruitSPIBackpack(conn, 2, 26)
	if err != nil {
		log.Fatal(err)
	}

	_ = display.Clear()
	_, _ = display.WriteString("Hello")
	time.Sleep(5 * time.Second)
	_ = displaytest.TestTextDisplay(display, true)
}

func ExampleNewPCF857xBackpack() {
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
	dev, err := hd44780.NewPCF857xBackpack(bus, pcf857x.DefaultAddress, 4, 20)
	fmt.Println(dev.String())
	if err != nil {
		log.Fatal(err)

	}
	for range 5 {
		fmt.Println("toggling backlight")
		_ = dev.Backlight(0)
		time.Sleep(500 * time.Millisecond)
		_ = dev.Backlight(255)
		time.Sleep(500 * time.Millisecond)

	}
	_ = dev.Clear()
	_, _ = dev.WriteString("Hello")
	fmt.Println("wrote hello")
	time.Sleep(5 * time.Second)
	fmt.Println("calling test text display")
	_ = displaytest.TestTextDisplay(dev, true)
}
