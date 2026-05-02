// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package gc9a01

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
)

// GC9A01 command registers.
const (
	_SWRESET = 0x01
	_SLPOUT  = 0x11
	_INVOFF  = 0x20
	_INVON   = 0x21
	_DISPOFF = 0x28
	_DISPON  = 0x29
	_CASET   = 0x2A
	_RASET   = 0x2B
	_RAMWR   = 0x2C
	_MADCTL  = 0x36
	_COLMOD  = 0x3A
)

// MADCTL flags.
const (
	_MADCTL_MY  = 0x80
	_MADCTL_MX  = 0x40
	_MADCTL_MV  = 0x20
	_MADCTL_BGR = 0x08
)

const (
	_width   = 240
	_height  = 240
	_bufSize = _width * _height * 2 // RGB565: 2 bytes per pixel
)

// Rotation describes the display orientation.
type Rotation int

const (
	// Rotation0 is the default orientation.
	Rotation0 Rotation = iota
	// Rotation90 rotates 90 degrees clockwise.
	Rotation90
	// Rotation180 rotates 180 degrees.
	Rotation180
	// Rotation270 rotates 270 degrees clockwise.
	Rotation270
)

// madctlValues maps Rotation to MADCTL register values.
var madctlValues = [4]byte{
	_MADCTL_MX | _MADCTL_BGR,                            // Rotation0: 0x48
	_MADCTL_MV | _MADCTL_BGR,                            // Rotation90: 0x28
	_MADCTL_MY | _MADCTL_BGR,                            // Rotation180: 0x88
	_MADCTL_MX | _MADCTL_MY | _MADCTL_MV | _MADCTL_BGR, // Rotation270: 0xE8
}

// DefaultOpts is the recommended default options.
var DefaultOpts = Opts{}

// Opts defines the options for the device.
type Opts struct {
	// Rotation sets the display rotation. Default is Rotation0.
	Rotation Rotation
}

// Dev is an open handle to a GC9A01 display controller.
type Dev struct {
	c    conn.Conn
	dc   gpio.PinOut
	rect image.Rectangle

	buffer  []byte       // last-sent RGB565 data
	nextBuf []byte       // pre-allocated RGB565 conversion target
	next    *image.NRGBA // lazily allocated intermediate draw target
	dirty   bool         // forces full redraw on first Draw
	halted  bool
}

// New returns a Dev object that communicates over SPI to a GC9A01 display
// controller.
//
// The dc pin is the Data/Command pin for 4-wire SPI mode. The rst pin is
// optional; pass nil if not connected.
func New(p spi.Port, dc gpio.PinOut, rst gpio.PinOut, opts *Opts) (*Dev, error) {
	if dc == gpio.INVALID {
		return nil, fmt.Errorf("gc9a01: invalid dc pin")
	}
	if err := dc.Out(gpio.Low); err != nil {
		return nil, err
	}
	c, err := p.Connect(16*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return nil, err
	}
	d := &Dev{
		c:       c,
		dc:      dc,
		rect:    image.Rect(0, 0, _width, _height),
		buffer:  make([]byte, _bufSize),
		nextBuf: make([]byte, _bufSize),
		dirty:   true,
	}
	if rst != nil {
		if err := rst.Out(gpio.Low); err != nil {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
		if err := rst.Out(gpio.High); err != nil {
			return nil, err
		}
		time.Sleep(120 * time.Millisecond)
	}
	if err := d.initDisplay(opts); err != nil {
		return nil, err
	}
	return d, nil
}

// String implements display.Drawer.
func (d *Dev) String() string {
	return fmt.Sprintf("GC9A01{%s, %s, %s}", d.c, d.dc, d.rect.Max)
}

// ColorModel implements display.Drawer.
func (d *Dev) ColorModel() color.Model {
	return color.NRGBAModel
}

// Bounds implements display.Drawer. Min is guaranteed to be {0, 0}.
func (d *Dev) Bounds() image.Rectangle {
	return d.rect
}

// Draw implements display.Drawer.
//
// It draws synchronously, once this function returns, the display is updated.
// Using *image.NRGBA as source with matching bounds is the fastest path.
func (d *Dev) Draw(dstRect image.Rectangle, src image.Image, sp image.Point) error {
	var srcNRGBA *image.NRGBA
	if img, ok := src.(*image.NRGBA); ok && dstRect == d.rect && img.Bounds() == d.rect && sp.X == 0 && sp.Y == 0 {
		srcNRGBA = img
	} else {
		if d.next == nil {
			d.next = image.NewNRGBA(d.rect)
		}
		draw.Src.Draw(d.next, dstRect, src, sp)
		srcNRGBA = d.next
	}
	// Convert NRGBA to RGB565.
	pix := srcNRGBA.Pix
	for y := 0; y < _height; y++ {
		for x := 0; x < _width; x++ {
			srcOff := y*srcNRGBA.Stride + x*4
			dstOff := (y*_width + x) * 2
			d.nextBuf[dstOff], d.nextBuf[dstOff+1] = nrgbaToRGB565(pix[srcOff], pix[srcOff+1], pix[srcOff+2])
		}
	}
	return d.drawInternal()
}

// Halt turns off the display.
//
// Sending any other command afterward reenables the display.
func (d *Dev) Halt() error {
	d.halted = false
	err := d.sendCommand([]byte{_DISPOFF})
	if err == nil {
		d.halted = true
	}
	return err
}

// Invert the display colors.
func (d *Dev) Invert(on bool) error {
	if on {
		return d.sendCommand([]byte{_INVON})
	}
	return d.sendCommand([]byte{_INVOFF})
}

// nrgbaToRGB565 converts 8-bit R, G, B to RGB565 big-endian.
func nrgbaToRGB565(r, g, b byte) (byte, byte) {
	r5 := r >> 3
	g6 := g >> 2
	b5 := b >> 3
	hi := (r5 << 3) | (g6 >> 3)
	lo := (g6 << 5) | b5
	return hi, lo
}

// drawInternal compares nextBuf against buffer and sends only changed pixels.
func (d *Dev) drawInternal() error {
	startRow, endRow, startCol, endCol, skip := d.calculateDirtyRect()
	if skip {
		return nil
	}

	// Set address window.
	if err := d.setWindow(startCol, startRow, endCol-1, endRow-1); err != nil {
		return err
	}
	// Send RAMWR command.
	if err := d.sendCommand([]byte{_RAMWR}); err != nil {
		return err
	}

	// Build the pixel data for the dirty rectangle.
	w := endCol - startCol
	data := make([]byte, 0, (endRow-startRow)*w*2)
	for y := startRow; y < endRow; y++ {
		rowStart := (y*_width + startCol) * 2
		rowEnd := rowStart + w*2
		data = append(data, d.nextBuf[rowStart:rowEnd]...)
	}
	if err := d.sendData(data); err != nil {
		return err
	}

	// Update buffer with sent data.
	for y := startRow; y < endRow; y++ {
		rowStart := (y*_width + startCol) * 2
		rowEnd := rowStart + w*2
		copy(d.buffer[rowStart:rowEnd], d.nextBuf[rowStart:rowEnd])
	}
	return nil
}

// calculateDirtyRect finds the minimal bounding rectangle of changed pixels.
func (d *Dev) calculateDirtyRect() (startRow, endRow, startCol, endCol int, skip bool) {
	startRow = 0
	endRow = _height
	startCol = 0
	endCol = _width

	if d.dirty {
		d.dirty = false
		return startRow, endRow, startCol, endCol, false
	}

	rowBytes := _width * 2

	// Scan from top.
	for ; startRow < endRow; startRow++ {
		off := startRow * rowBytes
		if !bytes.Equal(d.buffer[off:off+rowBytes], d.nextBuf[off:off+rowBytes]) {
			break
		}
	}
	// Scan from bottom.
	for ; endRow > startRow; endRow-- {
		off := (endRow - 1) * rowBytes
		if !bytes.Equal(d.buffer[off:off+rowBytes], d.nextBuf[off:off+rowBytes]) {
			break
		}
	}
	if startRow == endRow {
		return 0, 0, 0, 0, true
	}

	// Scan from left (2 bytes per pixel).
	for ; startCol < endCol; startCol++ {
		changed := false
		for y := startRow; y < endRow; y++ {
			off := (y*_width + startCol) * 2
			if d.buffer[off] != d.nextBuf[off] || d.buffer[off+1] != d.nextBuf[off+1] {
				changed = true
				break
			}
		}
		if changed {
			break
		}
	}

	// Scan from right.
	for ; endCol > startCol; endCol-- {
		changed := false
		for y := startRow; y < endRow; y++ {
			off := (y*_width + endCol - 1) * 2
			if d.buffer[off] != d.nextBuf[off] || d.buffer[off+1] != d.nextBuf[off+1] {
				changed = true
				break
			}
		}
		if changed {
			break
		}
	}

	return startRow, endRow, startCol, endCol, false
}

// setWindow sets the column and row address window for subsequent RAMWR.
func (d *Dev) setWindow(x0, y0, x1, y1 int) error {
	if err := d.sendCommand([]byte{_CASET}); err != nil {
		return err
	}
	if err := d.sendData([]byte{byte(x0 >> 8), byte(x0), byte(x1 >> 8), byte(x1)}); err != nil {
		return err
	}
	if err := d.sendCommand([]byte{_RASET}); err != nil {
		return err
	}
	return d.sendData([]byte{byte(y0 >> 8), byte(y0), byte(y1 >> 8), byte(y1)})
}

func (d *Dev) sendCommand(c []byte) error {
	if d.halted {
		c = append([]byte{_DISPON}, c...)
		d.halted = false
	}
	if err := d.dc.Out(gpio.Low); err != nil {
		return err
	}
	return d.c.Tx(c, nil)
}

func (d *Dev) sendData(data []byte) error {
	if d.halted {
		if err := d.sendCommand(nil); err != nil {
			return err
		}
	}
	if err := d.dc.Out(gpio.High); err != nil {
		return err
	}
	// Chunk large data to avoid exceeding SPI driver buffer limits.
	const maxChunk = 4096
	for len(data) > 0 {
		chunk := data
		if len(chunk) > maxChunk {
			chunk = data[:maxChunk]
		}
		if err := d.c.Tx(chunk, nil); err != nil {
			return err
		}
		data = data[len(chunk):]
	}
	return nil
}

// initDisplay sends the initialization command sequence.
// The sequence is derived from the Adafruit GC9A01A Arduino driver.
func (d *Dev) initDisplay(opts *Opts) error {
	rotation := Rotation0
	if opts != nil {
		rotation = opts.Rotation
	}
	if rotation < Rotation0 || rotation > Rotation270 {
		rotation = Rotation0
	}

	type cmd struct {
		c     byte
		data  []byte
		delay time.Duration
	}

	cmds := []cmd{
		// Software reset.
		{_SWRESET, nil, 150 * time.Millisecond},

		// Undocumented vendor init registers (from Adafruit reference).
		{0xEF, nil, 0},
		{0xEB, []byte{0x14}, 0},
		{0xFE, nil, 0},
		{0xEF, nil, 0},
		{0xEB, []byte{0x14}, 0},
		{0x84, []byte{0x40}, 0},
		{0x85, []byte{0xFF}, 0},
		{0x86, []byte{0xFF}, 0},
		{0x87, []byte{0xFF}, 0},
		{0x88, []byte{0x0A}, 0},
		{0x89, []byte{0x21}, 0},
		{0x8A, []byte{0x00}, 0},
		{0x8B, []byte{0x80}, 0},
		{0x8C, []byte{0x01}, 0},
		{0x8D, []byte{0x01}, 0},
		{0x8E, []byte{0xFF}, 0},
		{0x8F, []byte{0xFF}, 0},
		{0xB6, []byte{0x00, 0x00}, 0},

		// MADCTL: memory access control (rotation + BGR).
		{_MADCTL, []byte{madctlValues[rotation]}, 0},

		// COLMOD: 16-bit color (RGB565).
		{_COLMOD, []byte{0x05}, 0},

		// More undocumented vendor registers.
		{0x90, []byte{0x08, 0x08, 0x08, 0x08}, 0},
		{0xBD, []byte{0x06}, 0},
		{0xBC, []byte{0x00}, 0},
		{0xFF, []byte{0x60, 0x01, 0x04}, 0},
		{0xC3, []byte{0x13}, 0}, // Power control 2.
		{0xC4, []byte{0x13}, 0}, // Power control 3.
		{0xC9, []byte{0x22}, 0}, // Power control 4.
		{0xBE, []byte{0x11}, 0},
		{0xE1, []byte{0x10, 0x0E}, 0},
		{0xDF, []byte{0x21, 0x0C, 0x02}, 0},

		// Gamma correction.
		{0xF0, []byte{0x45, 0x09, 0x08, 0x08, 0x26, 0x2A}, 0},
		{0xF1, []byte{0x43, 0x70, 0x72, 0x36, 0x37, 0x6F}, 0},
		{0xF2, []byte{0x45, 0x09, 0x08, 0x08, 0x26, 0x2A}, 0},
		{0xF3, []byte{0x43, 0x70, 0x72, 0x36, 0x37, 0x6F}, 0},

		{0xED, []byte{0x1B, 0x0B}, 0},
		{0xAE, []byte{0x77}, 0},
		{0xCD, []byte{0x63}, 0},
		{0x70, []byte{0x07, 0x07, 0x04, 0x0E, 0x0F, 0x09, 0x07, 0x08, 0x03}, 0},

		// Frame rate control.
		{0xE8, []byte{0x34}, 0},

		{0x62, []byte{0x18, 0x0D, 0x71, 0xED, 0x70, 0x70, 0x18, 0x0F, 0x71, 0xEF, 0x70, 0x70}, 0},
		{0x63, []byte{0x18, 0x11, 0x71, 0xF1, 0x70, 0x70, 0x18, 0x13, 0x71, 0xF3, 0x70, 0x70}, 0},
		{0x64, []byte{0x28, 0x29, 0xF1, 0x01, 0xF1, 0x00, 0x07}, 0},
		{0x66, []byte{0x3C, 0x00, 0xCD, 0x67, 0x45, 0x45, 0x10, 0x00, 0x00, 0x00}, 0},
		{0x67, []byte{0x00, 0x3C, 0x00, 0x00, 0x00, 0x01, 0x54, 0x10, 0x32, 0x98}, 0},
		{0x74, []byte{0x10, 0x85, 0x80, 0x00, 0x00, 0x4E, 0x00}, 0},
		{0x98, []byte{0x3E, 0x07}, 0},

		{0x35, nil, 0}, // Tearing effect line ON.
		// Display inversion ON — required for correct colors on most modules.
		{_INVON, nil, 0},

		// Exit sleep mode.
		{_SLPOUT, nil, 150 * time.Millisecond},
		// Display ON.
		{_DISPON, nil, 20 * time.Millisecond},
	}

	for _, c := range cmds {
		if err := d.sendCommand([]byte{c.c}); err != nil {
			return err
		}
		if len(c.data) > 0 {
			if err := d.sendData(c.data); err != nil {
				return err
			}
		}
		if c.delay > 0 {
			time.Sleep(c.delay)
		}
	}
	return nil
}

var _ display.Drawer = &Dev{}
