// Copyright 2018 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package ssd1306_test

import (
    "flag"
	"image"
	"log"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/ssd1306"
	"periph.io/x/devices/v3/ssd1306/image1bit"
	"periph.io/x/host/v3"
)

func Example() {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}
	var width = flag.Int("width", 128, "Display Width")
	var height = flag.Int("height", 64, "Display Height")
	flag.Parse()
	// Use i2creg I²C bus registry to find the first available I²C bus.
	b, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer b.Close()
	opts := ssd1306.DefaultOpts
	opts.W = *width
	opts.H = *height
	if opts.H == 32 {
		opts.Sequential = true
	}
	dev, err := ssd1306.NewI2C(b, &opts)
	if err != nil {
		log.Fatalf("failed to initialize display: %s", err.Error())
	}
	fmt.Printf("device=%s\n", dev.String())

	img := image1bit.NewVerticalLSB(dev.Bounds())
	// Note: this code is commented out so periph does not depend on:
	//    "golang.org/x/image/font"
	//    "golang.org/x/image/font/basicfont"
	//    "golang.org/x/image/math/fixed"
	//
	// Draw on it.
	/*
		f := basicfont.Face7x13
		drawer := font.Drawer{
			Dst: img,
			Src: &image.Uniform{image1bit.On},

			Face: f,
			Dot:  fixed.P(0, img.Bounds().Dy()-1-f.Descent),
		}
		drawer.DrawString("Hello from periph!")
		_ = dev.Draw(dev.Bounds(), img, image.Point{})
		time.Sleep(5 * time.Second)
	*/

	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 255}
	colors := []color.RGBA{white, black}

	rectNum := 0
	// Draw some nested rectangles
	for w, h := opts.W, opts.H; w > 0 && h > 0; w, h = w-4, h-4 {
		rect := image.Rect(0, 0, w, h)
		draw.Draw(img, rect.Add(image.Point{X: rectNum * 2, Y: rectNum * 2}), &image.Uniform{colors[rectNum%2]}, image.Point{}, draw.Src)
		rectNum += 1
	}

	if err := dev.Draw(dev.Bounds(), img, image.Point{}); err != nil {
		log.Fatal(err)
	}
	time.Sleep(5 * time.Second)

	// Draw a Sine Wave
	_ = dev.Invert(true)
	img = image1bit.NewVerticalLSB(dev.Bounds())
	img.DrawHLine(0, opts.W, opts.H>>1-1, image1bit.On)
	img.DrawVLine(0, opts.H, opts.W>>1-1, image1bit.On)
	angle := float64(0)
	angleStep := float64((4 * math.Pi) / float64(opts.W))
	scale := float64((opts.H >> 1) - 4)
	for step := opts.W - 1; step >= 0; step -= 1 {
		y := int(float64(math.Sin(angle)*scale)) + opts.H>>1
		img.SetBit(step, y, image1bit.On)
		angle += angleStep
	}
	_ = dev.Draw(dev.Bounds(), img, image.Point{})
	time.Sleep(10 * time.Second)

	_ = dev.Halt()
}
