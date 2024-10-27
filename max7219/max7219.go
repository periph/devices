// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.
//
// The max7219 package provides a simple interface for displaying data on
// numeric 7-segment displays, or on matrix displays. It simplifies writes
// and provides useful features like scrolling characters on either type
// of display unit.
package max7219

import (
	"errors"
	"fmt"
	"log"
	"time"

	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
)

// DecodeMode is the mode for handling data. Refer to the datasheet for
// more information.
type DecodeMode byte

const (
	_REGISTER_NOOP         byte = 0x0
	_REGISTER_DECODE_MODE  byte = 0x9
	_REGISTER_INTENSITY    byte = 0xa
	_REGISTER_SCAN_LIMIT   byte = 0xb
	_REGISTER_SHUTDOWN     byte = 0xc
	_REGISTER_DISPLAY_TEST byte = 0xf

	// Value to write to a Code B Font decoded register to blank out the
	// digit.
	ClearDigit byte = 0x0f
	// Value to write for a minus sign symbol
	MinusSign byte = 0x0a
	// To turn the decimal point on for a digit display, OR the value of the
	// digit with DecimalPoint
	DecimalPoint byte = 0x80

	// DecodeB is used for numeric segment displays. E.G. given a binary 0,
	// it would turn on the appropriate segments to display the character 0.
	DecodeB DecodeMode = 0xff
	// DecodeNone is RAW mode, or not decoded. For each byte, bits that are
	// one are turned on in the matrix, and bits that are 0 turn off the
	// led at that row/column.
	DecodeNone DecodeMode = 0
)

// Type for a Maxim MAX7219/MAX7221 device.
type Dev struct {
	conn spi.Conn
	// decode mode for all data registers
	decode DecodeMode
	// units is the number of 7219 units daisy-chained together.
	units int
	// The number of digits (or the NxN matrix size) in this display.
	digits byte
	// charset is the character set for matrix display use. The first index
	// is the code point (e.g. 0x20=32=space. The second index is the 8 byte
	// raster values for each line in the matrix.
	glyphs [][]byte
}

// emptyBytes creates a slice of empty bytes (digit values or byte values)
// for the display.
func (d *Dev) emptyBytes() []byte {
	b := make([]byte, d.digits)
	if d.decode == DecodeB {
		for ix := range b {
			b[ix] = ClearDigit
		}
	}
	return b
}

// init puts the display in the default mode. The default is to
// clear the display, set intensity to middle, and set Decode
// to DecodeB for a single unit, or DecodeNone for multi-unit.
func (d *Dev) init() {
	var initCommands = [][]byte{
		{_REGISTER_DISPLAY_TEST, 0x0},
		{_REGISTER_SHUTDOWN, 0x00},
		{_REGISTER_INTENSITY, 0x08},
		{_REGISTER_SCAN_LIMIT, d.digits - 1},
		{_REGISTER_SHUTDOWN, 0x01}}

	for _, cmd := range initCommands {
		err := d.sendCommand(cmd[0], cmd[1])
		if err != nil {
			log.Println(err)
			break
		}
	}
	if d.units == 1 {
		_ = d.SetDecode(DecodeB)
	} else {
		_ = d.SetDecode(DecodeNone)
	}

	_ = d.Clear()
}

// sendCommand writes to a data register or command register.
// Data registers are 1-8, and command registers are > 8.
// If multiple units are daisychained together, the command
// is repeated and sent to all units.
func (d *Dev) sendCommand(register, data byte) error {
	w := make([]byte, d.units*2)
	for ix := range d.units {
		w[ix*2] = register
		w[ix*2+1] = data
	}
	return d.conn.Tx(w, nil)
}

// NewSPI creates a new Max7219 using the specified spi.Port. units is the number
// of Max7219 chips daisy-chained together. numDigits is the number of digits
// displayed.
func NewSPI(p spi.Port, units, numDigits int) (*Dev, error) {
	if units <= 0 {
		return nil, errors.New("max7219: invalid value for number of cascaded units")
	}
	if numDigits <= 0 || numDigits > 8 {
		return nil, errors.New("max7219: invalid value for number of digits")
	}

	// It works in Mode0, Mode2 and Mode3.
	c, err := p.Connect(10*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return nil, fmt.Errorf("max7219: %v", err)
	}
	d := &Dev{conn: c, digits: byte(numDigits), units: units, glyphs: nil}
	d.init()
	return d, nil
}

// Clear erases the content of all display segments or matrix LEDs.
func (d *Dev) Clear() error {
	empty := d.emptyBytes()
	if d.units > 1 {
		w := make([][]byte, d.units)
		for ix := range d.units {
			w[ix] = empty
		}
		return d.WriteCascadedUnits(w)
	} else {
		return d.Write(empty)
	}
}

// shiftBytes shifts an array of raster characters left one LED Column (bit).
// Used to continuously scroll a display of glyphs.
func shiftBytes(bytes [][]byte) {
	// At least this is how my matrix is wired :)
	const rightMostBit byte = 0x80
	const leftMostBit byte = 0x01
	var rasterLimit = len(bytes[0])
	// Save the left most character so when we shift the rightmost
	// character, we can put the leftmost bit of the first character into it.
	saveReg := make([]byte, rasterLimit)
	copy(saveReg, bytes[0])

	var nextByte, curByte byte
	var limit = len(bytes) - 1

	for char := 0; char <= limit; char++ {
		for rasterLine := 0; rasterLine < rasterLimit; rasterLine++ {
			curByte = bytes[char][rasterLine] >> 1
			if char == limit {
				nextByte = saveReg[rasterLine]
			} else {
				nextByte = bytes[char+1][rasterLine]
			}
			if nextByte&leftMostBit > 0 {
				curByte |= rightMostBit
			}
			bytes[char][rasterLine] = curByte
		}
	}
}

// ScrollChars takes a character array and scrolls it from right-to-left, one
// led column at a time. This can be used to scroll a matrix display of glyphs,
// or digits on a seven-segment display. If the length of data is less than
// the number of display units, it writes that directly without scrolling.
func (d *Dev) ScrollChars(data []byte, scrollCount int, updateInterval time.Duration) {
	if d.decode == DecodeNone {
		// This is a matrix

		// Create a temporary copy of the data to modify - We don't want to munge
		// our glyph set.
		bytes := make([][]byte, len(data))
		for ix, val := range data {
			newVals := make([]byte, 8)
			copy(newVals, d.glyphs[val])
			bytes[ix] = newVals
		}
		_ = d.WriteCascadedUnits(bytes)

		for shifts := scrollCount * len(data) * int(d.digits); shifts > 0; shifts-- {
			time.Sleep(updateInterval)
			shiftBytes(bytes)
			_ = d.WriteCascadedUnits(bytes)
		}
	} else {
		// This is a seven segment display.
		data = convertBytes(data)
		if len(data) <= int(d.digits)*d.units {
			_ = d.Write(data)
			time.Sleep(time.Duration(scrollCount*len(data)) * updateInterval)
			return
		}

		displayData := make([]byte, 0)
		displayData = append(displayData, data...)
		displayData = append(displayData, byte(ClearDigit))
		displayData = append(displayData, data...)

		var pos int
		for shifts := scrollCount * len(data); shifts > 0; shifts-- {
			_ = d.Write(displayData[pos : pos+int(d.digits)*d.units])
			pos = pos + 1
			if pos >= (len(data) + 1) {
				pos = 0
			}
			time.Sleep(updateInterval)
		}
	}
}

// SetGlyphs allows you to set the character set for use by the matrix display.
// If the endianness of the charset doesn't match that used by the max7219,
// pass true for reverse and it will change the endianness of the raster
// values.
//
// You only have to supply glyphs for values you intend to write. So if you're
// just writing digits, you just need 0-9 and whatever punctuation marks you
// need.
func (d *Dev) SetGlyphs(glyphs [][]byte, reverse bool) {
	if reverse {
		d.glyphs = reverseGlyphs(glyphs)
	} else {
		d.glyphs = glyphs
	}
}

// SetDecode tells the Max7219 whether values should be decoded for a 7 segment
// display, or if they should be interpreted literally. Refer to the datasheet
// for more detailed information.
func (d *Dev) SetDecode(mode DecodeMode) error {
	d.decode = mode
	return d.sendCommand(_REGISTER_DECODE_MODE, byte(mode))
}

// SetIntensity controls the brightness of the display. The allowed range for
// intensity is from 0-15. Keep in mind that the brighter display, the more
// current drawn.
func (d *Dev) SetIntensity(intensity byte) error {
	return d.sendCommand(_REGISTER_INTENSITY, intensity&0x0f)
}

// TestDisplay turns on the 7219 display mode which set all segments (or LEDs) on,
// and  the intensity to maximum. If you're using multiple units, you should be
// aware  of the current draw, and limit how long you leave this on.
func (d *Dev) TestDisplay(on bool) error {
	if on {
		return d.sendCommand(_REGISTER_DISPLAY_TEST, 1)
	} else {
		return d.sendCommand(_REGISTER_DISPLAY_TEST, 0)
	}
}

// writeChars sends characters as glyphs out to the device(s). It stacks them
// into a two-dimensional slice and then writes them out.
func (d *Dev) writeChars(bytes []byte) error {
	w := make([][]byte, d.units)
	for ix := range d.units {
		x := make([]byte, 8)
		copy(x, d.glyphs[0x20])
		w[ix] = x
	}
	charPos := len(bytes) - 1
	for ix := d.units - 1; ix >= 0 && charPos >= 0; ix-- {
		w[ix] = d.glyphs[bytes[charPos]]
		charPos = charPos - 1
	}
	return d.WriteCascadedUnits(w)
}

// convertBytes converts ascii characters into their appropriate CodeB
// representations. Refer to the datasheet.
func convertBytes(bytes []byte) []byte {
	newBytes := make([]byte, 0, len(bytes))
	for ix, c := range bytes {
		if c >= '0' && c <= '9' {
			newBytes = append(newBytes, c-'0')
		} else {
			switch c {
			case ' ':
				newBytes = append(newBytes, ClearDigit)
			case '-':
				newBytes = append(newBytes, MinusSign)
			case '.':
				if ix > 0 {
					// A decimal point is OR'd onto the previous
					// digit to turn it on.
					newBytes[len(newBytes)-1] |= DecimalPoint
				}
			case 'E':
				newBytes = append(newBytes, 0xb)
			case 'H':
				newBytes = append(newBytes, 0xc)
			case 'L':
				newBytes = append(newBytes, 0xd)
			case 'P':
				newBytes = append(newBytes, 0xe)
			default:
				newBytes = append(newBytes, c)
			}
		}
	}
	return newBytes
}

// Write sends data to the display unit.
//
// If decode is DecodeNone, then it's assumed we're writing to a matrix. If
// a glyph set has been set, then bytes are treated as offsets into the
// character table.
//
// If decode is DecodeB, then any ASCII characters are converted into their
// supported CodeB values and written. If units are cascaded, this method
// automatically handles re-formatting the data, and writing it to the cascaded
// 7219 units.
func (d *Dev) Write(bytes []byte) error {

	if d.decode == DecodeNone {
		return d.writeChars(bytes)
	}

	bytes = convertBytes(bytes)
	if d.units == 1 {
		// single-unit numeric display.
		var digit = d.digits
		w := make([]byte, 2)
		for _, val := range bytes {
			w[0] = digit
			w[1] = val

			err := d.conn.Tx(w, nil)
			if err != nil {
				return err
			}
			digit -= 1
		}
	} else {
		// A multi-unit display.
		writeData := make([][]byte, d.units)
		for padding := 0; padding < d.units; padding++ {
			writeData[padding] = make([]byte, d.digits)
			for ix := 0; ix < int(d.digits); ix++ {
				writeData[padding][ix] = ClearDigit
			}
		}
		unit := d.units - 1
		digit := d.digits
		for char := len(bytes) - 1; char >= 0 && unit >= 0; char-- {
			c := bytes[char]
			writeData[unit][digit-1] = c
			if digit == 1 {
				digit = d.digits
				unit -= 1
			} else {
				digit -= 1
			}
		}
		return d.WriteCascadedUnits(writeData)
	}
	return nil
}

// WriteInt provide a convenience method that displays the specified integer
// value on the display. It will work for either matrixes (with a glyph set)
// or numeric displays.
func (d *Dev) WriteInt(value int) error {
	digits := d.units
	if d.decode != DecodeNone {
		digits *= int(d.digits)
	}
	return d.Write([]byte(fmt.Sprintf("%*d", digits, value)))
}

// WriteCascadedUnits writes a 2D array of raster characters to
// a a set of cascaded max7219 devices. For example, a 4 unit
// 8*8 matrix, or two eight digit 7-segment LEDs. This handles
// the complexities of how data is shifted from one 7219
// to the next in a chain.
func (d *Dev) WriteCascadedUnits(bytes [][]byte) error {
	matrixCount := len(bytes)
	for rasterLine := 0; rasterLine < int(d.digits); rasterLine++ {
		w := make([]byte, 0)
		for matrix := matrixCount - 1; matrix >= 0; matrix-- {
			w = append(w, byte(rasterLine+1))
			w = append(w, bytes[matrix][int(d.digits-1)-rasterLine])
		}
		err := d.conn.Tx(w, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// WriteCascadedUnit writes data to a single display unit in
// a set of cascaded 7219 chips. offset is the 0 based number of
// the unit to write to. You could use this to update one segment
// of a cascaded matrix. Imagine rolling a digit upwards to bring
// in a new one...
func (d *Dev) WriteCascadedUnit(offset int, data []byte) error {
	for i := byte(0); i < d.digits; i++ {
		w := make([]byte, 0)
		for matrix := d.units - 1; matrix >= 0; matrix-- {
			if matrix == offset {
				w = append(w, byte(i+1))
				w = append(w, data[(d.digits-1)-i])
			} else {
				w = append(w, _REGISTER_NOOP)
				w = append(w, 0)
			}

		}
		err := d.conn.Tx(w, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
