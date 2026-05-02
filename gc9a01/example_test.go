// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package gc9a01_test

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"time"

	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/gc9a01"
	"periph.io/x/host/v3"
)

func Example() {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Use spireg SPI bus registry to find the first available SPI bus.
	p, err := spireg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer p.Close()

	// Data/Command pin.
	dc := gpioreg.ByName("GPIO25")
	if dc == nil {
		log.Fatal("failed to find DC pin")
	}

	// Optional reset pin.
	rst := gpioreg.ByName("GPIO27")

	dev, err := gc9a01.New(p, dc, rst, &gc9a01.DefaultOpts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("device=%s\n", dev.String())

	// Draw a colorful bullseye pattern.
	img := image.NewNRGBA(dev.Bounds())
	colors := []color.NRGBA{
		{0xFF, 0x00, 0x00, 0xFF}, // Red
		{0xFF, 0xA5, 0x00, 0xFF}, // Orange
		{0xFF, 0xFF, 0x00, 0xFF}, // Yellow
		{0x00, 0xFF, 0x00, 0xFF}, // Green
		{0x00, 0x00, 0xFF, 0xFF}, // Blue
		{0x4B, 0x00, 0x82, 0xFF}, // Indigo
		{0xEE, 0x82, 0xEE, 0xFF}, // Violet
	}
	cx, cy := 120, 120
	for y := 0; y < 240; y++ {
		for x := 0; x < 240; x++ {
			dx := float64(x - cx)
			dy := float64(y - cy)
			dist := math.Sqrt(dx*dx + dy*dy)
			ring := int(dist / 18)
			if ring < len(colors) {
				img.SetNRGBA(x, y, colors[ring])
			}
		}
	}
	if err := dev.Draw(dev.Bounds(), img, image.Point{}); err != nil {
		log.Fatal(err)
	}
	time.Sleep(3 * time.Second)

	// Draw a gradient.
	for y := 0; y < 240; y++ {
		for x := 0; x < 240; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 255 / 239),
				G: uint8(y * 255 / 239),
				B: 128,
				A: 255,
			})
		}
	}
	if err := dev.Draw(dev.Bounds(), img, image.Point{}); err != nil {
		log.Fatal(err)
	}
	time.Sleep(3 * time.Second)

	// Clear to white using image.Uniform.
	white := &image.Uniform{color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}}
	if err := dev.Draw(dev.Bounds(), white, image.Point{}); err != nil {
		log.Fatal(err)
	}

	// Draw a filled red circle in the center using draw.Draw.
	red := &image.Uniform{color.NRGBA{0xFF, 0x00, 0x00, 0xFF}}
	circle := image.NewNRGBA(dev.Bounds())
	draw.Draw(circle, circle.Bounds(), white, image.Point{}, draw.Src)
	for y := 0; y < 240; y++ {
		for x := 0; x < 240; x++ {
			dx := float64(x - cx)
			dy := float64(y - cy)
			if dx*dx+dy*dy <= 80*80 {
				circle.Set(x, y, red)
			}
		}
	}
	if err := dev.Draw(dev.Bounds(), circle, image.Point{}); err != nil {
		log.Fatal(err)
	}
	time.Sleep(3 * time.Second)

	_ = dev.Halt()
}
