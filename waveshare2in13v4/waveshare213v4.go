// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v4

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

// Commands
const (
	driverOutputControl                byte = 0x01
	gateDrivingVoltageControl          byte = 0x03
	sourceDrivingVoltageControl        byte = 0x04
	initialCodeSettingOTPProgram       byte = 0x08
	writeRegisterForInitialCodeSetting byte = 0x09
	readRegisterForInitialCodeSetting  byte = 0x0A
	boosterSoftStartControl            byte = 0x0C
	deepSleepMode                      byte = 0x10
	dataEntryModeSetting               byte = 0x11
	swReset                            byte = 0x12
	hvReadyDetection                   byte = 0x14
	vciDetection                       byte = 0x15
	tempSensorSelect                   byte = 0x18
	tempSensorRegWrite                 byte = 0x1A
	tempSensorRegRead                  byte = 0x1B
	tempSensorExtWrite                 byte = 0x1C
	masterActivation                   byte = 0x20
	displayUpdateControl1              byte = 0x21
	displayUpdateControl2              byte = 0x22
	writeRAMBW                         byte = 0x24
	writeRAMRed                        byte = 0x26
	readRAM                            byte = 0x27
	vcomSense                          byte = 0x28
	vcomDuration                       byte = 0x29
	vcomProgramOTP                     byte = 0x2A
	vcomWriteRegisterControl           byte = 0x2B
	vcomRegisterWrite                  byte = 0x2C
	otpReadRegisterDisplayOpt          byte = 0x2D
	userIDRead                         byte = 0x2E
	statusBitRead                      byte = 0x2F
	otpProgramWaveformSetting          byte = 0x30
	otpLoadWaveformSetting             byte = 0x31
	writeLutRegister                   byte = 0x32
	crcCalculation                     byte = 0x34
	crcStatusRead                      byte = 0x35
	otpProgramSelect                   byte = 0x36
	writeRegisterForDisplayOption      byte = 0x37
	writeRegisterForUserID             byte = 0x38
	otpProgramMode                     byte = 0x39
	borderWaveformControl              byte = 0x3C
	endOptionEOPT                      byte = 0x3F
	readRAMOpt                         byte = 0x41
	setRAMXAddressStartEndPosition     byte = 0x44
	setRAMYAddressStartEndPosition     byte = 0x45
	autoWriteRedRAMRegpattern          byte = 0x46
	autoWriteBWRamRegPattern           byte = 0x47
	setRAMXAddressCounter              byte = 0x4E
	setRAMYAddressCounter              byte = 0x4F
	nop                                byte = 0x7F
)

// Flags for the displayUpdateControl2 command
const (
	displayUpdateDisableClock byte = 1 << iota
	displayUpdateDisableAnalog
	displayUpdateDisplay
	displayUpdateMode2
	displayUpdateLoadLUTFromOTP
	displayUpdateLoadTemperature
	displayUpdateEnableClock
	displayUpdateEnableAnalog
)

// Dev defines the handler which is used to access the display.
type Dev struct {
	c conn.Conn

	dc   gpio.PinOut
	cs   gpio.PinOut
	rst  gpio.PinOut
	busy gpio.PinIn

	bounds image.Rectangle
	buffer *image1bit.VerticalLSB
	mode   PartialUpdate

	opts *Opts
}

// Corner describes a corner on the physical device and is used to define the
// origin for drawing operations.
type Corner uint8

const (
	TopLeft Corner = iota
	TopRight
	BottomRight
	BottomLeft
)

// LUT contains the waveform that is used to program the display.
type LUT []byte

// Opts definies the structure of the display configuration.
type Opts struct {
	Width  int
	Height int
	Origin Corner
}

// PartialUpdate defines if the display should do a full update or just a partial update.
type PartialUpdate bool

const (
	// Full should update the complete display.
	Full PartialUpdate = false
	// Partial should update only partial parts of the display.
	Partial PartialUpdate = true
)

// FastDisplay defines if the display can display in fast mode.
type FastDisplay bool

const (
	// Updates the display fast.
	Fast PartialUpdate = true
	// Updates the display at normal speed.
	Normal PartialUpdate = false
)

// flipPt returns a new image.Point with the X and Y coordinates exchanged.
func flipPt(pt image.Point) image.Point {
	return image.Point{X: pt.Y, Y: pt.X}
}

// New creates new handler which is used to access the display.
func New(p spi.Port, dc, cs, rst gpio.PinOut, busy gpio.PinIn, opts *Opts) (*Dev, error) {
	c, err := p.Connect(4*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return nil, err
	}

	if err := busy.In(gpio.Float, gpio.FallingEdge); err != nil {
		return nil, err
	}

	displaySize := image.Pt(opts.Width, opts.Height)

	// The physical X axis is sized to have one-byte alignment on the (0,0)
	// on-display position after rotation.
	bufferSize := image.Pt((opts.Width+7)/8*8, opts.Height)

	switch opts.Origin {
	case TopLeft, BottomRight:
	case TopRight, BottomLeft:
		displaySize = flipPt(displaySize)
		bufferSize = flipPt(bufferSize)
	default:
		return nil, fmt.Errorf("unknown corner %v", opts.Origin)
	}

	d := &Dev{
		c:      c,
		dc:     dc,
		cs:     cs,
		rst:    rst,
		busy:   busy,
		bounds: image.Rectangle{Max: displaySize},
		buffer: image1bit.NewVerticalLSB(image.Rectangle{
			Max: bufferSize,
		}),
		mode: Full,
		opts: opts,
	}

	// Default color
	draw.Src.Draw(d.buffer, d.buffer.Bounds(), &image.Uniform{image1bit.On}, image.Point{})

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

// func (d *Dev) configMode(ctrl controller) {

// 	configDisplayMode(ctrl, d.mode)
// }

// Init configures the display for usage through the other functions.
func (d *Dev) Init() error {
	// Hardware Reset
	if err := d.Reset(); err != nil {
		return err
	}

	eh := errorHandler{d: *d}

	initDisplay(&eh, d.opts)

	// if eh.err == nil {
	// 	d.configMode(&eh)
	// }

	return eh.err
}

// // SetUpdateMode changes the way updates to the displayed image are applied. In
// // Full mode (the default) a full refresh is done with all pixels cleared and
// // re-applied. In Partial mode only the changed pixels are updated (aligned to
// // multiples of 8 on the horizontal axis), potentially leaving behind small
// // optical artifacts due to the way e-paper displays work.
// //
// // The vendor datasheet recommends a full update at least once every 24 hours.
// // When using partial updates the Clear function can be used for the purpose,
// // followed by re-drawing.
// func (d *Dev) SetUpdateMode(mode PartialUpdate) error {
// 	d.mode = mode

// 	eh := errorHandler{d: *d}
// 	d.configMode(&eh)

// 	return eh.err
// }

// Clear clears the display.
func (d *Dev) Clear(color color.Color) error {
	return d.Draw(d.buffer.Bounds(), &image.Uniform{
		C: image1bit.BitModel.Convert(color).(image1bit.Bit),
	}, image.Point{})
}

// ColorModel returns a 1Bit color model.
func (d *Dev) ColorModel() color.Model {
	return image1bit.BitModel
}

// Bounds returns the bounds for the configurated display.
func (d *Dev) Bounds() image.Rectangle {
	return d.bounds
}

// Draw draws the given image to the display. Only the destination area is
// uploaded. Depending on the update mode the whole display or the destination
// area is refreshed.
func (d *Dev) Draw(dstRect image.Rectangle, src image.Image, srcPts image.Point) error {
	opts := drawOpts{
		devSize: d.bounds.Max,
		origin:  d.opts.Origin,
		buffer:  d.buffer,
		dstRect: dstRect,
		src:     src,
		srcPts:  srcPts,
	}

	eh := errorHandler{d: *d}

	drawImage(&eh, &opts, d.mode)

	if eh.err == nil {
		updateDisplay(&eh, d.mode)
	}

	return eh.err
}

// DrawPartial draws the given image to the display.
//
// Deprecated: Use Draw instead. DrawPartial merely forwards all calls.
func (d *Dev) DrawPartial(dstRect image.Rectangle, src image.Image, srcPts image.Point) error {
	return d.Draw(dstRect, src, srcPts)
}

// Halt clears the display.
func (d *Dev) Halt() error {
	return d.Clear(image1bit.On)
}

// String returns a string containing configuration information.
func (d *Dev) String() string {
	return fmt.Sprintf("epd.Dev{%s, %s, Width: %d, Height: %d}", d.c, d.dc, d.bounds.Dx(), d.bounds.Dy())
}

// Sleep makes the controller enter deep sleep mode. It can be woken up by
// calling Init again.
func (d *Dev) Sleep() error {
	eh := errorHandler{d: *d}

	// Turn off DC/DC converter, clock, output load and MCU. RAM content is
	// retained.
	eh.sendCommand(deepSleepMode)
	eh.sendData([]byte{0x01})

	return eh.err
}

var _ display.Drawer = &Dev{}

// refactored

// EPD2in13v4 cointains display configuration for the Waveshare 2in13v2.
var EPD2in13v4 = Opts{
	Width:  122,
	Height: 250,
}

// Reset the hardware.
func (d *Dev) Reset() error {
	eh := errorHandler{d: *d}

	eh.rstOut(gpio.High)
	time.Sleep(20 * time.Millisecond)
	eh.rstOut(gpio.Low)
	time.Sleep(2 * time.Millisecond)
	eh.rstOut(gpio.High)
	time.Sleep(20 * time.Millisecond)

	return eh.err
}
