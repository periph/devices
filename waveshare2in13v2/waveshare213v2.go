// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

import (
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
	"periph.io/x/devices/v3/ssd1306/image1bit"
	"periph.io/x/host/v3/rpi"
)

// Dev defines the handler which is used to access the display.
type Dev struct {
	c conn.Conn

	dc   gpio.PinOut
	cs   gpio.PinOut
	rst  gpio.PinOut
	busy gpio.PinIO

	opts *Opts
}

// LUT contains the waveform that is used to program the display.
type LUT []byte

// Opts definies the structure of the display configuration.
type Opts struct {
	Width         int
	Height        int
	FullUpdate    LUT
	PartialUpdate LUT
}

// PartialUpdate defines if the display should do a full update or just a partial update.
type PartialUpdate bool

// errorHandler is a wrapper for error management.
type errorHandler struct {
	d   Dev
	err error
}

const (
	// Full should update the complete display.
	Full PartialUpdate = false
	// Partial should update only partial parts of the display.
	Partial PartialUpdate = true
)

// EPD2in13v2 cointains display configuration for the Waveshare 2in13v2.
var EPD2in13v2 = Opts{
	Width:  122,
	Height: 250,
	FullUpdate: LUT{
		0x80, 0x60, 0x40, 0x00, 0x00, 0x00, 0x00,
		0x10, 0x60, 0x20, 0x00, 0x00, 0x00, 0x00,
		0x80, 0x60, 0x40, 0x00, 0x00, 0x00, 0x00,
		0x10, 0x60, 0x20, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

		0x03, 0x03, 0x00, 0x00, 0x02,
		0x09, 0x09, 0x00, 0x00, 0x02,
		0x03, 0x03, 0x00, 0x00, 0x02,
		0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,

		0x15, 0x41, 0xA8, 0x32, 0x30, 0x0A,
	},
	PartialUpdate: LUT{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

		0x0A, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,

		0x15, 0x41, 0xA8, 0x32, 0x30, 0x0A,
	},
}

func (eh *errorHandler) rstOut(l gpio.Level) {
	if eh.err != nil {
		return
	}
	eh.err = eh.d.rst.Out(l)
}

func (eh *errorHandler) cTx(w []byte, r []byte) {
	if eh.err != nil {
		return
	}
	eh.err = eh.d.c.Tx(w, r)
}

func (eh *errorHandler) dcOut(l gpio.Level) {
	if eh.err != nil {
		return
	}
	eh.err = eh.d.dc.Out(l)
}

func (eh *errorHandler) csOut(l gpio.Level) {
	if eh.err != nil {
		return
	}
	eh.err = eh.d.cs.Out(l)
}

func (eh *errorHandler) sendCommand(c []byte) {
	if eh.err != nil {
		return
	}
	eh.err = eh.d.sendCommand(c)
}

func (eh *errorHandler) sendData(d []byte) {
	if eh.err != nil {
		return
	}
	eh.err = eh.d.sendData(d)
}

// New creates new handler which is used to access the display.
func New(p spi.Port, dc, cs, rst gpio.PinOut, busy gpio.PinIO, opts *Opts) (*Dev, error) {
	c, err := p.Connect(5*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return nil, err
	}

	d := &Dev{
		c:    c,
		dc:   dc,
		cs:   cs,
		rst:  rst,
		busy: busy,
		opts: opts,
	}

	return d, nil
}

// NewHat creates new handler which is used to access the display. Default Waveshare Hat configuration is used.
func NewHat(p spi.Port, opts *Opts) (*Dev, error) {
	dc := rpi.P1_22
	cs := rpi.P1_24
	rst := rpi.P1_11
	busy := rpi.P1_18
	return New(p, dc, cs, rst, busy, opts)
}

// Init will initialize the display with the partial-update or full-update mode.
func (d *Dev) Init(partialUpdate PartialUpdate) error {

	eh := errorHandler{d: *d}

	// Hardware Reset
	if err := d.reset(); err != nil {
		return err
	}

	if partialUpdate {
		// Partital Update Mode

		// VCOM Voltage
		eh.sendCommand([]byte{0x2C})
		eh.sendData([]byte{0x26})

		d.waitUntilIdle()

		eh.sendCommand([]byte{0x32})
		for i := range [70]int{} {
			eh.sendData([]byte{d.opts.PartialUpdate[i]})
		}

		eh.sendCommand([]byte{0x37})
		eh.sendData([]byte{0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00})

		eh.sendCommand([]byte{0x22})
		eh.sendData([]byte{0xC0})

		eh.sendCommand([]byte{0x20})

		d.waitUntilIdle()

		// Border Waveform
		eh.sendCommand([]byte{0x3C})
		eh.sendData([]byte{0x01})

	} else {
		// Full Update Mode

		// Software Reset
		d.waitUntilIdle()
		eh.sendCommand([]byte{0x12})
		d.waitUntilIdle()

		// Set analog block control
		eh.sendCommand([]byte{0x74})
		eh.sendData([]byte{0x54})

		// Set digital block control
		eh.sendCommand([]byte{0x7E})
		eh.sendData([]byte{0x3B})

		// Driver output control
		eh.sendCommand([]byte{0x01})
		eh.sendData([]byte{0xF9, 0x00, 0x00})

		// Data entry mode
		eh.sendCommand([]byte{0x11})
		eh.sendData([]byte{0x01})

		// Set Ram-X address start/end position
		eh.sendCommand([]byte{0x44})
		eh.sendData([]byte{0x00, 0x0F})

		// Set Ram-Y address start/end position
		eh.sendCommand([]byte{0x45})
		eh.sendData([]byte{0xF9, 0x00, 0x00, 0x00}) //0xF9-->(249+1)=250

		// Border Waveform
		eh.sendCommand([]byte{0x3C})
		eh.sendData([]byte{0x03})

		// VCOM Voltage
		eh.sendCommand([]byte{0x2C})
		eh.sendData([]byte{0x55})

		eh.sendCommand([]byte{0x03})
		eh.sendData([]byte{d.opts.FullUpdate[70]})

		eh.sendCommand([]byte{0x04})
		eh.sendData([]byte{d.opts.FullUpdate[71], d.opts.FullUpdate[72], d.opts.FullUpdate[73]})

		// Dummy Line
		eh.sendCommand([]byte{0x3A})
		eh.sendData([]byte{d.opts.FullUpdate[74]})

		// Gate Time
		eh.sendCommand([]byte{0x3B})
		eh.sendData([]byte{d.opts.FullUpdate[75]})

		eh.sendCommand([]byte{0x32})
		for i := range [70]int{} {
			eh.sendData([]byte{d.opts.FullUpdate[i]})
		}

		// Set RAM x address count to 0
		eh.sendCommand([]byte{0x4E})
		eh.sendData([]byte{0x00})

		// Set RAM y address count to 0X127
		eh.sendCommand([]byte{0x4F})
		eh.sendData([]byte{0xF9, 0x00})

		d.waitUntilIdle()
	}

	return eh.err
}

// Clear clears the display.
func (d *Dev) Clear(color byte) error {
	linewidth := 0
	if d.opts.Width%8 == 0 {
		linewidth = d.opts.Width / 8
	} else {
		linewidth = d.opts.Width/8 + 1
	}
	if err := d.sendCommand([]byte{0x24}); err != nil {
		return err
	}

	for i := 0; i <= d.opts.Height; i++ {
		for i := 0; i <= linewidth; i++ {
			if err := d.sendData([]byte{color}); err != nil {
				return err
			}
		}
	}

	return d.turnOnDisplay()
}

// ColorModel returns a 1Bit color model.
func (d *Dev) ColorModel() color.Model {
	return image1bit.BitModel
}

// Bounds returns the bounds for the configurated display.
func (d *Dev) Bounds() image.Rectangle {
	return image.Rect(0, 0, d.opts.Width, d.opts.Height)
}

// Draw draws the given image to the display.
func (d *Dev) Draw(dstRect image.Rectangle, src image.Image, srcPts image.Point) error {
	next := image1bit.NewVerticalLSB(dstRect)
	draw.Src.Draw(next, dstRect, src, srcPts)

	var byteToSend byte = 0x00
	for y := 0; y <= d.opts.Height; y++ {
		if err := d.setMemoryPointer(0, y); err != nil {
			return err
		}
		if err := d.sendCommand([]byte{0x24}); err != nil {
			return err
		}
		for x := 0; x <= d.opts.Width; x++ {
			bit := next.BitAt(x, y)
			if bit {
				byteToSend |= 0x80 >> (uint32(x) % 8)
			}
			if x%8 == 7 {
				if err := d.sendData([]byte{byteToSend}); err != nil {
					return err
				}
				byteToSend = 0x00
			}
		}
	}
	return d.turnOnDisplay()
}

// DrawPartial draws the given image to the display. Display will update only changed pixel.
func (d *Dev) DrawPartial(dstRect image.Rectangle, src image.Image, srcPts image.Point) error {
	next := image1bit.NewVerticalLSB(dstRect)
	draw.Src.Draw(next, dstRect, src, srcPts)

	var byteToSend byte = 0x00
	for y := 0; y < d.opts.Height; y++ {
		if err := d.setMemoryPointer(0, y); err != nil {
			return err
		}
		if err := d.sendCommand([]byte{0x24}); err != nil {
			return err
		}
		for x := 0; x < d.opts.Width; x++ {
			bit := next.BitAt(x, y)
			if bit {
				byteToSend |= 0x80 >> (uint32(x) % 8)
			}
			if x%8 == 7 {
				if err := d.sendData([]byte{byteToSend}); err != nil {
					return err
				}
				byteToSend = 0x00
			}
		}
	}

	byteToSend = 0x00
	for y := 0; y < d.opts.Height; y++ {
		if err := d.setMemoryPointer(0, y); err != nil {
			return err
		}
		if err := d.sendCommand([]byte{0x26}); err != nil {
			return err
		}
		for x := 0; x < d.opts.Width; x++ {
			bit := next.BitAt(x, y)
			if bit {
				byteToSend |= 0x80 >> (uint32(x) % 8)
			}
			if x%8 == 7 {
				if err := d.sendData([]byte{^byteToSend}); err != nil {
					return err
				}
				byteToSend = 0x00
			}
		}
	}

	return d.turnOnDisplay()
}

// Halt clears the display.
func (d *Dev) Halt() error {
	return d.Clear(0xFF)
}

// String returns a string containing configuration information.
func (d *Dev) String() string {
	return fmt.Sprintf("epd.Dev{%s, %s, Height: %d, Width: %d}", d.c, d.dc, d.opts.Height, d.opts.Width)
}

func (d *Dev) sendData(c []byte) error {
	eh := errorHandler{d: *d}

	eh.dcOut(gpio.High)
	eh.csOut(gpio.Low)
	eh.cTx(c, nil)
	eh.csOut(gpio.High)

	return eh.err
}

func (d *Dev) sendCommand(c []byte) error {
	eh := errorHandler{d: *d}

	eh.dcOut(gpio.Low)
	eh.csOut(gpio.Low)
	eh.cTx(c, nil)
	eh.csOut(gpio.High)

	return eh.err
}

func (d *Dev) turnOnDisplay() error {
	eh := errorHandler{d: *d}

	eh.sendCommand([]byte{0x22})
	eh.sendData([]byte{0xC7})
	eh.sendCommand([]byte{0x20})

	d.waitUntilIdle()

	return eh.err
}

// Reset the hardware
func (d *Dev) reset() error {
	eh := errorHandler{d: *d}

	eh.rstOut(gpio.High)
	time.Sleep(200 * time.Millisecond)
	eh.rstOut(gpio.Low)
	time.Sleep(200 * time.Millisecond)
	eh.rstOut(gpio.High)
	time.Sleep(200 * time.Millisecond)

	return eh.err
}

func (d *Dev) waitUntilIdle() {
	for d.busy.Read() == gpio.High {
		time.Sleep(100 * time.Millisecond)
	}
}

func (d *Dev) setMemoryPointer(x, y int) error {
	eh := errorHandler{d: *d}

	eh.sendCommand([]byte{0x4E})
	eh.sendData([]byte{byte((x >> 3) & 0xFF)})
	eh.sendCommand([]byte{0x4F})
	eh.sendData([]byte{byte(y & 0xFF)})
	eh.sendData([]byte{byte((y >> 8) & 0xFF)})

	d.waitUntilIdle()

	return eh.err
}

var _ display.Drawer = &Dev{}
