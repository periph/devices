// Copyright 2023 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package inky

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
)

var _ display.Drawer = &DevImpression{}
var _ conn.Resource = &DevImpression{}
var _ draw.Image = &DevImpression{}

var (
	// For more: https://github.com/pimoroni/inky/issues/115#issuecomment-887453065
	dsc = []color.NRGBA{
		{0, 0, 0, 0},         // Black
		{255, 255, 255, 255}, // White
		{0, 255, 0, 255},     // Green
		{0, 0, 255, 255},     // Blue
		{255, 0, 0, 255},     // Red
		{255, 255, 0, 255},   // Yellow
		{255, 140, 0, 255},   // Orange
		{255, 255, 255, 255},
	}

	sc = []color.NRGBA{
		{57, 48, 57, 0},      // Black
		{255, 255, 255, 255}, // White
		{58, 91, 70, 255},    // Green
		{61, 59, 94, 255},    // Blue
		{156, 72, 75, 255},   // Red
		{208, 190, 71, 255},  // Yellow
		{177, 106, 73, 255},  // Orange
		{255, 255, 255, 255},
	}

	sc7 = []color.NRGBA{
		{0, 0, 0, 0},         // Black
		{217, 242, 255, 255}, // White
		{3, 124, 76, 255},    // Green
		{27, 46, 198, 255},   // Blue
		{245, 80, 34, 255},   // Red
		{255, 255, 68, 255},  // Yellow
		{239, 121, 44, 255},  // Orange
		{255, 255, 255, 255},
	}
)

const (
	ucSpeed = 3000 * physic.KiloHertz
	acSpeed = 5000 * physic.KiloHertz

	uc8159PSR   = 0x00
	uc8159PWR   = 0x01
	uc8159POF   = 0x02
	uc8159PFS   = 0x03
	uc8159PON   = 0x04
	uc8159BTST  = 0x06
	uc8159DSLP  = 0x07
	uc8159DTM1  = 0x10
	uc8159DSP   = 0x11
	uc8159DRF   = 0x12
	uc8159IPC   = 0x13
	uc8159PLL   = 0x30
	uc8159TSC   = 0x40
	uc8159TSE   = 0x41
	uc8159TSW   = 0x42
	uc8159TSR   = 0x43
	uc8159CDI   = 0x50
	uc8159LPD   = 0x51
	uc8159TCON  = 0x60
	uc8159TRES  = 0x61
	uc8159DAM   = 0x65
	uc8159REV   = 0x70
	uc8159FLG   = 0x71
	uc8159AMV   = 0x80
	uc8159VV    = 0x81
	uc8159VDCS  = 0x82
	uc8159PWS   = 0xE3
	uc8159TSSET = 0xE5

	ac073TC1PSR   = 0x00
	ac073TC1PWR   = 0x01
	ac073TC1POF   = 0x02
	ac073TC1POFS  = 0x03
	ac073TC1PON   = 0x04
	ac073TC1BTST1 = 0x05
	ac073TC1BTST2 = 0x06
	ac073TC1DSLP  = 0x07
	ac073TC1BTST3 = 0x08
	ac073TC1DTM   = 0x10
	ac073TC1DSP   = 0x11
	ac073TC1DRF   = 0x12
	ac073TC1IPC   = 0x13
	ac073TC1PLL   = 0x30
	ac073TC1TSC   = 0x40
	ac073TC1TSE   = 0x41
	ac073TC1TSW   = 0x42
	ac073TC1TSR   = 0x43
	ac073TC1CDI   = 0x50
	ac073TC1LPD   = 0x51
	ac073TC1TCON  = 0x60
	ac073TC1TRES  = 0x61
	ac073TC1DAM   = 0x65
	ac073TC1REV   = 0x70
	ac073TC1FLG   = 0x71
	ac073TC1AMV   = 0x80
	ac073TC1VV    = 0x81
	ac073TC1VDCS  = 0x82
	ac073TC1TVDCS = 0x84
	ac073TC1AGID  = 0x86
	ac073TC1CMDH  = 0xAA
	ac073TC1CCSET = 0xE0
	ac073TC1PWS   = 0xE3
	ac073TC1TSSET = 0xE6
)

// DevImpression is a handle to an Inky Impression.
type DevImpression struct {
	*Dev

	// Color Palette used to convert images to the 7 color.
	Palette color.Palette
	// Representation of the pixels.
	Pix []uint8

	// Saturation level used by the color palette.
	saturation uint
	// Resolution magic number used for resetting the panel.
	res int
}

// NewImpression opens a handle to an Inky Impression.
func NewImpression(p spi.Port, dc gpio.PinOut, reset gpio.PinOut, busy gpio.PinIn, o *Opts) (*DevImpression, error) {
	if o.ModelColor != Multi {
		return nil, fmt.Errorf("unsupported color: %v", o.ModelColor)
	}

	cSpeed := ucSpeed
	if o.Model == IMPRESSION73 {
		cSpeed = acSpeed
	}
	c, err := p.Connect(cSpeed, spi.Mode0, cs0Pin)
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
		if o.Model == IMPRESSION73 {
			maxTxSize = math.MaxInt // No chunking for AC073TC1 by default.
		} else {
			maxTxSize = 4096 // Use a conservative default.
		}
	}

	d := &DevImpression{
		Dev: &Dev{
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
		},
		saturation: 50, // Looks good enough for most of the images.
	}

	switch o.Model {
	case IMPRESSION4:
		d.width = 640
		d.height = 400
		d.res = 0b10
	case IMPRESSION57:
		d.width = 600
		d.height = 448
		d.res = 0b11
	case IMPRESSION73:
		d.width = 800
		d.height = 480
		d.res = 0b11
	}
	// Prefer the passed in values via Opts.
	if o.Width == 0 && o.Height == 0 {
		d.width = o.Width
		d.height = o.Height
	}
	d.bounds = image.Rect(0, 0, d.width, d.height)

	d.Pix = make([]uint8, d.height*d.width)

	return d, nil
}

// blend recalculates the palette based on the saturation level.
func (d *DevImpression) blend() []color.Color {
	sat := float64(d.saturation / 100)

	pr := []color.Color{}
	for i := 0; i < 7; i++ {
		var rs, gs, bs uint8
		if d.Dev.model == IMPRESSION73 {
			rs, gs, bs =
				uint8(float64(sc7[i].R)*sat),
				uint8(float64(sc7[i].G)*sat),
				uint8(float64(sc7[i].B)*sat)
		} else {
			rs, gs, bs =
				uint8(float64(sc[i].R)*sat),
				uint8(float64(sc[i].G)*sat),
				uint8(float64(sc[i].B)*sat)
		}

		rd, gd, bd :=
			uint8(float64(dsc[i].R)*(1.0-sat)),
			uint8(float64(dsc[i].G)*(1.0-sat)),
			uint8(float64(dsc[i].B)*(1.0-sat))

		pr = append(pr, color.RGBA{rs + rd, gs + gd, bs + bd, dsc[i].A})
	}
	// Add Transparent color and return the result.
	return append(pr, color.RGBA{255, 255, 255, 0})
}

// Saturation returns the current saturation level.
func (d *DevImpression) Saturation() uint {
	return d.saturation
}

// SetSaturaton changes the saturation level. This will not take effect until the next Draw().
func (d *DevImpression) SetSaturation(level uint) error {
	if level > 100 {
		return fmt.Errorf("saturation level needs to be between 0 and 100")
	}
	d.saturation = level
	// so that caller can recalculate next time they need it.
	d.Palette = nil

	return nil
}

// SetBorder changes the border color. This will not take effect until the next Draw().
func (d *DevImpression) SetBorder(c ImpressionColor) {
	d.border = Color(c)
}

// Render renders the content of the Pix to the screen.
func (d *DevImpression) Render() error {
	if d.flipVertically {
		for w := 0; w < len(d.Pix)/2-1; w = w + d.width {
			for offset := 0; offset < d.width; offset++ {
				d.Pix[w+offset], d.Pix[len(d.Pix)-d.width-w+offset] = d.Pix[len(d.Pix)-d.width-w+offset], d.Pix[w+offset]
			}
		}
	}

	if d.flipHorizontally {
		for offset := 0; offset < len(d.Pix)-1; offset = offset + d.width {
			for i, j := 0, d.width-1; i < j; i, j = i+1, j-1 {
				d.Pix[i+offset], d.Pix[j+offset] = d.Pix[j+offset], d.Pix[i+offset]
			}
		}
	}

	merged := make([]uint8, len(d.Pix)/2)
	for i, offset := 0, 0; i < len(d.Pix)-1; i, offset = i+2, offset+1 {
		merged[offset] = (d.Pix[i]<<4)&0xF0 | d.Pix[i+1]&0x0F
	}

	return d.update(merged)
}

func (d *DevImpression) cycleResetGPIO() error {
	if err := d.r.Out(gpio.Low); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return d.r.Out(gpio.High)
}

func (d *DevImpression) resetUC() error {
	if err := d.cycleResetGPIO(); err != nil {
		return err
	}
	d.wait(1 * time.Second)

	// Resolution Setting
	// 10bit horizontal followed by a 10bit vertical resolution
	tres := make([]byte, 4)
	binary.LittleEndian.PutUint16(tres[0:], uint16(d.width))
	binary.LittleEndian.PutUint16(tres[2:], uint16(d.height))

	if err := d.sendCommand(uc8159TRES, tres); err != nil {
		return err
	}

	// Panel Setting
	// 0b11000000 = Resolution select, 0b00 = 640x480, our panel is 0b11 = 600x448
	// 0b00100000 = LUT selection, 0 = ext flash, 1 = registers, we use ext flash
	// 0b00010000 = Ignore
	// 0b00001000 = Gate scan direction, 0 = down, 1 = up (default)
	// 0b00000100 = Source shift direction, 0 = left, 1 = right (default)
	// 0b00000010 = DC-DC converter, 0 = off, 1 = on
	// 0b00000001 = Soft reset, 0 = Reset, 1 = Normal (Default)
	// 0b11 = 600x448
	// 0b10 = 640x400
	if err := d.sendCommand(
		uc8159PSR,
		[]byte{
			byte(d.res<<6) | 0b101111, // See above for more magic numbers
			0x08,                      // display_colours == UC81597C
		}); err != nil {
		return err
	}

	// Power Settings
	if err := d.sendCommand(
		uc8159PWR,
		[]byte{
			(0x06 << 3) | // ??? - not documented in UC8159 datasheet
				(0x01 << 2) | // SOURCE_INTERNAL_DC_DC
				(0x01 << 1) | // GATE_INTERNAL_DC_DC
				(0x01), // LV_SOURCE_INTERNAL_DC_DC
			0x00, // VGx_20V
			0x23, // UC81597C
			0x23, // UC81597C
		}); err != nil {
		return err
	}

	// Set the PLL clock frequency to 50Hz
	// 0b11000000 = Ignore
	// 0b00111000 = M
	// 0b00000111 = N
	// PLL = 2MHz * (M / N)
	// PLL = 2MHz * (7 / 4)
	// PLL = 2,800,000 ???
	if err := d.sendCommand(uc8159PLL, []byte{0x3C}); err != nil {
		return err
	}
	// 0b00111100
	// Send the TSE register to the display
	if err := d.sendCommand(uc8159TSE, []byte{0x00}); err != nil { // Color
		return err
	}
	// VCOM and Data Interval setting
	// 0b11100000 = Vborder control (0b001 = LUTB voltage)
	// 0b00010000 = Data polarity
	// 0b00001111 = Vcom and data interval (0b0111 = 10, default)

	cdi := make([]byte, 2)
	binary.LittleEndian.PutUint16(cdi[0:], uint16(d.border<<5)|0x17) // 0b00110111
	if err := d.sendCommand(uc8159CDI, cdi); err != nil {
		return err
	}

	// Gate/Source non-overlap period
	// 0b11110000 = Source to Gate (0b0010 = 12nS, default)
	// 0b00001111 = Gate to Source
	if err := d.sendCommand(uc8159TCON, []byte{0x22}); err != nil { // 0b00100010
		return err
	}

	// Disable external flash
	if err := d.sendCommand(uc8159DAM, []byte{0x00}); err != nil {
		return err
	}

	// UC81597C
	if err := d.sendCommand(uc8159PWS, []byte{0xAA}); err != nil {
		return err
	}

	// Power off sequence
	// 0b00110000 = power off sequence of VDH and VDL, 0b00 = 1 frame (default)
	// All other bits ignored?
	if err := d.sendCommand(uc8159PFS, []byte{0x00}); err != nil { // PFS_1_FRAME
		return err
	}

	return nil
}

func (d *DevImpression) resetAC() error {
	if err := d.cycleResetGPIO(); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	if err := d.cycleResetGPIO(); err != nil {
		return err
	}
	d.wait(1 * time.Second)

	// Sending init commands to display
	if err := d.sendCommand(ac073TC1CMDH, []byte{0x49, 0x55, 0x20, 0x08, 0x09, 0x18}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1PWR, []byte{0x3F, 0x00, 0x32, 0x2A, 0x0E, 0x2A}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1PSR, []byte{0x5F, 0x69}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1POFS, []byte{0x00, 0x54, 0x00, 0x44}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1BTST1, []byte{0x40, 0x1F, 0x1F, 0x2C}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1BTST2, []byte{0x6F, 0x1F, 0x16, 0x25}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1BTST3, []byte{0x6F, 0x1F, 0x1F, 0x22}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1IPC, []byte{0x00, 0x04}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1PLL, []byte{0x02}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1TSE, []byte{0x00}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1CDI, []byte{0x3F}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1TCON, []byte{0x02, 0x00}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1TRES, []byte{0x03, 0x20, 0x01, 0xE0}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1VDCS, []byte{0x1E}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1TVDCS, []byte{0x00}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1AGID, []byte{0x00}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1PWS, []byte{0x2F}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1CCSET, []byte{0x00}); err != nil {
		return err
	}
	if err := d.sendCommand(ac073TC1TSSET, []byte{0x00}); err != nil {
		return err
	}

	return nil
}

func (d *DevImpression) update(pix []uint8) error {
	if d.model == IMPRESSION73 {
		return d.updateAC(pix)
	}
	return d.updateUC(pix)
}

func (d *DevImpression) updateUC(pix []uint8) error {
	if err := d.resetUC(); err != nil {
		return err
	}

	if err := d.sendCommand(uc8159DTM1, pix); err != nil {
		return err
	}

	if err := d.sendCommand(uc8159PON, nil); err != nil {
		return err
	}
	d.wait(200 * time.Millisecond)

	if err := d.sendCommand(uc8159DRF, nil); err != nil {
		return err
	}
	d.wait(32 * time.Second)

	if err := d.sendCommand(uc8159POF, nil); err != nil {
		return err
	}
	d.wait(200 * time.Millisecond)

	return nil
}

func (d *DevImpression) updateAC(pix []uint8) error {
	if err := d.resetAC(); err != nil {
		return err
	}

	// TODO there has to be a better way to force the white colour to be used instead of clear...
	buf := make([]byte, len(pix))
	for i := range pix {
		if pix[i]&0xF == 7 {
			buf[i] = (pix[i] & 0xF0) + 1
		}
		if pix[i]&0xF0 == 0x70 {
			buf[i] = (pix[i] & 0xF) + 0x10
		}
	}

	if err := d.sendCommand(ac073TC1DTM, buf); err != nil {
		return err
	}

	if err := d.sendCommand(ac073TC1PON, nil); err != nil {
		return err
	}
	d.wait(400 * time.Millisecond)

	if err := d.sendCommand(ac073TC1DRF, []byte{0x00}); err != nil {
		return err
	}
	d.wait(45 * time.Second) // 41 seconds in testing

	if err := d.sendCommand(ac073TC1POF, []byte{0x00}); err != nil {
		return err
	}
	d.wait(400 * time.Millisecond)

	return nil
}

// Wait for busy/wait pin.
func (d *DevImpression) wait(dur time.Duration) {
	// Set it as input, with a pull down and enable rising edge triggering.
	if err := d.busy.In(gpio.PullDown, gpio.RisingEdge); err != nil {
		log.Printf("Err: %s", err)
		return
	}
	// Wait for rising edges (Low -> High) or the timeout.
	d.busy.WaitForEdge(dur)
}

// ColorModel returns the device native color model.
func (d *DevImpression) ColorModel() color.Model {
	if d.Palette == nil {
		d.Palette = d.blend()
	}
	return d.Palette
}

// At returns the color of the pixel at (x, y).
func (d *DevImpression) At(x, y int) color.Color {
	if d.Palette == nil {
		d.Palette = d.blend()
	}
	return d.Palette[d.Pix[y*d.width+x]]
}

// Set sets the pixel at (x, y) to the given color. This will not take effect until the next Draw().
func (d *DevImpression) Set(x, y int, c color.Color) {
	if d.Palette == nil {
		d.Palette = d.blend()
	}
	d.Pix[y*d.width+x] = uint8(d.Palette.Index(c))
}

// Draw updates the display with the image.
func (d *DevImpression) Draw(r image.Rectangle, src image.Image, sp image.Point) error {
	if r != d.Bounds() {
		return fmt.Errorf("partial updates are not supported")
	}

	if src.Bounds() != d.Bounds() {
		return fmt.Errorf("image must be the same size as bounds: %v", d.Bounds())
	}

	// Dither the image using Floyd–Steinberg dithering algorithm otherwise it won't look as good on the screen.
	draw.FloydSteinberg.Draw(d, r, src, image.Point{})
	return d.Render()
}

// DrawAll redraws the whole display.
func (d *DevImpression) DrawAll(src image.Image) error {
	return d.Draw(d.Bounds(), src, image.Point{})
}
