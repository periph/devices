// Copyright 2019 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package inky

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
)

var _ display.Drawer = &Dev{}
var _ conn.Resource = &Dev{}

var borderColor = map[Color]byte{
	Black:  0x00,
	Red:    0x73,
	Yellow: 0x33,
	White:  0x31,
}

const (
	cs0Pin     = "GPIO8"
	csEnabled  = gpio.Low
	csDisabled = gpio.High
)

// Dev is a handle to an Inky.
type Dev struct {
	c conn.Conn
	// Maximum number of bytes allowed to be sent as a single I/O on c.
	maxTxSize int
	// Low when sending a command, high when sending data.
	dc gpio.PinOut
	// Reset pin, active low.
	r gpio.PinOut
	// High when device is busy.
	busy gpio.PinIn
	// Size of this model's display.
	bounds image.Rectangle
	// Whether this model needs the image flipped vertically.
	flipVertically bool
	// Whether this model needs the image flipped horizontally.
	flipHorizontally bool
	// Color of device screen (red, yellow or black).
	color Color
	// Modifiable color of border.
	border Color

	// Width of the panel.
	width int
	// Height of the panel.
	height int

	// Model being used.
	model Model
	// Variant  of the panel.
	variant uint
	// PCB Variant of the panel. Represents a version string as a number (12 -> 1.2).
	pcbVariant uint
	// cs is the chip-select pin for SPI. Refer to setCSPin() for information.
	cs gpio.PinOut
}

// New opens a handle to an Inky pHAT or wHAT.
func New(p spi.Port, dc gpio.PinOut, reset gpio.PinOut, busy gpio.PinIn, o *Opts) (*Dev, error) {
	if o.ModelColor != Black && o.ModelColor != Red && o.ModelColor != Yellow {
		return nil, fmt.Errorf("unsupported color: %v", o.ModelColor)
	}

	c, err := p.Connect(488*physic.KiloHertz, spi.Mode0, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to inky over spi: %v", err)
	}

	// Get the maxTxSize from the conn if it implements the conn.Limits interface,
	// otherwise use 4096 bytes.
	maxTxSize := 0
	if limits, ok := c.(conn.Limits); ok {
		maxTxSize = limits.MaxTxSize()
	}
	if maxTxSize == 0 {
		maxTxSize = 4096 // Use a conservative default.
	}
	// If possible, grab the CS pin.
	cs := gpioreg.ByName(cs0Pin)
	if cs != nil && cs.Out(csDisabled) != nil {
		cs = nil
	}
	d := &Dev{
		c:          c,
		maxTxSize:  maxTxSize,
		dc:         dc,
		r:          reset,
		busy:       busy,
		color:      o.ModelColor,
		border:     o.BorderColor,
		model:      o.Model,
		variant:    o.DisplayVariant,
		pcbVariant: o.PCBVariant,
		cs:         cs,
	}

	switch o.Model {
	case PHAT:
		d.width = 104
		d.height = 212
		d.flipVertically = true
	case PHAT2:
		d.width = 122
		d.height = 250
		d.flipVertically = true
	case WHAT:
		d.width = 400
		d.height = 300
	}
	// Prefer the passed in values via Opts.
	if o.Width == 0 && o.Height == 0 {
		d.width = o.Width
		d.height = o.Height
	}
	d.bounds = image.Rect(0, 0, d.width, d.height)
	return d, nil
}

// SetBorder changes the border color. This will take effect on the next call to [*Dev.Draw]().
func (d *Dev) SetBorder(c Color) {
	d.border = c
}

// SetModelColor changes the model color. This will take effect on the next call to [*Dev.Draw]().
// Useful if you want to switch between two-color and three-color drawing.
func (d *Dev) SetModelColor(c Color) error {
	if c != Black && c != Red && c != Yellow {
		return fmt.Errorf("unsupported color: %v", c)
	}
	d.color = c
	return nil
}

// String implements interface [conn.Resource].
func (d *Dev) String() string {
	index := int(d.variant)
	if index < len(displayVariantMap) {
		return displayVariantMap[index]
	}
	return "Inky pHAT"
}

func (d *Dev) Height() int {
	return d.height
}

func (d *Dev) Width() int {
	return d.width
}

// SetFlipVertically flips the image horizontally.
func (d *Dev) SetFlipVertically(f bool) {
	d.flipVertically = f
}

// SetFlipHorizontally flips the image horizontally.
func (d *Dev) SetFlipHorizontally(f bool) {
	d.flipHorizontally = f
}

// Halt implements interface [conn.Resource].
func (d *Dev) Halt() error {
	return nil
}

// ColorModel implements interface [display.Drawer].
// Maps white to white, black to black and anything else as red. Red is used as
// a placeholder for the display's third color, i.e., red or yellow.
func (d *Dev) ColorModel() color.Model {
	return color.ModelFunc(func(c color.Color) color.Color {
		r, g, b, _ := c.RGBA()
		if r == 0 && g == 0 && b == 0 {
			return color.RGBA{
				R: 0,
				G: 0,
				B: 0,
				A: 255,
			}
		} else if r == 0xffff && g == 0xffff && b == 0xffff {
			return color.RGBA{
				R: 255,
				G: 255,
				B: 255,
				A: 255,
			}
		}
		return color.RGBA{
			R: 255,
			G: 0,
			B: 0,
			A: 255,
		}
	})
}

// Bounds implements interface [display.Drawer].
func (d *Dev) Bounds() image.Rectangle {
	return d.bounds
}

// Draw implements interface [display.Drawer].
func (d *Dev) Draw(dstRect image.Rectangle, src image.Image, srcPtrs image.Point) error {
	if dstRect != d.Bounds() {
		return fmt.Errorf("partial update not supported")
	}

	if src.Bounds() != d.Bounds() {
		return fmt.Errorf("image must be the same size as bounds: %v", d.Bounds())
	}

	b := src.Bounds()
	// Black/white pixels.
	white := make([]bool, b.Size().Y*b.Size().X)
	// Red/Transparent pixels.
	red := make([]bool, b.Size().Y*b.Size().X)
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			i := y*b.Size().X + x
			srcX := x
			srcY := y
			if d.flipVertically {
				srcY = b.Max.Y - y - 1
			}
			r, g, b, _ := d.ColorModel().Convert(src.At(srcX, srcY)).RGBA()
			if r >= 0x8000 && g >= 0x8000 && b >= 0x8000 {
				white[i] = true
			} else if r >= 0x8000 {
				// Red pixels also need white behind them.
				white[i] = true
				red[i] = true
			}
		}
	}

	bufA, _ := pack(white)
	bufB, _ := pack(red)
	return d.update(borderColor[d.border], bufA, bufB)
}

// DrawAll redraws the whole display.
func (d *Dev) DrawAll(src image.Image) error {
	return d.Draw(d.Bounds(), src, image.Point{})
}

func (d *Dev) update(border byte, black []byte, red []byte) error {
	if err := d.reset(); err != nil {
		return err
	}

	r := [3]byte{}
	binary.LittleEndian.PutUint16(r[:], uint16(d.Bounds().Size().Y))
	h := [4]byte{}
	binary.LittleEndian.PutUint16(h[2:], uint16(d.Bounds().Size().Y))

	type cmdData struct {
		cmd  byte
		data []byte
	}
	cmds := []cmdData{
		{0x01, r[:]},                     // Gate setting
		{0x74, []byte{0x54}},             // Set Analog Block Control.
		{0x7e, []byte{0x3b}},             // Set Digital Block Control.
		{0x03, []byte{0x17}},             // Gate Driving Voltage.
		{0x04, []byte{0x41, 0xac, 0x32}}, // Gate Driving Voltage.
		{0x3a, []byte{0x07}},             // Dummy line period
		{0x3b, []byte{0x04}},             // Gate line width
		{0x11, []byte{0x03}},             // Data entry mode setting 0x03 = X/Y increment
		{0x2c, []byte{0x3c}},             // VCOM Register, 0x3c = -1.5v?
		{0x3c, []byte{0x00}},
		{0x3c, []byte{byte(border)}}, // Border colour
		{0x32, modelLUT[d.color]},    // Set LUTs.
		{0x44, []byte{0x00, byte(d.Bounds().Size().X/8) - 1}}, // Set RAM Y Start/End
		{0x45, h[:]},               // Set RAM X Start/End
		{0x4e, []byte{0x00}},       // Set RAM X Pointer Start
		{0x4f, []byte{0x00, 0x00}}, // Set RAM Y Pointer Start
		{0x24, black},
		{0x4e, []byte{0x00}},       // Set RAM X Pointer Start
		{0x4f, []byte{0x00, 0x00}}, // Set RAM Y Pointer Start
		{0x26, red},
	}
	if d.color == Yellow {
		cmds = append(cmds, cmdData{0x04, []byte{0x07, 0xac, 0x32}}) // Set voltage of VSH and VSL
	}
	cmds = append(cmds, cmdData{0x22, []byte{0xc7}}) // Update the image.

	for _, c := range cmds {
		if err := d.sendCommand(c.cmd, c.data); err != nil {
			return err
		}
	}

	if err := d.busy.In(gpio.PullUp, gpio.FallingEdge); err != nil {
		return err
	}
	var err error
	if err = d.sendCommand(0x20, nil); err == nil {
		d.busy.WaitForEdge(-1)
		// Enter deep sleep.
		err = d.sendCommand(0x10, []byte{0x01})
	}
	if err2 := d.busy.In(gpio.PullUp, gpio.NoEdge); err2 != nil {
		err = err2
	}
	return err
}

func (d *Dev) reset() (err error) {
	if err = d.r.Out(gpio.Low); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	if err = d.r.Out(gpio.High); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)

	if err = d.busy.In(gpio.PullUp, gpio.FallingEdge); err != nil {
		return err
	}
	defer func() {
		if err2 := d.busy.In(gpio.PullUp, gpio.NoEdge); err2 != nil {
			err = err2
		}
	}()
	if err := d.sendCommand(0x12, nil); err != nil { // Soft Reset
		return fmt.Errorf("inky: failed to reset inky: %v", err)
	}
	d.busy.WaitForEdge(-1)
	return
}

// setCSPin sets the ChipSelect pin to the desired mode. The Pimoroni driver
// uses manual control over the CS pin. To do this, they require the
// Raspberry Pi /boot/firmware/config.txt to have dtloverlay=spi0-0cs set.
//
// So, if we run with automatic CS handling, we won't be compatible with the
// pimoroni samples. If we run with manual control required, we then require
// the dtoverlay setting. We really don't want to be incompatible with the
// Pimoroni driver because that will confuse people. If the CS Pin is
// not in use, use manual control, and if it is used by the SPI driver, let
// it handle it.
func (d *Dev) setCSPin(mode gpio.Level) error {
	if d.cs != nil {
		return d.cs.Out(mode)
	}
	return nil
}

func (d *Dev) sendCommand(command byte, data []byte) (err error) {
	err = d.setCSPin(csEnabled)
	if err != nil {
		return
	}

	if err = d.dc.Out(gpio.Low); err != nil {
		return
	}
	if err = d.c.Tx([]byte{command}, nil); err != nil {
		err = fmt.Errorf("inky: failed to send command %x to inky: %v", command, err)
		return
	}
	err = d.setCSPin(csDisabled)
	if err != nil {
		return
	}

	if data != nil {
		if err = d.sendData(data); err != nil {
			err = fmt.Errorf("inky: failed to send data for command %x to inky: %v", command, err)
			return
		}
	}
	return
}

func (d *Dev) sendData(data []byte) (err error) {
	err = d.setCSPin(csEnabled)
	if err != nil {
		return
	}
	if err = d.dc.Out(gpio.High); err != nil {
		return err
	}

	for len(data) != 0 {
		var chunk []byte
		if len(data) > d.maxTxSize {
			chunk, data = data[:d.maxTxSize], data[d.maxTxSize:]
		} else {
			chunk, data = data, nil
		}
		if err = d.c.Tx(chunk, nil); err != nil {
			err = fmt.Errorf("inky: failed to send data to inky: %v", err)
			return
		}
	}
	err = d.setCSPin(csDisabled)
	return
}

func pack(bits []bool) ([]byte, error) {
	if len(bits)%8 != 0 {
		return nil, fmt.Errorf("len(bits) must be multiple of 8 but is %d", len(bits))
	}

	ret := make([]byte, len(bits)/8)
	for i, b := range bits {
		if b {
			ret[i/8] |= 1 << (7 - uint(i)%8)
		}
	}
	return ret, nil
}
