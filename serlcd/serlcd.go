// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This package provides an implementation for the SparkFun SerLCD intelligent
// LCD display. This display provides hardware interfaces for SPI, I2C, and
// UART. Implements conn.display.TextDisplay
package serlcd

import (
	"errors"
	"fmt"
	"io"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/display"
)

// Representation of a SerLCD display.
type Dev struct {
	conn conn.Conn
	w    io.Writer
	cols int
	rows int
	// Display on/off, Curosr, Blink
	displayDCB byte
}

const (
	DefaultI2CAddress uint16 = 0x72
)

var settingMode byte = 0x7c
var cmdMode byte = 0xfe
var clear = []byte{settingMode, 0x2d}

func wrap(err error) error {
	return fmt.Errorf("serlcd: %w", err)
}

// Create a SerLCD display using a hardware interface that provides io.Writer.
// That can be the i2c.Bus, or a 3rd Party UART library for serial
// communications.
func NewSerLCD(writer io.Writer, rows, cols int) *Dev {
	dev := &Dev{w: writer, rows: rows, cols: cols}
	_ = dev.Display(true)
	return dev
}

// Create a SerLCD display using a hardware interface that provides
// conn.Conn. For example, a conn.spi.Conn
func NewConn(conn conn.Conn, rows, cols int) *Dev {
	dev := &Dev{conn: conn, rows: rows, cols: cols, w: nil}
	return dev
}

// Enable/Disable auto scroll
func (dev *Dev) AutoScroll(enabled bool) (err error) {
	return wrap(display.ErrNotImplemented)
}

// Return the number of columns the display supports
func (dev *Dev) Cols() int {
	return dev.cols
}

// Clear the display and move the cursor home.
func (dev *Dev) Clear() (err error) {
	_, err = dev.Write(clear)
	time.Sleep(2 * time.Millisecond)
	return
}

// Set the cursor mode. You can pass multiple arguments.
// Cursor(CursorOff, CursorUnderline)
func (dev *Dev) Cursor(mode ...display.CursorMode) (err error) {
	dev.displayDCB &= 0x04
	for _, cmd := range mode {
		switch cmd {
		case display.CursorBlink:
			dev.displayDCB |= 0x01
		case display.CursorUnderline:
			dev.displayDCB |= 0x02
		case display.CursorBlock:
			dev.displayDCB |= 0x01
		case display.CursorOff:
		default:
			err = wrap(display.ErrInvalidCommand)
			return
		}
	}
	dev.displayDCB = (dev.displayDCB | 0x08) & 0xf

	_, err = dev.Write([]byte{cmdMode, dev.displayDCB})
	return
}

// Halt shuts down the display. If the IO source implements io.Closer, it is
// called.
func (dev *Dev) Halt() (err error) {
	err = dev.Clear()
	if err != nil {
		return
	}
	err = dev.Display(false)
	if err != nil {
		return
	}
	if dev.w != nil {
		if cl, ok := dev.w.(io.Closer); ok {
			err = cl.Close()
		}
	}
	return
}

// Move the cursor home (MinRow(),MinCol())
func (dev *Dev) Home() (err error) {
	err = dev.MoveTo(dev.MinRow(), dev.MinCol())
	time.Sleep(2 * time.Millisecond)
	return
}

// Return the min column position.
func (dev *Dev) MinCol() int {
	return 0
}

// Return the min row position.
func (dev *Dev) MinRow() int {
	return 0
}

// Move the cursor forward or backward.
func (dev *Dev) Move(dir display.CursorDirection) (err error) {
	cmdByte := byte(0x10)
	switch dir {
	case display.Backward:
		// Nothing
	case display.Forward:
		cmdByte |= 0x04
	case display.Down:
		fallthrough
	case display.Up:
		fallthrough
	default:
		err = wrap(display.ErrNotImplemented)
		return
	}
	_, err = dev.Write([]byte{cmdMode, cmdByte})
	return
}

// Move the cursor to an arbitrary position.
func (dev *Dev) MoveTo(row, col int) (err error) {
	lineOffsets := []byte{0, 64, 20, 84}
	if row < dev.MinRow() || row >= dev.Rows() ||
		col < dev.MinCol() || col >= dev.Cols() {
		return errors.New("serlcd: invalid MoveTo() offset")
	}
	cmdByte := byte(0x80) + lineOffsets[row] + byte(col)
	_, err = dev.Write([]byte{cmdMode, byte(cmdByte)})
	return
}

// Return the number of rows the display supports.
func (dev *Dev) Rows() int {
	return dev.rows
}

// Turn the display on / off
func (dev *Dev) Display(on bool) (err error) {
	if on {
		dev.displayDCB |= 0x04
	} else {
		dev.displayDCB ^= 0x04
	}
	_, err = dev.Write([]byte{cmdMode, (dev.displayDCB | 0x08) & 0x0f})
	return
}

// return info about the display.
func (dev *Dev) String() string {
	ioType := "None"
	if dev.conn != nil {
		ioType = "periph.io.Conn.Conn"
	} else if dev.w != nil {
		ioType = fmt.Sprintf("%#v", dev.w)
	}
	return fmt.Sprintf("SparkFun SerLCD %dx%d Display - %s", dev.cols, dev.rows, ioType)
}

// Write a set of bytes to the display.
func (dev *Dev) Write(p []byte) (n int, err error) {
	if dev.w != nil {
		n, err = dev.w.Write(p)
		return
	}
	// Evidently, for i2c there's a buffer limitation of 32 bytes. Writing
	// more than that will lock the device up.
	writeLimit := 32
	for n < len(p) {
		bytesToWrite := len(p) - n
		if bytesToWrite > writeLimit {
			bytesToWrite = 32
		}
		w := p[n : n+bytesToWrite]
		err = dev.conn.Tx(w, nil)
		if err != nil {
			break
		}
		n = n + bytesToWrite
		time.Sleep(time.Duration(40*bytesToWrite) * time.Microsecond)
	}

	return
}

// Write a string output to the display.
func (dev *Dev) WriteString(text string) (n int, err error) {
	n, err = dev.Write([]byte(text))
	return
}

// Set the backlight intensity with 0 being off, and 255 being maximum.
func (dev *Dev) Backlight(intensity display.Intensity) error {
	return dev.RGBBacklight(intensity, intensity, intensity)
}

// Set the character contrast on the device. Writes to EEPROM, so this should
// be used sparingly. The default device Contrast is 40.
func (dev *Dev) Contrast(contrast display.Contrast) error {
	_, err := dev.Write([]byte{settingMode, 0x18, byte(contrast)})
	return err
}

// Set the backlight color with 0 being off, and 255 being maximum intensity
// for each color.
func (dev *Dev) RGBBacklight(red, green, blue display.Intensity) error {
	_, err := dev.Write([]byte{settingMode, 0x2b, byte(red & 0xff), byte(green & 0xff), byte(blue & 0xff)})
	return err
}

var _ display.TextDisplay = &Dev{}
var _ display.DisplayContrast = &Dev{}
var _ display.DisplayBacklight = &Dev{}
var _ display.DisplayRGBBacklight = &Dev{}
var _ conn.Resource = &Dev{}
