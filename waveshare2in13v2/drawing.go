// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

import (
	"bytes"
	"image"
	"image/draw"

	"periph.io/x/devices/v3/ssd1306/image1bit"
)

// setMemoryArea configures the target drawing area (horizontal is in bytes,
// vertical in pixels).
func setMemoryArea(ctrl controller, area image.Rectangle) {
	ctrl.sendCommand(dataEntryModeSetting)
	ctrl.sendData([]byte{
		// Y increment, X increment; update address counter in X direction
		0b011,
	})

	ctrl.sendCommand(setRAMXAddressStartEndPosition)
	ctrl.sendData([]byte{
		// Start
		byte(area.Min.X),

		// End
		byte(area.Max.X - 1),
	})

	ctrl.sendCommand(setRAMYAddressStartEndPosition)
	ctrl.sendData([]byte{
		// Start
		byte(area.Min.Y % 0xFF),
		byte(area.Min.Y / 0xFF),

		// End
		byte((area.Max.Y - 1) % 0xFF),
		byte((area.Max.Y - 1) / 0xFF),
	})

	ctrl.sendCommand(setRAMXAddressCounter)
	ctrl.sendData([]byte{byte(area.Min.X)})

	ctrl.sendCommand(setRAMYAddressCounter)
	ctrl.sendData([]byte{
		byte(area.Min.Y % 0xFF),
		byte(area.Min.Y / 0xFF),
	})
}

type drawOpts struct {
	cmd     byte
	devSize image.Point
	dstRect image.Rectangle
	src     image.Image
	srcPts  image.Point
}

type drawSpec struct {
	// Destination on display in pixels, normalized to fit into actual size.
	DstRect image.Rectangle

	// Size of memory area to write; horizontally in bytes, vertically in
	// pixels.
	MemRect image.Rectangle

	// Size of image buffer, horizontally aligned to multiples of 8 pixels.
	BufferSize image.Point

	// Destination rectangle within image buffer.
	BufferRect image.Rectangle
}

func (o *drawOpts) spec() drawSpec {
	s := drawSpec{
		DstRect: image.Rectangle{Max: o.devSize}.Intersect(o.dstRect),
	}

	s.MemRect = image.Rect(
		s.DstRect.Min.X/8, s.DstRect.Min.Y,
		(s.DstRect.Max.X+7)/8, s.DstRect.Max.Y,
	)
	s.BufferSize = image.Pt(s.MemRect.Dx()*8, s.MemRect.Dy())
	s.BufferRect = image.Rectangle{
		Min: image.Point{X: s.DstRect.Min.X - (s.MemRect.Min.X * 8)},
		Max: image.Point{Y: s.DstRect.Dy()},
	}
	s.BufferRect.Max.X = s.BufferRect.Min.X + s.DstRect.Dx()

	return s
}

// drawImage sends an image to the controller after setting up the registers.
func drawImage(ctrl controller, opts *drawOpts) {
	s := opts.spec()

	if s.MemRect.Empty() {
		return
	}

	img := image1bit.NewVerticalLSB(image.Rectangle{Max: s.BufferSize})
	draw.Src.Draw(img, s.BufferRect, opts.src, opts.srcPts)

	setMemoryArea(ctrl, s.MemRect)

	ctrl.sendCommand(opts.cmd)

	rowData := make([]byte, s.MemRect.Dx())

	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < len(rowData); x++ {
			rowData[x] = 0

			for bit := 0; bit < 8; bit++ {
				if img.BitAt((x*8)+bit, y) {
					rowData[x] |= 0x80 >> bit
				}
			}
		}

		ctrl.sendData(rowData)
	}
}

func clearDisplay(ctrl controller, size image.Point, color image1bit.Bit) {
	var colorValue byte

	if color == image1bit.On {
		colorValue = 0xff
	}

	spec := (&drawOpts{
		devSize: size,
		dstRect: image.Rectangle{Max: size},
	}).spec()

	if spec.MemRect.Empty() {
		return
	}

	setMemoryArea(ctrl, spec.MemRect)

	ctrl.sendCommand(writeRAMBW)

	data := bytes.Repeat([]byte{colorValue}, spec.MemRect.Dx())

	for y := 0; y < spec.MemRect.Max.Y; y++ {
		ctrl.sendData(data)
	}
}
