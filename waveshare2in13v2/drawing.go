// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

import (
	"encoding/binary"
	"image"
	"image/draw"

	"periph.io/x/devices/v3/ssd1306/image1bit"
)

// setMemoryArea configures the target drawing area (horizontal is in bytes,
// vertical in pixels).
func setMemoryArea(ctrl controller, area image.Rectangle) {
	startX, endX := uint8(area.Min.X), uint8(area.Max.X-1)
	startY, endY := uint16(area.Min.Y), uint16(area.Max.Y-1)

	startEndY := [4]byte{}
	binary.LittleEndian.PutUint16(startEndY[0:], startY)
	binary.LittleEndian.PutUint16(startEndY[2:], endY)

	ctrl.sendCommand(dataEntryModeSetting)
	ctrl.sendData([]byte{
		// Y increment, X increment; update address counter in X direction
		0b011,
	})

	ctrl.sendCommand(setRAMXAddressStartEndPosition)
	ctrl.sendData([]byte{startX, endX})

	ctrl.sendCommand(setRAMYAddressStartEndPosition)
	ctrl.sendData(startEndY[:4])

	ctrl.sendCommand(setRAMXAddressCounter)
	ctrl.sendData([]byte{startX})

	ctrl.sendCommand(setRAMYAddressCounter)
	ctrl.sendData(startEndY[:2])
}

type drawOpts struct {
	commands []byte
	devSize  image.Point
	buffer   *image1bit.VerticalLSB
	dstRect  image.Rectangle
	src      image.Image
	srcPts   image.Point
}

type drawSpec struct {
	// Destination on display in pixels, normalized to fit into actual size.
	dstRect image.Rectangle

	// Area to send to device; horizontally in bytes (thus aligned to
	// 8 pixels), vertically in pixels.
	memRect image.Rectangle
}

func (o *drawOpts) spec() drawSpec {
	s := drawSpec{
		dstRect: image.Rectangle{Max: o.devSize}.Intersect(o.dstRect),
	}

	s.memRect = image.Rect(
		s.dstRect.Min.X/8, s.dstRect.Min.Y,
		(s.dstRect.Max.X+7)/8, s.dstRect.Max.Y,
	)

	return s
}

// sendImage sends an image to the controller after setting up the registers.
func (o *drawOpts) sendImage(ctrl controller, cmd byte, spec *drawSpec) {
	if spec.memRect.Empty() {
		return
	}

	setMemoryArea(ctrl, spec.memRect)

	ctrl.sendCommand(cmd)

	rowData := make([]byte, spec.memRect.Dx())

	for y := spec.memRect.Min.Y; y < spec.memRect.Max.Y; y++ {
		for x := 0; x < len(rowData); x++ {
			rowData[x] = 0

			for bit := 0; bit < 8; bit++ {
				if o.buffer.BitAt(((spec.memRect.Min.X+x)*8)+bit, y) {
					rowData[x] |= 0x80 >> bit
				}
			}
		}

		ctrl.sendData(rowData)
	}
}

func drawImage(ctrl controller, opts *drawOpts) {
	s := opts.spec()

	if s.memRect.Empty() {
		return
	}

	draw.Src.Draw(opts.buffer, s.dstRect, opts.src, opts.srcPts)

	commands := opts.commands

	if len(commands) == 0 {
		commands = []byte{writeRAMBW, writeRAMRed}
	}

	// Keep the two buffers in sync.
	for _, cmd := range commands {
		opts.sendImage(ctrl, cmd, &s)
	}
}
