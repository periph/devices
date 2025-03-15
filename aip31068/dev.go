// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// The aip31068 is an HD44780 compatible I²C driver chip. It provides an I²C
// interface to an LCD. This is not a _backpack_ chip in the sense that it
// provides GPIO pins via an I²C interface. The I²C write commands go directly
// to the LCD display driver.
//
// Implements periph.io/x/conn/display/TextDisplay
//
// # Datasheet
//
// https://support.newhavendisplay.com/hc/en-us/article_attachments/4414498095511
package aip31068

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/i2c"
)

const (
	busyFlag     byte = 0x80
	cmdByte      byte = 0xfe
	dataByte     byte = 0x40
	moreControls byte = 0x80
	packageName       = "aip31068"
)

var (
	ErrNotImplemented = fmt.Errorf("%s: %w", packageName, display.ErrNotImplemented)

	rowConstants      = [][]byte{{0, 0, 64}, {0, 0, 64, 20, 84}}
	clearScreen       = []byte{cmdByte, 0x01}
	goHome            = []byte{cmdByte, 0x02}
	setCursorPosition = []byte{cmdByte, 0x80}
	displayMode       = []byte{cmdByte, 0x20}
	defaultEntryMode  = []byte{cmdByte, 0x06}
)

type Dev struct {
	rows int
	cols int

	mu     sync.Mutex
	d      *i2c.Dev
	blink  bool
	on     bool
	cursor bool
	blMono display.DisplayBacklight
	blRGB  display.DisplayRGBBacklight
}

func wrap(err error) error {
	if err == nil || strings.HasPrefix(err.Error(), packageName) {
		return err
	}
	return fmt.Errorf("%s: %w", packageName, err)
}

// New creates an aip31068 based LCD.
//
// backlight is a controller that manipulates the display backlight. If the
// display backlight is hard-wired on, then this can be nil. Otherwise, it
// should implement either display.DisplayBacklight or
// display.DisplayRGBBacklight.
func New(bus i2c.Bus,
	address uint16,
	backlight any,
	rows,
	cols int) (*Dev, error) {

	dev := &Dev{
		d:    &i2c.Dev{Bus: bus, Addr: address},
		rows: rows,
		cols: cols,
	}
	switch bl := backlight.(type) {
	case display.DisplayBacklight:
		dev.blMono = bl
	case display.DisplayRGBBacklight:
		dev.blRGB = bl
	}

	err := dev.init()
	if err != nil {
		dev = nil
	}
	return dev, wrap(err)
}

// Perform the display initialization routine,
func (dev *Dev) init() error {
	// Set the lines display value
	var modeToSet = []byte{cmdByte, displayMode[1]}
	if dev.rows > 1 {
		modeToSet[1] = modeToSet[1] | 0x08
	}
	_, err := dev.Write(modeToSet)
	if err == nil {
		err = dev.Display(true)
		time.Sleep(40 * time.Microsecond)
	}
	if err == nil {
		err = dev.Clear()
		time.Sleep(2000 * time.Microsecond)
	}

	if err == nil {
		err = dev.Home()
		time.Sleep(40 * time.Microsecond)
	}

	if err == nil {
		// Set the entry mode
		_, err = dev.Write(defaultEntryMode)
	}
	if err == nil {
		_ = dev.Backlight(0xff)
	}
	if err != nil {
		err = wrap(err)
	}
	return err
}

// Return the row offset value
func getRowConstant(row, maxcols int) byte {
	var offset int
	if maxcols != 16 {
		offset = 1
	}
	return rowConstants[offset][row]
}

// Enable/Disable auto scroll
func (dev *Dev) AutoScroll(enabled bool) error {
	return ErrNotImplemented
}

// Return the number of columns the display supports
func (dev *Dev) Cols() int {
	return dev.cols
}

// Clear the display and move the cursor home.
func (dev *Dev) Clear() error {
	_, err := dev.Write(clearScreen)
	if err != nil {
		err = wrap(err)
	}
	return err
}

// Set the cursor mode. You can pass multiple arguments.
// Cursor(CursorOff, CursorUnderline)
func (dev *Dev) Cursor(modes ...display.CursorMode) (err error) {
	var val = byte(0x08)
	if dev.on {
		val |= 0x04
	}
	for _, mode := range modes {
		switch mode {
		case display.CursorOff:
			// dev.Write(underlineCursorOff)
			dev.blink = false
			dev.cursor = false
		case display.CursorBlink:
			dev.blink = true
			dev.cursor = true
			val |= 0x01
		case display.CursorUnderline:
			dev.cursor = true
			dev.blink = true
			// dev.Write(underlineCursorOn)
			val |= 0x02
		case display.CursorBlock:
			dev.cursor = true
			dev.blink = true
			val |= 0x01
		default:
			err = fmt.Errorf("Waveshare1602 - unexpected cursor: %d", mode)
			return
		}
	}
	_, err = dev.Write([]byte{cmdByte, val & 0x0f})
	return wrap(err)

}

// Turn the display on / off
func (dev *Dev) Display(on bool) error {
	dev.on = on
	val := byte(0x08)
	if on {
		val |= 0x04
	}
	if dev.blink {
		val |= 0x01
	}
	if dev.cursor {
		val |= 0x02
	}
	_, err := dev.Write([]byte{cmdByte, val})
	return err

}

// Halt clears the display, turns the backlight off, and turns the display off.
// Halt() is called for the data pins gpio.Group.
func (dev *Dev) Halt() error {
	_ = dev.Clear()
	_ = dev.Display(false)
	_ = dev.Backlight(0)
	return nil
}

// Move the cursor home (MinRow(),MinCol())
func (dev *Dev) Home() error {
	_, err := dev.Write(goHome)
	return err
}

// Return the min column position.
func (dev *Dev) MinCol() int {
	return 1
}

// Return the min row position.
func (dev *Dev) MinRow() int {
	return 1
}

// Move the cursor forward or backward.
func (dev *Dev) Move(dir display.CursorDirection) (err error) {
	var val byte = 0x10
	switch dir {
	case display.Backward:

	case display.Forward:
		val |= 0x04
	case display.Down, display.Up:
		fallthrough
	default:
		err = ErrNotImplemented
		return
	}
	_, err = dev.Write([]byte{cmdByte, val})
	err = wrap(err)
	return
}

// Move the cursor to arbitrary position.
func (dev *Dev) MoveTo(row, col int) (err error) {
	if row < dev.MinRow() || row > dev.rows || col < dev.MinCol() || col > dev.cols {
		err = fmt.Errorf("%s.MoveTo(%d,%d) value out of range", packageName, row, col)
		return
	}
	var cmd = []byte{cmdByte, setCursorPosition[1]}
	cmd[1] |= getRowConstant(row, dev.cols) + byte(col-1)
	_, err = dev.Write(cmd)
	err = wrap(err)
	return err
}

// Return the number of rows the display supports.
func (dev *Dev) Rows() int {
	return dev.rows
}

func (dev *Dev) String() string {
	return fmt.Sprintf("%s Rows: %d Cols: %d", packageName, dev.rows, dev.cols)
}

// Read the busy flag to make sure it's clear to write. It's a little wonky
// initially but then smooths out, so it makes a best effort and ignores errors.
func (dev *Dev) waitForFree() {
	tLimit := time.Now().Add(3 * time.Millisecond)
	w := make([]byte, 2)
	r := make([]byte, 1)
	for time.Now().Before(tLimit) {
		err := dev.d.Tx(w, r)
		if err == nil && (r[0]&busyFlag) == 0 {
			break
		}
		time.Sleep(100 * time.Microsecond)
	}
}

// Write a set of bytes to the display. This routine handles control
// and data characters transparently.
func (dev *Dev) Write(p []byte) (n int, err error) {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	dev.waitForFree()

	lastControl := -1
	for i := range len(p) {
		if p[i] == cmdByte {
			lastControl = i
		}
	}

	w := make([]byte, 0, len(p))

	for pos := 0; pos < len(p); {

		// So, when we're writing, we need to send a control byte first
		// that says type data, or cmd. We then send the bytes. If the
		// type changes, then we need to send a new control byte.
		//
		// If there are more control bytes, then the control byte has bit 7
		// set, and we send a control byte for each character sent.
		var controlByte byte = 0x00
		if p[pos] == cmdByte {
			pos += 1
		} else {
			controlByte |= dataByte
		}
		if pos < lastControl {
			controlByte |= moreControls
		}

		if (pos - 1) <= lastControl {
			w = append(w, controlByte)
		}

		w = append(w, p[pos])
		pos += 1
	}
	err = dev.d.Tx(w, nil)
	if err == nil {
		n = len(p)
	}
	err = wrap(err)
	return n, err
}

// Write a string output to the display.
func (dev *Dev) WriteString(text string) (n int, err error) {
	return dev.Write([]byte(text))
}

// Set the backlight intensity.
func (dev *Dev) Backlight(intensity display.Intensity) error {
	if dev.blMono != nil {
		return dev.blMono.Backlight(intensity)
	} else if dev.blRGB != nil {
		return dev.blRGB.RGBBacklight(intensity, intensity, intensity)
	}
	return ErrNotImplemented
}

// For units that have an RGB Backlight, set the backlight color/intensity.
// The range of the values is 0-255.
func (dev *Dev) RGBBacklight(red, green, blue display.Intensity) error {
	if dev.blRGB != nil {
		return dev.blRGB.RGBBacklight(red, green, blue)
	} else if dev.blMono != nil {
		return dev.blMono.Backlight(red | green | blue)
	}
	return ErrNotImplemented
}

var _ conn.Resource = &Dev{}
var _ display.TextDisplay = &Dev{}
var _ display.DisplayBacklight = &Dev{}
var _ display.DisplayRGBBacklight = &Dev{}
