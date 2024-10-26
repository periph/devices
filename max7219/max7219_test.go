// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package max7219

import (
	"fmt"
	"testing"
	"time"

	"periph.io/x/conn/v3/conntest"
	"periph.io/x/conn/v3/spi/spitest"
)

func TestReverseGlyphs(t *testing.T) {
	testVals := [][]byte{{0x01, 0xaa}, {0x80, 0x55}}
	expected := [][]byte{{0x80, 0x55}, {0x01, 0xaa}}
	testVals = reverseGlyphs(testVals)
	for outer := range len(expected) {
		for inner := range len(expected[0]) {
			if testVals[outer][inner] != expected[outer][inner] {
				t.Errorf("testVals[%d][%d] expected 0x%x found: 0x%x", outer, inner, expected[outer][inner], testVals[outer][inner])
			}
		}
	}
}

func TestConvertBytes(t *testing.T) {
	testStr := "-0.123456789EHLP "
	expected := []byte{MinusSign, 0 | DecimalPoint, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xb, 0xc, 0xd, 0xe, 0xf}
	converted := convertBytes([]byte(testStr))
	if len(converted) != len(expected) {
		t.Error("converted bytes length not as expected")
	}
	for ix := range len(expected) {
		if converted[ix] != expected[ix] {
			t.Errorf("error . expected 0x%x received 0x%x", expected[ix], converted[ix])
		}
	}
}

func TestGlyphs(t *testing.T) {
	// Verify our glyphs look OK.
	if len(CP437Glyphs) != 256 {
		t.Errorf("CP437 glphys not expected length. Got: %d", len(CP437Glyphs))
	}
	for ix := range len(CP437Glyphs) {
		if len(CP437Glyphs[ix]) != 8 {
			t.Errorf("Invalid glyph 0x%x found. Length: %d", ix, len(CP437Glyphs[ix]))
		}
	}
}

func verifyOperations(found, expected []conntest.IO) error {
	if len(found) != len(expected) {
		return fmt.Errorf("invalid length. found length: %d expected length: %d", len(found), len(expected))
	}
	for outer := range len(expected) {
		for inner := range len(found[outer].W) {
			if expected[outer].W[inner] != found[outer].W[inner] {
				return fmt.Errorf("data not as expected. found[%d][%d]=0x%x expected 0x%x",
					outer,
					inner,
					found[outer].W[inner],
					expected[outer].W[inner])
			}
		}
	}
	return nil
}

func TestInit(t *testing.T) {
	pb := &spitest.Playback{Playback: conntest.Playback{DontPanic: true, Count: 1}}
	defer pb.Close()
	record := &spitest.Record{}

	_, err := NewSPI(record, 1, 8)
	if err != nil {
		t.Error(err)
	}
	expected := []conntest.IO{
		{W: []uint8{0xf, 0x0}},  // Disable self-test
		{W: []uint8{0xc, 0x0}},  // Shutdown - Enter Shutdown Mode
		{W: []uint8{0xa, 0x8}},  // Intensity
		{W: []uint8{0xb, 0x7}},  // Scan Limit
		{W: []uint8{0xc, 0x1}},  // Shutdown - Resume Normal Mode
		{W: []uint8{0x9, 0xff}}, // Decode Mode
		{W: []uint8{0x8, 0xf}},  // Clear digits 1-8
		{W: []uint8{0x7, 0xf}},
		{W: []uint8{0x6, 0xf}},
		{W: []uint8{0x5, 0xf}},
		{W: []uint8{0x4, 0xf}},
		{W: []uint8{0x3, 0xf}},
		{W: []uint8{0x2, 0xf}},
		{W: []uint8{0x1, 0xf}}}

	err = verifyOperations(record.Ops, expected)
	if err != nil {
		t.Error(err)
	}
}

func TestWrite(t *testing.T) {
	pb := &spitest.Playback{Playback: conntest.Playback{DontPanic: true, Count: 1}}
	defer pb.Close()
	record := &spitest.Record{}

	dev, err := NewSPI(record, 1, 8)
	record.Ops = make([]conntest.IO, 0)
	dev.Write([]byte("12345678"))
	if err != nil {
		t.Error(err)
	}

	expected := []conntest.IO{
		{W: []uint8{0x8, 0x1}},
		{W: []uint8{0x7, 0x2}},
		{W: []uint8{0x6, 0x3}},
		{W: []uint8{0x5, 0x4}},
		{W: []uint8{0x4, 0x5}},
		{W: []uint8{0x3, 0x6}},
		{W: []uint8{0x2, 0x7}},
		{W: []uint8{0x1, 0x8}}}

	err = verifyOperations(record.Ops, expected)
	if err != nil {
		t.Error(err)
	}
	record.Ops = make([]conntest.IO, 0)
	dev.WriteInt(12345678)
	if err != nil {
		t.Error(err)
	}
	err = verifyOperations(record.Ops, expected)
	if err != nil {
		t.Error(err)
	}
}

func TestScroll(t *testing.T) {
	pb := &spitest.Playback{Playback: conntest.Playback{DontPanic: true, Count: 1}}
	defer pb.Close()
	record := &spitest.Record{}

	dev, err := NewSPI(record, 1, 1)
	record.Ops = make([]conntest.IO, 0)
	dev.ScrollChars([]byte("12"), 2, time.Millisecond)
	if err != nil {
		t.Error(err)
	}

	expected := []conntest.IO{
		{W: []uint8{0x1, 0x1}}, // Write 1st digit to position 1
		{W: []uint8{0x1, 0x2}}, // Write 2nd digit to position 1
		{W: []uint8{0x1, 0xf}}, // Write blank
		{W: []uint8{0x1, 0x1}}} // Write 1st digit to position 1

	err = verifyOperations(record.Ops, expected)
	if err != nil {
		t.Error(err)
	}
}

func TestScrollGlyphs(t *testing.T) {
	pb := &spitest.Playback{Playback: conntest.Playback{DontPanic: true, Count: 1}}
	defer pb.Close()
	record := &spitest.Record{}

	dev, err := NewSPI(record, 4, 8) // Simulate a 4 unit matrix
	dev.SetDecode(DecodeNone)
	dev.SetGlyphs(CP437Glyphs, true)
	record.Ops = make([]conntest.IO, 0)
	dev.ScrollChars([]byte("123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"), 3, time.Millisecond)
	if err != nil {
		t.Error(err)
	}
	// This generates a lot of data. Just make sure the operation count matches
	// what we expected.
	expectedOps := 6728
	if len(record.Ops) != expectedOps {
		t.Errorf("expected %d operations, received %d", expectedOps, len(record.Ops))
	}

}

func TestCascadedWrite(t *testing.T) {
	// ops := make([]conntest.IO, 0)
	pb := &spitest.Playback{Playback: conntest.Playback{DontPanic: true, Count: 1}}
	defer pb.Close()
	record := &spitest.Record{}

	dev, err := NewSPI(record, 2, 4)
	dev.SetDecode(DecodeB)
	record.Ops = make([]conntest.IO, 0)
	// imaginarily write 8 characters to two 4 digit displays.
	dev.Write([]byte("12345678"))
	if err != nil {
		t.Error(err)
	}

	expected := []conntest.IO{
		{W: []uint8{0x1, 0x8, 0x1, 0x4}}, // Unit 1 digit 8, Unit 0 digit 4
		{W: []uint8{0x2, 0x7, 0x2, 0x3}}, // Unit 1 digit 7, unit 0 digit 3
		{W: []uint8{0x3, 0x6, 0x3, 0x2}}, // unit 1 digit 6, unit 0 digit 2
		{W: []uint8{0x4, 0x5, 0x4, 0x1}}} // unit 1 digit 5, unit 0 digit 1

	err = verifyOperations(record.Ops, expected)
	if err != nil {
		t.Error(err)
	}
}

func TestCommand(t *testing.T) {
	// Verify a command is replicated #units times.
	pb := &spitest.Playback{Playback: conntest.Playback{DontPanic: true, Count: 1}}
	defer pb.Close()
	record := &spitest.Record{}

	dev, err := NewSPI(record, 4, 8)
	record.Ops = make([]conntest.IO, 0)
	err = dev.SetIntensity(0x0b)

	if err != nil {
		t.Error(err)
	}
	expected := []conntest.IO{
		{W: []uint8{0xa, 0xb, 0xa, 0xb, 0xa, 0xb, 0xa, 0xb}}} // Set intensity register and the value replicated units times.

	err = verifyOperations(record.Ops, expected)
	if err != nil {
		t.Error(err)
	}
}
