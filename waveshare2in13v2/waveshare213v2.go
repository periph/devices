// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

import (
	"fmt"
	"image"
	"image/color"
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
	busy gpio.PinIn

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

// New creates new handler which is used to access the display.
func New(p spi.Port, dc, cs, rst gpio.PinOut, busy gpio.PinIn, opts *Opts) (*Dev, error) {
	c, err := p.Connect(5*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return nil, err
	}

	if err := busy.In(gpio.Float, gpio.FallingEdge); err != nil {
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
	eh.waitUntilIdle()
	eh.sendCommand(swReset)
	eh.waitUntilIdle()

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

	eh.waitUntilIdle()

	return eh.err
}

func (d *Dev) initPartial() error {
	eh := errorHandler{d: *d}

	// VCOM Voltage
	eh.sendCommand(writeVcomRegister)
	eh.sendData([]byte{0x26})

	eh.waitUntilIdle()

	eh.sendCommand(writeLutRegister)
	eh.sendData(d.opts.PartialUpdate[:70])

	eh.sendCommand(0x37)
	eh.sendData([]byte{0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00})

	eh.sendCommand(displayUpdateControl2)
	eh.sendData([]byte{0xC0})

	eh.sendCommand(masterActivation)

	eh.waitUntilIdle()

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
func (d *Dev) Clear(color color.Color) error {
	eh := errorHandler{d: *d}

	clearDisplay(&eh, image.Pt(d.opts.Width, d.opts.Height),
		image1bit.BitModel.Convert(color).(image1bit.Bit))

	if eh.err == nil {
		eh.err = d.turnOnDisplay()
	}

	return eh.err
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
	opts := drawOpts{
		cmd:     writeRAMBW,
		devSize: image.Pt(d.opts.Width, d.opts.Height),
		dstRect: dstRect,
		src:     src,
		srcPts:  srcPts,
	}

	eh := errorHandler{d: *d}

	drawImage(&eh, &opts)

	if eh.err == nil {
		eh.err = d.turnOnDisplay()
	}

	return eh.err
}

// DrawPartial draws the given image to the display. Display will update only changed pixel.
func (d *Dev) DrawPartial(dstRect image.Rectangle, src image.Image, srcPts image.Point) error {
	opts := drawOpts{
		devSize: image.Pt(d.opts.Width, d.opts.Height),
		dstRect: dstRect,
		src:     src,
		srcPts:  srcPts,
	}

	eh := errorHandler{d: *d}

	for _, cmd := range []byte{writeRAMBW, writeRAMRed} {
		opts.cmd = cmd

		drawImage(&eh, &opts)

		if eh.err != nil {
			break
		}
	}

	if eh.err == nil {
		eh.err = d.turnOnDisplay()
	}

	return eh.err
}

// Halt clears the display.
func (d *Dev) Halt() error {
	return d.Clear(image1bit.On)
}

// String returns a string containing configuration information.
func (d *Dev) String() string {
	return fmt.Sprintf("epd.Dev{%s, %s, Height: %d, Width: %d}", d.c, d.dc, d.opts.Height, d.opts.Width)
}

func (d *Dev) turnOnDisplay() error {
	eh := errorHandler{d: *d}

	eh.sendCommand(displayUpdateControl2)
	eh.sendData([]byte{0xC7})
	eh.sendCommand(masterActivation)

	eh.waitUntilIdle()

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

var _ display.Drawer = &Dev{}
