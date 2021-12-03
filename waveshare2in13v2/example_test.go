// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2_test

import (
	"image"
	"image/draw"
	"log"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/ssd1306/image1bit"
	"periph.io/x/devices/v3/waveshare2in13v2"
	"periph.io/x/host/v3"
)

func Example() {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Use spireg SPI bus registry to find the first available SPI bus.
	b, err := spireg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer b.Close()

	dev, err := waveshare2in13v2.NewHat(b, &waveshare2in13v2.EPD2in13v2) // Display config and size
	if err != nil {
		log.Fatalf("Failed to initialize driver: %v", err)
	}

	err = dev.Init(false)
	if err != nil {
		log.Fatalf("Failed to initialize display: %v", err)
	}

	// Draw on it. Black text on a white background.
	img := image1bit.NewVerticalLSB(dev.Bounds())
	draw.Draw(img, img.Bounds(), &image.Uniform{image1bit.On}, image.Point{}, draw.Src)
	f := basicfont.Face7x13
	drawer := font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{image1bit.Off},
		Face: f,
		Dot:  fixed.P(0, img.Bounds().Dy()-1-f.Descent),
	}
	drawer.DrawString("Hello from periph!")

	if err := dev.Draw(dev.Bounds(), img, image.Point{}); err != nil {
		log.Fatal(err)
	}
}

func Example_other() {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Use spireg SPI bus registry to find the first available SPI bus.
	b, err := spireg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer b.Close()

	dev, err := waveshare2in13v2.NewHat(b, &waveshare2in13v2.EPD2in13v2) // Display config and size
	if err != nil {
		log.Fatalf("Failed to initialize driver: %v", err)
	}

	err = dev.Init(false)
	if err != nil {
		log.Fatalf("Failed to initialize display: %v", err)
	}

	var img image.Image
	// Note: this code is commented out so periph does not depend on:
	//    "github.com/fogleman/gg"
	//    "github.com/golang/freetype/truetype"
	//    "golang.org/x/image/font/gofont/goregular"
	// bounds := dev.Bounds()
	// w := bounds.Dx()
	// h := bounds.Dy()
	// dc := gg.NewContext(w, h)
	// im, err := gg.LoadPNG("gopher.png")
	// if err != nil {
	// 	panic(err)
	// }
	// dc.SetRGB(1, 1, 1)
	// dc.Clear()
	// dc.SetRGB(0, 0, 0)
	// dc.Rotate(gg.Radians(90))
	// dc.Translate(0.0, -float64(h/2))
	// font, err := truetype.Parse(goregular.TTF)
	// if err != nil {
	// 	panic(err)
	// }
	// face := truetype.NewFace(font, &truetype.Options{
	// 	Size: 16,
	// })
	// dc.SetFontFace(face)
	// text := "Hello from periph!"
	// tw, th := dc.MeasureString(text)
	// dc.DrawImage(im, 120, 30)
	// padding := 8.0
	// dc.DrawRoundedRectangle(padding*2, padding*2, tw+padding*2, th+padding, 10)
	// dc.Stroke()
	// dc.DrawString(text, padding*3, padding*2+th)
	// for i := 0; i < 10; i++ {
	// 	dc.DrawCircle(float64(30+(10*i)), 100, 5)
	// }
	// for i := 0; i < 10; i++ {
	// 	dc.DrawRectangle(float64(30+(10*i)), 80, 5, 5)
	// }
	// dc.Fill()
	// img = dc.Image()

	if err := dev.Draw(dev.Bounds(), img, image.Point{}); err != nil {
		log.Fatal(err)
	}
}
