// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package hd44780 controls the Hitachi LCD display chipset HD-44780
//
// # Datasheet
//
// https://www.sparkfun.com/datasheets/LCD/HD44780.pdf
package hd44780

import (
	"fmt"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/gpio"
)

type writeMode bool

type ifMode byte

const (
	modeCommand writeMode = false
	modeData    writeMode = true

	cmdByte byte = 0xfe

	mode4Bit ifMode = 0x04
	mode8Bit ifMode = 0x08
)

// HD44780 is an implementation that supports writing to LCD displays using a
// gpio.Group for the data pins, and discrete pins for the reset, enable, and
// and backlight pins.
//
// Implements periph.io/conn/x/display/TextDisplay and display.DisplayBacklight
type HD44780 struct {
	dataPins     gpio.Group
	resetPin     gpio.PinOut
	enablePin    gpio.PinOut
	backlightPin gpio.PinOut
	mode         ifMode
	rows         int
	cols         int
	on           bool
	cursor       bool
	blink        bool
	lastWrite    int64
}

const (
	delayCommand   time.Duration = 2000
	delayCharacter time.Duration = 200
)

var rowConstants = [][]byte{{0, 0, 64}, {0, 0, 64, 20, 84}}
var clearScreen = []byte{cmdByte, 0x01}
var goHome = []byte{cmdByte, 0x02}
var setCursorPosition = []byte{cmdByte, 0x80}

// Return the row offset value
func getRowConstant(row, maxcols int) byte {
	var offset int
	if maxcols != 16 {
		offset = 1
	}
	return rowConstants[offset][row]
}

// NewHD44780 takes a GPIO group, and gpio.PinOut for reset, enable, and
// backlight. It returns a the HD44780 device in an initialized state and
// ready for use.
//
// The first 4 or 8 pins of the data group must be connected to the data lines
// To use 4 bit mode, you would connect lines D4-D7 on the display, and for
// 8 bit mode, D0-D7. If dataPinGroup is 8 or more pins, then it's assumed the display is
// connected using all 8 pins.
func NewHD44780(
	dataPinGroup gpio.Group,
	resetPin, enablePin, backlightPin *gpio.PinOut,
	rows, cols int) (*HD44780, error) {

	mode := mode4Bit
	if len(dataPinGroup.Pins()) >= 8 {
		mode = mode8Bit
	}

	display := &HD44780{
		dataPins:     dataPinGroup,
		resetPin:     *resetPin,
		enablePin:    *enablePin,
		backlightPin: *backlightPin,
		mode:         mode,
		rows:         rows,
		cols:         cols,
		on:           true,
	}
	return display, display.init()
}

// Not supported by this device. Returns display.ErrNotImplemented
func (lcd *HD44780) AutoScroll(enabled bool) error {
	// TODO: Wrap
	return display.ErrNotImplemented
}

// Clears the screen and moves the cursor to the first position.
func (lcd *HD44780) Clear() error {
	_, err := lcd.Write(clearScreen)
	return err
}

// Return the number of columns the display supports
func (lcd *HD44780) Cols() int {
	return lcd.cols
}

// Set the cursor mode. You can pass multiple arguments.
// Cursor(CursorOff, CursorUnderline)
func (lcd *HD44780) Cursor(modes ...display.CursorMode) (err error) {
	var val = byte(0x08)
	if lcd.on {
		val |= 0x04
	}
	for _, mode := range modes {
		switch mode {
		case display.CursorOff:
			// lcd.Write(underlineCursorOff)
			lcd.blink = false
			lcd.cursor = false
		case display.CursorBlink:
			lcd.blink = true
			lcd.cursor = true
			val |= 0x01
		case display.CursorUnderline:
			lcd.cursor = true
			lcd.blink = true
			// lcd.Write(underlineCursorOn)
			val |= 0x02
		case display.CursorBlock:
			lcd.cursor = true
			lcd.blink = true
			val |= 0x01
		default:
			err = fmt.Errorf("HD44780 - unexpected cursor: %d", mode)
			return
		}
	}
	_, err = lcd.Write([]byte{cmdByte, val & 0x0f})
	return err
}

// Move the cursor home (MinRow(),MinCol())
func (lcd *HD44780) Home() (err error) {
	_, err = lcd.Write(goHome)
	return err
}

// Return the min column position.
func (lcd *HD44780) MinCol() int {
	return 1
}

// Return the min row position.
func (lcd *HD44780) MinRow() int {
	return 1
}

// Move the cursor forward or backward.
func (lcd *HD44780) Move(dir display.CursorDirection) (err error) {
	var val byte = 0x10
	switch dir {
	case display.Backward:
	case display.Forward:
		val |= 0x04
	case display.Down, display.Up:
		fallthrough
	default:
		err = fmt.Errorf("hd44780: %w", display.ErrNotImplemented)
		return
	}
	_, err = lcd.Write([]byte{cmdByte, val})
	return
}

// Move the cursor to arbitrary position.
func (lcd *HD44780) MoveTo(row, col int) (err error) {
	if row < lcd.MinRow() || row > lcd.rows || col < lcd.MinCol() || col > lcd.cols {
		err = fmt.Errorf("HD44780.MoveTo(%d,%d) value out of range.", row, col)
		return
	}
	var cmd = []byte{cmdByte, setCursorPosition[1]}
	cmd[1] |= getRowConstant(row, lcd.cols) + byte(col-1)
	_, err = lcd.Write(cmd)
	return
}

// Return the number of rows the display supports.
func (lcd *HD44780) Rows() int {
	return lcd.rows
}

// Return info about the dsiplay.
func (lcd *HD44780) String() string {
	return fmt.Sprintf("HD44780::%s - Rows: %d, Cols: %d", lcd.dataPins.String(), lcd.rows, lcd.cols)
}

// Turn the display on / off
func (lcd *HD44780) Display(on bool) error {
	lcd.on = on
	val := byte(0x08)
	if on {
		val |= 0x04
	}
	if lcd.blink {
		val |= 0x01
	}
	if lcd.cursor {
		val |= 0x02
	}
	_, err := lcd.Write([]byte{cmdByte, val})
	return err

}

// Write a set of bytes to the display.
func (lcd *HD44780) Write(p []byte) (n int, err error) {

	if len(p) == 0 {
		return
	}
	if p[0] == cmdByte {
		n = len(p) - 1
		err = lcd.sendCommand(p[1:])
		return
	}
	lcd.delayWrite(delayCommand)
	err = lcd.resetPin.Out(gpio.Level(modeData))
	if err != nil {
		return
	}

	for _, byteVal := range p {
		lcd.lastWrite = time.Now().UnixMicro()
		if lcd.mode == mode4Bit {
			err = lcd.write4Bits(byteVal >> 4)
			if err == nil {
				err = lcd.write4Bits(byteVal & 0x0f)
			}
		} else {
			err = lcd.write8Bits(byteVal)
		}
		if err != nil {
			return
		}
		n += 1
		time.Sleep(delayCharacter * time.Microsecond)
	}
	lcd.lastWrite = time.Now().UnixMicro()
	return
}

// Write a string output to the display.
func (lcd *HD44780) WriteString(text string) (int, error) {
	return lcd.Write([]byte(text))
}

// Halt clears the display, turns the backlight off, and turns the display off.
// Halt() is called for the data pins gpio.Group.
func (lcd *HD44780) Halt() error {
	_ = lcd.Clear()
	_ = lcd.Backlight(0)
	_ = lcd.Display(false)
	return lcd.dataPins.Halt()
}

// Turn the display's backlight on or off. You must supply a backlight control
// pin when creating the display to use this.
func (lcd *HD44780) Backlight(intensity display.Intensity) error {
	on := (intensity > 0)
	err := lcd.Display(on)
	if err != nil {
		return err
	}
	if lcd.backlightPin != nil {
		err = lcd.backlightPin.Out(gpio.Level(on))
	}
	return err
}

// delayWrite looks at the time of the last LCD write and if
// the specified microseconds period has not elapsed, it
// invokes time.Sleep() with the difference.
//
// Some I/O methods, like direct GPIO on a Pi are very fast, while other methods
// like i2c take longer. Without delays, on very fast I/O paths, the LCD will
// display garbage. The correct way to handle this would be to read the Busy flag
// on the LCD display. However, some backpacks don't have the capability to
// check the Busy flag because the R/W pin isn't connected. So, we can't correctly
// handle io delays. This handles the very fast interfaces, while not
// penalizing the slower ones with unnecessary delays.
//
// The value of lcd.lastWrite is updated to the current time by the call.
func (lcd *HD44780) delayWrite(microseconds time.Duration) {
	diff := microseconds - time.Duration(time.Now().UnixMicro()-lcd.lastWrite)
	if diff > 0 {
		time.Sleep(time.Duration(diff) * time.Microsecond)
	}
	lcd.lastWrite = time.Now().UnixMicro()
}

// Init the display. The HD44780 has a fairly complex initialization cycle
// with variations for 4 and 8 pin mode.
func (lcd *HD44780) init() error {
	/*
	   This is the startup sequence for the Hitachi HD44780U chip as
	   documented in the Datasheet.
	*/
	lcd.lastWrite = time.Now().UnixMicro()
	if lcd.mode == mode4Bit {
		var lineMode byte = 0x20
		if lcd.rows > 1 {
			lineMode |= 0x08
		}
		err := lcd.resetPin.Out(gpio.Level(modeCommand))
		if err != nil {
			return err
		}
		err = lcd.enablePin.Out(gpio.Low)
		if err != nil {
			return err
		}
		err = lcd.write4Bits(0x03)
		if err != nil {
			return err
		}
		time.Sleep(4100 * time.Microsecond)
		_ = lcd.write4Bits(0x03)
		_ = lcd.write4Bits(0x03)
		_ = lcd.write4Bits(0x02)
		_ = lcd.sendCommand([]byte{lineMode})
	} else {
		// Init the display for 8 pin operation.
		lineMode := byte(0x30) // Set the line mode and interface to 8 bits
		if lcd.rows > 1 {
			lineMode |= 0x08
		}
		err := lcd.resetPin.Out(gpio.Level(modeCommand))
		if err != nil {
			return err
		}
		err = lcd.enablePin.Out(gpio.Low)
		if err != nil {
			return err
		}

		_ = lcd.write8Bits(0x03 << 4) // Get it's attention
		time.Sleep(4100 * time.Microsecond)
		_ = lcd.write8Bits(0x03 << 4)
		_ = lcd.write8Bits(0x03 << 4)
		_ = lcd.write8Bits(lineMode)
		_ = lcd.write8Bits(0x4) // set entry mode
	}
	_ = lcd.Cursor(display.CursorOff)
	_ = lcd.Display(true)
	_ = lcd.Clear()
	_ = lcd.Home()

	return lcd.Backlight(0xff)
}

func (lcd *HD44780) sendCommand(commands []byte) error {
	lcd.delayWrite(delayCommand)
	err := lcd.resetPin.Out(gpio.Level(modeCommand))
	if err != nil {
		return err
	}
	for _, command := range commands {
		if lcd.mode == mode4Bit {
			err = lcd.write4Bits(byte(command >> 4))
			if err == nil {
				err = lcd.write4Bits(byte(command))
			}
		} else {
			err = lcd.write8Bits(command)
		}
		if err != nil {
			break
		}

	}
	lcd.lastWrite = time.Now().UnixMicro()
	return err
}

func (lcd *HD44780) write4Bits(value byte) error {
	return lcd.writeBits(gpio.GPIOValue(value), 0x0f)
}

func (lcd *HD44780) write8Bits(value byte) error {
	return lcd.writeBits(gpio.GPIOValue(value), 0xff)
}

func (lcd *HD44780) writeBits(value, mask gpio.GPIOValue) error {
	err := lcd.dataPins.Out(value, mask)
	if err != nil {
		return err
	}
	err = lcd.enablePin.Out(gpio.High)
	if err == nil {
		time.Sleep(2 * time.Microsecond)
		err = lcd.enablePin.Out(gpio.Low)
	}
	return err
}

var _ display.TextDisplay = &HD44780{}
var _ display.DisplayBacklight = &HD44780{}
var _ conn.Resource = &HD44780{}
