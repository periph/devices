// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

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
	"periph.io/x/devices/v3/ssd1306/image1bit"
	"periph.io/x/host/v3/rpi"
)

// Commands
const (
	driverOutputControl            byte = 0x01
	gateDrivingVoltageControl      byte = 0x03
	sourceDrivingVoltageControl    byte = 0x04
	dataEntryModeSetting           byte = 0x11
	swReset                        byte = 0x12
	masterActivation               byte = 0x20
	displayUpdateControl1          byte = 0x21
	displayUpdateControl2          byte = 0x22
	writeRAMBW                     byte = 0x24
	writeRAMRed                    byte = 0x26
	writeVcomRegister              byte = 0x2C
	writeLutRegister               byte = 0x32
	setDummyLinePeriod             byte = 0x3A
	setGateTime                    byte = 0x3B
	borderWaveformControl          byte = 0x3C
	setRAMXAddressStartEndPosition byte = 0x44
	setRAMYAddressStartEndPosition byte = 0x45
	setRAMXAddressCounter          byte = 0x4E
	setRAMYAddressCounter          byte = 0x4F
	setAnalogBlockControl          byte = 0x74
	setDigitalBlockControl         byte = 0x7E
)

// Register values
const (
	gateDrivingVoltage19V = 0x15

	sourceDrivingVoltageVSH1_15V   = 0x41
	sourceDrivingVoltageVSH2_5V    = 0xA8
	sourceDrivingVoltageVSL_neg15V = 0x32
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
		0x80, 0x60, 0x40, 0x00, 0x00, 0x00, 0x00, //LUT0: BB:     VS 0 ~7
		0x10, 0x60, 0x20, 0x00, 0x00, 0x00, 0x00, //LUT1: BW:     VS 0 ~7
		0x80, 0x60, 0x40, 0x00, 0x00, 0x00, 0x00, //LUT2: WB:     VS 0 ~7
		0x10, 0x60, 0x20, 0x00, 0x00, 0x00, 0x00, //LUT3: WW:     VS 0 ~7
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //LUT4: VCOM:   VS 0 ~7

		0x03, 0x03, 0x00, 0x00, 0x02, // TP0 A~D RP0
		0x09, 0x09, 0x00, 0x00, 0x02, // TP1 A~D RP1
		0x03, 0x03, 0x00, 0x00, 0x02, // TP2 A~D RP2
		0x00, 0x00, 0x00, 0x00, 0x00, // TP3 A~D RP3
		0x00, 0x00, 0x00, 0x00, 0x00, // TP4 A~D RP4
		0x00, 0x00, 0x00, 0x00, 0x00, // TP5 A~D RP5
		0x00, 0x00, 0x00, 0x00, 0x00, // TP6 A~D RP6
	},
	PartialUpdate: LUT{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //LUT0: BB:     VS 0 ~7
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //LUT1: BW:     VS 0 ~7
		0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //LUT2: WB:     VS 0 ~7
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //LUT3: WW:     VS 0 ~7
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, //LUT4: VCOM:   VS 0 ~7

		0x0A, 0x00, 0x00, 0x00, 0x00, // TP0 A~D RP0
		0x00, 0x00, 0x00, 0x00, 0x00, // TP1 A~D RP1
		0x00, 0x00, 0x00, 0x00, 0x00, // TP2 A~D RP2
		0x00, 0x00, 0x00, 0x00, 0x00, // TP3 A~D RP3
		0x00, 0x00, 0x00, 0x00, 0x00, // TP4 A~D RP4
		0x00, 0x00, 0x00, 0x00, 0x00, // TP5 A~D RP5
		0x00, 0x00, 0x00, 0x00, 0x00, // TP6 A~D RP6
	},
}

// dataDimensions returns the size in terms of bytes needed to fill the
// display.
func dataDimensions(opts *Opts) (int, int) {
	return opts.Height, (opts.Width + 7) / 8
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

func (eh *errorHandler) sendCommand(cmd byte) {
	if eh.err != nil {
		return
	}
	eh.err = eh.d.sendCommand(cmd)
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

func (d *Dev) initFull() error {
	eh := errorHandler{d: *d}

	// Software Reset
	d.waitUntilIdle()
	eh.sendCommand(swReset)
	d.waitUntilIdle()

	// Set analog block control
	eh.sendCommand(setAnalogBlockControl)
	eh.sendData([]byte{0x54})

	// Set digital block control
	eh.sendCommand(setDigitalBlockControl)
	eh.sendData([]byte{0x3B})

	// Driver output control
	eh.sendCommand(driverOutputControl)
	eh.sendData([]byte{
		byte((d.opts.Height - 1) % 0xFF),
		byte((d.opts.Height - 1) / 0xFF),
		0x00,
	})

	// Border Waveform
	eh.sendCommand(borderWaveformControl)
	eh.sendData([]byte{0x03})

	// VCOM Voltage
	eh.sendCommand(writeVcomRegister)
	eh.sendData([]byte{0x55})

	eh.sendCommand(gateDrivingVoltageControl)
	eh.sendData([]byte{gateDrivingVoltage19V})

	eh.sendCommand(sourceDrivingVoltageControl)
	eh.sendData([]byte{sourceDrivingVoltageVSH1_15V, sourceDrivingVoltageVSH2_5V, sourceDrivingVoltageVSL_neg15V})

	// Dummy Line
	eh.sendCommand(setDummyLinePeriod)
	eh.sendData([]byte{0x30})

	// Gate Time
	eh.sendCommand(setGateTime)
	eh.sendData([]byte{0x0A})

	eh.sendCommand(writeLutRegister)
	eh.sendData(d.opts.FullUpdate[:70])

	d.waitUntilIdle()

	return eh.err
}

func (d *Dev) initPartial() error {
	eh := errorHandler{d: *d}

	// VCOM Voltage
	eh.sendCommand(writeVcomRegister)
	eh.sendData([]byte{0x26})

	d.waitUntilIdle()

	eh.sendCommand(writeLutRegister)
	eh.sendData(d.opts.PartialUpdate[:70])

	eh.sendCommand(0x37)
	eh.sendData([]byte{0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00})

	eh.sendCommand(displayUpdateControl2)
	eh.sendData([]byte{0xC0})

	eh.sendCommand(masterActivation)

	d.waitUntilIdle()

	// Border Waveform
	eh.sendCommand(borderWaveformControl)
	eh.sendData([]byte{0x01})

	return eh.err
}

// Init will initialize the display with the partial-update or full-update mode.
func (d *Dev) Init(partialUpdate PartialUpdate) error {
	// Hardware Reset
	if err := d.reset(); err != nil {
		return err
	}

	if partialUpdate {
		return d.initPartial()
	}

	return d.initFull()
}

// Clear clears the display.
func (d *Dev) Clear(color byte) error {
	if err := d.setMemoryArea(d.Bounds()); err != nil {
		return err
	}

	rows, cols := dataDimensions(d.opts)
	data := bytes.Repeat([]byte{color}, cols)

	if err := d.sendCommand(writeRAMBW); err != nil {
		return err
	}

	for y := 0; y < rows; y++ {
		if err := d.sendData(data); err != nil {
			return err
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

func (d *Dev) sendImage(cmd byte, dstRect image.Rectangle, src *image1bit.VerticalLSB) error {
	// TODO: Handle dstRect not matching the device bounds.

	if err := d.setMemoryArea(dstRect); err != nil {
		return err
	}

	eh := errorHandler{d: *d}
	eh.sendCommand(cmd)

	rows, cols := dataDimensions(d.opts)
	data := make([]byte, cols)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			data[x] = 0

			for bit := 0; bit < 8; bit++ {
				if src.BitAt((x*8)+bit, y) {
					data[x] |= 0x80 >> bit
				}
			}
		}

		eh.sendData(data)
	}

	return eh.err
}

// Draw draws the given image to the display.
func (d *Dev) Draw(dstRect image.Rectangle, src image.Image, srcPts image.Point) error {
	next := image1bit.NewVerticalLSB(dstRect)
	draw.Src.Draw(next, dstRect, src, srcPts)

	if err := d.sendImage(writeRAMBW, dstRect, next); err != nil {
		return err
	}

	return d.turnOnDisplay()
}

// DrawPartial draws the given image to the display. Display will update only changed pixel.
func (d *Dev) DrawPartial(dstRect image.Rectangle, src image.Image, srcPts image.Point) error {
	next := image1bit.NewVerticalLSB(dstRect)
	draw.Src.Draw(next, dstRect, src, srcPts)

	if err := d.sendImage(writeRAMBW, dstRect, next); err != nil {
		return err
	}

	if err := d.sendImage(writeRAMRed, dstRect, next); err != nil {
		return err
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

func (d *Dev) sendCommand(cmd byte) error {
	eh := errorHandler{d: *d}

	eh.dcOut(gpio.Low)
	eh.csOut(gpio.Low)
	eh.cTx([]byte{cmd}, nil)
	eh.csOut(gpio.High)

	return eh.err
}

func (d *Dev) turnOnDisplay() error {
	eh := errorHandler{d: *d}

	eh.sendCommand(displayUpdateControl2)
	eh.sendData([]byte{0xC7})
	eh.sendCommand(masterActivation)

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

func (d *Dev) setMemoryArea(area image.Rectangle) error {
	eh := errorHandler{d: *d}

	eh.sendCommand(dataEntryModeSetting)
	eh.sendData([]byte{
		// Y increment, X increment; update address counter in X direction
		0b011,
	})

	eh.sendCommand(setRAMXAddressStartEndPosition)
	eh.sendData([]byte{
		// Start
		byte(area.Min.X / 8),

		// End
		byte((area.Max.X - 1) / 8),
	})

	eh.sendCommand(setRAMYAddressStartEndPosition)
	eh.sendData([]byte{
		// Start
		byte(area.Min.Y % 0xFF),
		byte(area.Min.Y / 0xFF),

		// End
		byte((area.Max.Y - 1) % 0xFF),
		byte((area.Max.Y - 1) / 0xFF),
	})

	eh.sendCommand(setRAMXAddressCounter)
	eh.sendData([]byte{byte(area.Min.X / 8)})

	eh.sendCommand(setRAMYAddressCounter)
	eh.sendData([]byte{
		byte(area.Min.Y & 0xFF),
		byte(area.Min.Y / 0xFF),
	})

	d.waitUntilIdle()

	return eh.err
}

var _ display.Drawer = &Dev{}
