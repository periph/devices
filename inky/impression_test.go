// Copyright 2019 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package inky

import (
	"image/color"
	"testing"
)

func TestDevImpression_BlendPrecision(t *testing.T) {
	// Saturation level that triggers truncation if not handled correctly.
	// For example, if rs=10.9 and rd=5.9:
	// Old: uint8(10.9) + uint8(5.9) = 10 + 5 = 15
	// New: uint8(10.9 + 5.9) = uint8(16.8) = 16

	// We need to simulate this with real palette values and saturation.
	// Let's use a simplified case.
	// satPalette[0].R = 109, dscPalette[0].R = 59, saturation = 10%
	// rs = 109 * 0.1 = 10.9
	// rd = 59 * 0.9 = 53.1
	// Old: 10 + 53 = 63
	// New: uint8(10.9 + 53.1) = uint8(64.0) = 64

	d := &DevImpression{
		saturation: 10,
	}
	d.Dev.model = IMPRESSION73 // This uses sc7 and dsc

	// Override sc7 and dsc for the test up to at least common length
	oldSc7 := sc7
	oldDsc := dsc
	defer func() {
		sc7 = oldSc7
		dsc = oldDsc
	}()

	sc7 = []color.NRGBA{{R: 109, G: 0, B: 0, A: 255}}
	dsc = []color.NRGBA{{R: 59, G: 0, B: 0, A: 255}}

	p := d.blend()
	c := p[0].(color.RGBA)

	if c.R != 64 {
		t.Errorf("Expected R=64, got %d. Precision loss detected!", c.R)
	}
}
