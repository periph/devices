// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This package provides an interface to MatrixOrbital Character LCD displays.
// The LK2047T display is compatible with the Adafruit USB-LCD Backpack.
package matrixorbital

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/gpio"
)

// Constants for programmable LEDs on some models.
type LEDColor int

const (
	// Constants for colors supported by LEDs.
	Off LEDColor = iota
	Red
	Green
	Yellow
)

// The LK2047T is a basic MatrixOrbital LCD display. It's a 20x4 LCD with a
// keypad and LEDs.
//
// Implements periph.io/x/conn/v3/display.TextDisplay, Backlight, and
// DisplayContrast.
type LK2047T struct {
	// Pins represents the set of gpio.PinOut pins exposed by the device. For
	// units with LEDS, the pins are used to manipulate them. For the Adafruit
	// USB/LCD backpack, 4 pins are exposed.
	Pins []gpio.PinOut
	rows int
	cols int

	mu         sync.Mutex
	d          conn.Conn
	writer     io.Writer
	chKeyboard chan byte
	shutdown   chan struct{}
}

type GPOEnabledDisplay interface {
	// SetGPO turns a GPO pin on or off
	GPO(pin int, l gpio.Level) error
}

// Command byte values used by the display. This only implements a subset of
// commands.
var cmdByte byte = 0xfe
var autoScrollOff = []byte{cmdByte, 0x52}
var autoScrollOn = []byte{cmdByte, 0x51}
var blockCursorOff = []byte{cmdByte, 0x54}
var blockCursorOn = []byte{cmdByte, 0x53}
var clearScreen = []byte{cmdByte, 0x58}
var cursorBack = []byte{cmdByte, 0x4c}
var cursorBlinkOff = []byte{cmdByte, 0x54}
var cursorBlinkOn = []byte{cmdByte, 0x53}
var cursorForward = []byte{cmdByte, 0x4d}
var displayOff = []byte{cmdByte, 0x46}
var displayOn = []byte{cmdByte, 0x42}
var goHome = []byte{cmdByte, 0x48}
var keypadBacklightOff = []byte{cmdByte, 0x98}
var setBrightness = []byte{cmdByte, 0x99}
var setContrast = []byte{cmdByte, 0x50}
var setCursorPosition = []byte{cmdByte, 0x47}
var setGPOOn = []byte{cmdByte, 0x57}
var setGPOOff = []byte{cmdByte, 0x56}
var underlineCursorOff = []byte{cmdByte, 0x4b}
var underlineCursorOn = []byte{cmdByte, 0x4a}

func wrapErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("lk2047t: %w", err)
}

// Create a new LCD device using a periph.io/conn/Conn
func NewConnLK2047T(conn conn.Conn, rows, cols int) *LK2047T {
	dev := &LK2047T{d: conn, rows: rows, cols: cols, Pins: make([]gpio.PinOut, 6)}
	a := GPOEnabledDisplay(dev)
	makePins(&a, dev.Pins)
	return dev
}

// Create a new LCD device using an io.Writer. If your display is connected
// using a hardware interface that periph.io doesn't support (e.g. UART),
// you can still use this package as long as the hardware interface provides
// the io.Writer interface. rows is the number of lines the display supports,
// and cols is the character width of the device.
func NewWriterLK2047T(writer io.Writer, rows, cols int) *LK2047T {
	dev := &LK2047T{writer: writer, rows: rows, cols: cols, Pins: make([]gpio.PinOut, 6)}
	a := GPOEnabledDisplay(dev)
	makePins(&a, dev.Pins)
	return dev
}

// Enable or disable AutoScroll.
func (dev *LK2047T) AutoScroll(enabled bool) (err error) {
	if enabled {
		_, err = dev.Write(autoScrollOn)
	} else {
		_, err = dev.Write(autoScrollOff)
	}
	return
}

// Clears the screen, and moves the cursor to the home position.
func (dev *LK2047T) Clear() (err error) {
	_, err = dev.Write(clearScreen)
	if err == nil {
		err = dev.Home()
	}
	return
}

// Return the number of columns supported by the device.
func (dev *LK2047T) Cols() int {
	return dev.cols
}

// Set the cursor mode. E.G. underline, block, etc.
func (dev *LK2047T) Cursor(modes ...display.CursorMode) (err error) {
	for _, mode := range modes {
		switch mode {
		case display.CursorOff:
			_, err = dev.Write(blockCursorOff)
			if err == nil {
				_, err = dev.Write(underlineCursorOff)
			}
			if err == nil {
				_, err = dev.Write(cursorBlinkOff)
			}
		case display.CursorUnderline:
			_, err = dev.Write(underlineCursorOn)
		case display.CursorBlock:
			_, err = dev.Write(blockCursorOn)
		case display.CursorBlink:
			_, err = dev.Write(cursorBlinkOn)
		default:
			err = fmt.Errorf("lk2047t: invalid cursor mode %d", mode)
		}
		if err != nil {
			break
		}
	}
	return
}

// Halt shuts down the display, and closes the output device if it implements
// io.Closer. If a keypad read operation is running, closing the device will
// terminate it.
func (dev *LK2047T) Halt() (err error) {
	err = dev.Display(false)
	_ = dev.KeypadBacklight(false)
	if err != nil {
		return err
	}
	dev.mu.Lock()
	defer dev.mu.Unlock()
	if dev.shutdown != nil {
		dev.shutdown <- struct{}{}
	}
	var cl io.Closer
	var ok bool
	if dev.d != nil {
		cl, ok = dev.d.(io.Closer)
	} else {
		cl, ok = dev.writer.(io.Closer)
	}

	if ok {
		err = cl.Close()
	} else {
		err = errors.New("output connection doesn't support io.Closer()")
	}
	err = wrapErr(err)
	return
}

// Home resets the cursor to the default position.
func (dev *LK2047T) Home() (err error) {
	_, err = dev.Write(goHome)
	return
}

// MinCol returns the numbering scheme of the device's minimum column number.
// Generally, it will be 0 or 1
func (dev *LK2047T) MinCol() int {
	return 1
}

// MinRow returns the numbering scheme of the device's minimum row (line)
// number. Generally, it will be 0 or 1.
func (dev *LK2047T) MinRow() int {
	return 1
}

// Move the cursor forward or backwards.
func (dev *LK2047T) Move(direction display.CursorDirection) (err error) {
	switch direction {
	case display.Forward:
		_, err = dev.Write(cursorForward)
	case display.Backward:
		_, err = dev.Write(cursorBack)
	case display.Up:
	case display.Down:
	default:
		err = errors.New("lk2047t: invalid move direction")
	}
	return
}

// Move the cursor to an arbitrary row/column on the device.
func (dev *LK2047T) MoveTo(row, col int) (err error) {
	if row < 1 || row > dev.rows || col < 1 || col > dev.cols {
		return fmt.Errorf("lk2047t: MoveTo(%d, %d) value out of range", row, col)
	}
	_, err = dev.Write([]byte{setCursorPosition[0], setCursorPosition[1], byte(col), byte(row)})
	return err
}

// ReadKeypad reads from the displays built-in keypad. The io device used by the
// display must implement io.Reader. If it does not, then an error is returned.
func (dev *LK2047T) ReadKeypad() (<-chan byte, error) {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	if dev.chKeyboard != nil {
		return dev.chKeyboard, nil
	}
	var rdr io.Reader
	var ok bool
	if dev.writer == nil {
		rdr, ok = dev.d.(io.Reader)
	} else {
		rdr, ok = dev.writer.(io.Reader)
	}
	if !ok {
		return nil, errors.New("lk2047t: output device does not implement io.Reader")
	}

	dev.chKeyboard = make(chan byte, 8)
	dev.shutdown = make(chan struct{})
	go func() {
		defer func() {
			dev.mu.Lock()
			close(dev.chKeyboard)
			dev.chKeyboard = nil
			dev.mu.Unlock()
		}()
		buf := make([]byte, 4)
		var err error
		var n int
		for err == nil {
			select {
			case <-dev.shutdown:
				return
			default:
				n, err = rdr.Read(buf)
				if n > 0 {
					for ix := range n {
						dev.chKeyboard <- buf[ix]
					}
				}
			}
		}
	}()
	return dev.chKeyboard, nil
}

// Return the number of rows supported by the device.
func (dev *LK2047T) Rows() int {
	return dev.rows
}

// Set the intensity of the backlight. Refer to the docs in the lcd package
// for warnings on this function. Provides periph.io/x/conn/v3/display.Backlight
func (dev *LK2047T) Backlight(intensity display.Intensity) error {
	_, err := dev.Write([]byte{setBrightness[0], setBrightness[1], byte(intensity)})
	return err
}

// Set the constrast of the display.  Refer to the docs in the lcd package
// for warnings on this function. Provides periph.io/x/conn/v3/display.DisplayContrast
func (dev *LK2047T) Contrast(contrast display.Contrast) error {
	_, err := dev.Write([]byte{setContrast[0], setContrast[1], byte(contrast)})
	return err
}

// Set the display on or off.
func (dev *LK2047T) Display(on bool) (err error) {
	if on {
		_, err = dev.Write([]byte{displayOn[0], displayOn[1], 0})
	} else {
		_, err = dev.Write(displayOff)
	}
	return
}

func (dev *LK2047T) KeypadBacklight(on bool) error {
	if on {
		return dev.Display(on)
	}
	_, err := dev.Write(keypadBacklightOff)
	return err
}

// Set the specified output pin state.
func (dev *LK2047T) GPO(pin int, on gpio.Level) (err error) {

	if on {
		_, err = dev.Write([]byte{setGPOOn[0], setGPOOn[1], byte(pin)})
	} else {
		_, err = dev.Write([]byte{setGPOOff[0], setGPOOff[1], byte(pin)})
	}

	return
}

// Set an led to a supported color. number is 0 based.
func (dev *LK2047T) LED(number int, color LEDColor) error {
	if color < Off || color > Yellow {
		return fmt.Errorf("lk2047t: invalid color: %d", color)
	}
	err := dev.Pins[number*2].Out(gpio.Level(color&Red == Red))
	if err != nil {
		return err
	}
	return dev.Pins[number*2+1].Out(gpio.Level(color&Green == Green))
}

func (dev *LK2047T) String() string {
	var ioType any
	if dev.d != nil {
		ioType = dev.d
	} else {
		ioType = dev.writer
	}
	return fmt.Sprintf("MatrixOrbital LK204-7T LCD Display: Rows: %d Cols: %d Connection: %T", dev.rows, dev.cols, ioType)
}

// Write commands or data to the display
func (dev *LK2047T) Write(p []byte) (n int, err error) {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	if dev.writer == nil {
		err = dev.d.Tx(p, nil)
		n = len(p)
	} else {
		n, err = dev.writer.Write(p)
	}
	err = wrapErr(err)
	return
}

// WriteString sends a text string to the display.
func (dev *LK2047T) WriteString(text string) (int, error) {
	n, err := dev.Write([]byte(text))
	return n, err
}

var _ display.TextDisplay = &LK2047T{}
var _ GPOEnabledDisplay = &LK2047T{}
var _ display.DisplayContrast = &LK2047T{}
var _ display.DisplayBacklight = &LK2047T{}
var _ conn.Resource = &LK2047T{}
