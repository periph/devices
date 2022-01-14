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
	origin   Corner
	buffer   *image1bit.VerticalLSB
	dstRect  image.Rectangle
	src      image.Image
	srcPts   image.Point
}

type drawSpec struct {
	// Amount by which buffer contents must be moved to align with the physical
	// top-left corner of the display.
	//
	// TODO: The offset shifts the buffer contents to be aligned such that the
	// translated position of the physical, on-display (0,0) location is at
	// a multiple of 8 on the equivalent to the physical X axis. With a bit of
	// additional work transfers for the TopRight and BottomLeft origins should
	// not require per-pixel processing by exploiting image1bit.VerticalLSB's
	// underlying pixel storage format.
	bufferDstOffset image.Point

	// Destination in buffer in pixels.
	bufferDstRect image.Rectangle

	// Destination in device RAM, rotated and shifted to match the origin.
	memDstRect image.Rectangle

	// Area to send to device; horizontally in bytes (thus aligned to
	// 8 pixels), vertically in pixels. Computed from memDstRect.
	memRect image.Rectangle
}

// spec pre-computes the various offsets required for sending image updates to
// the device.
func (o *drawOpts) spec() drawSpec {
	s := drawSpec{
		bufferDstRect: image.Rectangle{Max: o.devSize}.Intersect(o.dstRect),
	}

	switch o.origin {
	case TopRight:
		s.bufferDstOffset.Y = o.buffer.Bounds().Dy() - o.devSize.Y
	case BottomRight:
		s.bufferDstOffset.Y = o.buffer.Bounds().Dy() - o.devSize.Y
		s.bufferDstOffset.X = o.buffer.Bounds().Dx() - o.devSize.X
	case BottomLeft:
		s.bufferDstOffset.Y = o.buffer.Bounds().Dy() - o.devSize.Y
		s.bufferDstOffset.X = o.buffer.Bounds().Dx() - o.devSize.X
	}

	if !s.bufferDstRect.Empty() {
		switch o.origin {
		case TopLeft:
			s.memDstRect = s.bufferDstRect

		case TopRight:
			s.memDstRect.Min.X = o.devSize.Y - s.bufferDstRect.Max.Y
			s.memDstRect.Max.X = o.devSize.Y - s.bufferDstRect.Min.Y

			s.memDstRect.Min.Y = s.bufferDstRect.Min.X
			s.memDstRect.Max.Y = s.bufferDstRect.Max.X

		case BottomRight:
			s.memDstRect.Min.X = o.devSize.X - s.bufferDstRect.Max.X
			s.memDstRect.Max.X = o.devSize.X - s.bufferDstRect.Min.X

			s.memDstRect.Min.Y = o.devSize.Y - s.bufferDstRect.Max.Y
			s.memDstRect.Max.Y = o.devSize.Y - s.bufferDstRect.Min.Y

		case BottomLeft:
			s.memDstRect.Min.X = s.bufferDstRect.Min.Y
			s.memDstRect.Max.X = s.bufferDstRect.Max.Y

			s.memDstRect.Min.Y = o.devSize.X - s.bufferDstRect.Max.X
			s.memDstRect.Max.Y = o.devSize.X - s.bufferDstRect.Min.X
		}

		s.bufferDstRect = s.bufferDstRect.Add(s.bufferDstOffset)

		s.memRect.Min.X = s.memDstRect.Min.X / 8
		s.memRect.Max.X = (s.memDstRect.Max.X + 7) / 8
		s.memRect.Min.Y = s.memDstRect.Min.Y
		s.memRect.Max.Y = s.memDstRect.Max.Y
	}

	return s
}

// sendImage sends an image to the controller after setting up the registers.
func (o *drawOpts) sendImage(ctrl controller, cmd byte, spec *drawSpec) {
	if spec.memRect.Empty() {
		return
	}

	setMemoryArea(ctrl, spec.memRect)

	ctrl.sendCommand(cmd)

	var posFor func(destY, destX, bit int) image.Point

	switch o.origin {
	case TopLeft:
		posFor = func(destY, destX, bit int) image.Point {
			return image.Point{
				X: destX + bit,
				Y: destY,
			}
		}

	case TopRight:
		posFor = func(destY, destX, bit int) image.Point {
			return image.Point{
				X: destY,
				Y: o.devSize.Y - destX - bit - 1,
			}
		}

	case BottomRight:
		posFor = func(destY, destX, bit int) image.Point {
			return image.Point{
				X: o.devSize.X - destX - bit - 1,
				Y: o.devSize.Y - destY - 1,
			}
		}

	case BottomLeft:
		posFor = func(destY, destX, bit int) image.Point {
			return image.Point{
				X: o.devSize.X - destY - 1,
				Y: destX + bit,
			}
		}
	}

	rowData := make([]byte, spec.memRect.Dx())

	for destY := spec.memRect.Min.Y; destY < spec.memRect.Max.Y; destY++ {
		for destX := 0; destX < len(rowData); destX++ {
			rowData[destX] = 0

			for bit := 0; bit < 8; bit++ {
				bufPos := posFor(destY, (spec.memRect.Min.X+destX)*8, bit)
				bufPos = bufPos.Add(spec.bufferDstOffset)

				if o.buffer.BitAt(bufPos.X, bufPos.Y) {
					rowData[destX] |= 0x80 >> bit
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

	// The buffer is kept in logical orientation. Rotation and alignment with
	// the origin happens while sending the image data.
	draw.Src.Draw(opts.buffer, s.bufferDstRect, opts.src, opts.srcPts)

	commands := opts.commands

	if len(commands) == 0 {
		commands = []byte{writeRAMBW, writeRAMRed}
	}

	// Keep the two buffers in sync.
	for _, cmd := range commands {
		opts.sendImage(ctrl, cmd, &s)
	}
}
