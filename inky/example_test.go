// Copyright 2019 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package inky_test

import (
	"flag"
	"image"
	"image/png"
	"log"
	"os"

	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/inky"
	"periph.io/x/host/v3"
)

func Example() {
	path := flag.String("image", "", "Path to image file (212x104) to display")
	flag.Parse()

	f, err := os.Open(*path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	if _, err = host.Init(); err != nil {
		log.Fatal(err)
	}

	b, err := spireg.Open("SPI0.0")
	if err != nil {
		log.Fatal(err)
	}

	dc := gpioreg.ByName("22")
	reset := gpioreg.ByName("27")
	busy := gpioreg.ByName("17")

	dev, err := inky.New(b, dc, reset, busy, &inky.Opts{
		Model:       inky.PHAT,
		ModelColor:  inky.Red,
		BorderColor: inky.Black,
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := dev.Draw(img.Bounds(), img, image.Point{}); err != nil {
		log.Fatal(err)
	}
}

func ExampleNewImpression() {
	path := flag.String("image", "", "Path to image file (600x448) to display")
	flag.Parse()

	f, err := os.Open(*path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	m, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	b, err := spireg.Open("SPI0.0")
	if err != nil {
		log.Fatal(err)
	}

	dc := gpioreg.ByName("22")
	reset := gpioreg.ByName("27")
	busy := gpioreg.ByName("17")

	eeprom, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer eeprom.Close()

	o, err := inky.DetectOpts(eeprom)
	if err != nil {
		log.Fatal(err)
	}

	dev, err := inky.NewImpression(b, dc, reset, busy, o)
	if err != nil {
		log.Fatal(err)
	}

	if err := dev.Draw(m.Bounds(), m, image.Point{}); err != nil {
		log.Fatal(err)
	}
}
