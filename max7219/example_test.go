// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package max7219_test

import (
	"fmt"
	"log"
	"time"

	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/max7219"
	"periph.io/x/host/v3"
)

// basic test program. To do a numeric display, set matrix to false and
// matrixUnits to 1.
func Example() {
	// basic test program. To do a numeric display, set matrix to false and
	// matrixUnits to 1.
	matrix := false
	matrixUnits := 1
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	s, err := spireg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	dev, err := max7219.NewSPI(s, matrixUnits, 8)
	if err != nil {
		log.Fatal(err)
	}

	_ = dev.TestDisplay(true)
	time.Sleep(time.Second * 1)
	_ = dev.TestDisplay(false)

	_ = dev.SetIntensity(1)
	_ = dev.Clear()

	if matrix {
		dev.SetGlyphs(max7219.CP437Glyphs, true)
		dev.SetDecode(max7219.DecodeNone)
		dev.ScrollChars([]byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"), 1, 100*time.Millisecond)
	} else {
		dev.SetDecode(max7219.DecodeB)
	}
	for i := -128; i < 128; i++ {
		dev.WriteInt(i)
		time.Sleep(100 * time.Millisecond)
	}
	var tData []byte
	// Continuously display a clock
	for {
		t := time.Now()
		if matrix {
			// Assumes a 4 unit matrix
			tData = []byte(fmt.Sprintf("%2d%02d", t.Hour(), t.Minute()))
		} else {
			// 8 digit 7-segment LED display
			tData = []byte(t.Format(time.TimeOnly))
			tData[2] = max7219.ClearDigit
			tData[5] = max7219.ClearDigit
		}
		_ = dev.Write(tData)
		// Try to get the iteration exactly on time. FWIW, on a Pi Zero this loop
		// executes in ~ 2-3ms.
		dNext := time.Duration(1000-(t.UnixMilli()%1000)) * time.Millisecond
		time.Sleep(dNext)
	}
}
