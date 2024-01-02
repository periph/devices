// Copyright 2023 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package adxl345

import (
	"fmt"
	"log"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
	"time"
)

var I2CAddr uint16 = 0x53

// ExampleNewI2C uses an adxl345 device connected by I²C.
// You can set the I²C address by setting I2CAddr (default is 0x53).
// it reads the acceleration values every 30ms for 30 seconds.
// You can i use `i2dctools`  to find the I²C bus number
// e.g : sudo apt-get install i2c-tools
//
//	sudo i2cdetect -y 1
func ExampleNewI2C() {
	mustInitHost()

	// Use i2creg  to find the first available  I²C bus.
	// Generally I2C1 on raspberry pi.
	p, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(p.String())

	defer p.Close()

	d, err := NewI2C(p, I2CAddr, &DefaultOpts)
	if err != nil {
		panic(err)
	}
	measure(d, 30*time.Second)
}

// ExampleNewSpi uses an adxl345 device connected by SPI.
// it reads the acceleration values every 30ms for 30 seconds.
func ExampleNewSpi() {

	mustInitHost()

	// Use spireg SPI port registry to find the first available SPI bus.
	p, err := spireg.Open("")
	if err != nil {
		log.Fatal(err)
	}

	defer p.Close()

	d, err := NewSpi(p, &DefaultOpts)
	if err != nil {
		panic(err)
	}

	measure(d, 30*time.Second)
}

// mustInitHost Make sure host is initialized.
func mustInitHost() {

	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}
}

// measure reads the acceleration values every 30ms for <duration> seconds
func measure(d *Dev, duration time.Duration) {

	fmt.Println(d.String())

	mode := d.Mode()
	// use a ticker to read the acceleration values every 200ms
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	// stop after 3 seconds
	stop := time.After(duration)

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			a := d.Update()
			fmt.Println(mode, a)
		}
	}
}
