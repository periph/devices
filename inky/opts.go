// Copyright 2023 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package inky

import (
	"encoding/binary"
	"fmt"

	"periph.io/x/conn/v3/i2c"
)

var (
	displayVariantMap = map[int]string{
		0:  "",
		1:  "Red pHAT (High-Temp)",
		2:  "Yellow wHAT",
		3:  "Black wHAT",
		4:  "Black pHAT",
		5:  "Yellow pHAT",
		6:  "Red wHAT",
		7:  "Red wHAT (High-Temp)",
		8:  "Red wHAT",
		9:  "",
		10: "Black pHAT (SSD1608)",
		11: "Red pHAT (SSD1608)",
		12: "Yellow pHAT (SSD1608)",
		13: "",
		14: "7-Colour (UC8159)",
		15: "7-Colour 640x400 (UC8159)",
		16: "7-Colour 640x400 (UC8159)",
		17: "Black wHAT (SSD1683)",
		18: "Red wHAT (SSD1683)",
		19: "Yellow wHAT (SSD1683)",
	}
)

// Opts is the options to specify which device is being controlled and its
// default settings.
type Opts struct {
	// Boards's width and height.
	Width  int
	Height int

	// Model being used.
	Model Model
	// Model color.
	ModelColor Color
	// Initial border color. Will be set on the first Draw().
	BorderColor Color

	// Board information.
	PCBVariant     float64
	DisplayVariant int
}

// DetectOpts tries to read the device opts from EEPROM.
func DetectOpts(bus i2c.Bus) (*Opts, error) {
	// Read data from EEPROM
	data, err := readEep(bus)
	if err != nil {
		return nil, fmt.Errorf("failed to detect Inky board: %v", err)
	}
	options := new(Opts)

	options.Width = int(binary.LittleEndian.Uint16(data[0:]))
	options.Height = int(binary.LittleEndian.Uint16(data[2:]))

	switch data[4] {
	case 1:
		options.ModelColor = Black
		options.BorderColor = Black
	case 2:
		options.ModelColor = Red
		options.BorderColor = Red
	case 3:
		options.ModelColor = Yellow
		options.BorderColor = Yellow
	case 4:
		options.ModelColor = Multi
		options.BorderColor = Color(WhiteImpression)
	default:
		return nil, fmt.Errorf("failed to get ops: color %v not supported", data[4])
	}

	options.PCBVariant = float64(data[5] / 10.0)

	switch data[6] {
	case 1, 4, 5:
		options.Model = PHAT
	case 10, 11, 12:
		options.Model = PHAT2
	case 2, 3, 6, 7, 8:
		options.Model = WHAT
	case 14:
		options.Model = IMPRESSION57
	case 15, 16:
		options.Model = IMPRESSION4
	default:
		return nil, fmt.Errorf("failed to get ops: display type %v not supported", data[6])
	}

	options.DisplayVariant = int(data[6])

	return options, nil
}

func readEep(bus i2c.Bus) ([]byte, error) {
	// Inky uses SMBus, specify read registry with data
	write := []byte{0x00, 0x00}

	data := make([]byte, 29)

	if err := bus.Tx(0x50, write, data); err != nil {
		return nil, err
	}

	return data, nil
}
