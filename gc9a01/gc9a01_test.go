// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package gc9a01

import (
	"errors"
	"image"
	"image/color"
	"testing"

	"periph.io/x/conn/v3/conntest"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpiotest"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spitest"
)

func TestNew(t *testing.T) {
	port := getPlayback(t)
	dev, err := New(port, &gpiotest.Pin{N: "dc", Num: 1}, nil, &DefaultOpts)
	if err != nil {
		t.Fatal(err)
	}
	if dev == nil {
		t.Fatal("expected device")
	}
	if err := port.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNew_fail_invalid_dc(t *testing.T) {
	if d, err := New(&spitest.Playback{}, gpio.INVALID, nil, &DefaultOpts); d != nil || err == nil {
		t.Fatal("expected failure with gpio.INVALID dc pin")
	}
}

func TestNew_fail_dc_err(t *testing.T) {
	if d, err := New(&spitest.Playback{}, &failPin{fail: true}, nil, &DefaultOpts); d != nil || err == nil {
		t.Fatal("expected failure when dc pin fails")
	}
}

func TestNew_fail_connect(t *testing.T) {
	if d, err := New(&configFail{}, &gpiotest.Pin{N: "dc", Num: 1}, nil, &DefaultOpts); d != nil || err == nil {
		t.Fatal("expected failure when SPI connect fails")
	}
}

func TestColorModel(t *testing.T) {
	port := getPlayback(t)
	dev, err := New(port, &gpiotest.Pin{N: "dc", Num: 1}, nil, &DefaultOpts)
	if err != nil {
		t.Fatal(err)
	}
	if c := dev.ColorModel(); c != color.NRGBAModel {
		t.Fatalf("expected NRGBAModel, got %v", c)
	}
	if err := port.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBounds(t *testing.T) {
	port := getPlayback(t)
	dev, err := New(port, &gpiotest.Pin{N: "dc", Num: 1}, nil, &DefaultOpts)
	if err != nil {
		t.Fatal(err)
	}
	expected := image.Rect(0, 0, 240, 240)
	if b := dev.Bounds(); b != expected {
		t.Fatalf("expected %v, got %v", expected, b)
	}
	if err := port.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestString(t *testing.T) {
	port := getPlayback(t)
	dev, err := New(port, &gpiotest.Pin{N: "dc", Num: 1}, nil, &DefaultOpts)
	if err != nil {
		t.Fatal(err)
	}
	expected := "GC9A01{playback, dc(1), (240,240)}"
	if s := dev.String(); s != expected {
		t.Fatalf("%q != %q", expected, s)
	}
	if err := port.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHalt(t *testing.T) {
	port := &spitest.Playback{
		Playback: conntest.Playback{
			Ops: append(initOps(),
				// Halt: DC low, then DISPOFF
				conntest.IO{W: []byte{_DISPOFF}},
			),
		},
	}
	dev, err := New(port, &gpiotest.Pin{N: "dc", Num: 1}, nil, &DefaultOpts)
	if err != nil {
		t.Fatal(err)
	}
	if err := dev.Halt(); err != nil {
		t.Fatal(err)
	}
	if !dev.halted {
		t.Fatal("expected halted to be true")
	}
	if err := port.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestInvert(t *testing.T) {
	port := &spitest.Playback{
		Playback: conntest.Playback{
			Ops: append(initOps(),
				conntest.IO{W: []byte{_INVOFF}},
			),
		},
	}
	dev, err := New(port, &gpiotest.Pin{N: "dc", Num: 1}, nil, &DefaultOpts)
	if err != nil {
		t.Fatal(err)
	}
	// The display inversion is ON by default (from init), so Invert(false)
	// should send INVOFF.
	if err := dev.Invert(false); err != nil {
		t.Fatal(err)
	}
	if err := port.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDraw_noop(t *testing.T) {
	port := getPlayback(t)
	dev, err := New(port, &gpiotest.Pin{N: "dc", Num: 1}, nil, &DefaultOpts)
	if err != nil {
		t.Fatal(err)
	}
	// First draw: full frame of black (matches zero buffer). But dirty=true
	// forces full send.
	// Instead let's just clear dirty and test that a second identical draw
	// sends nothing.
	dev.dirty = false
	// Draw all black (matches the zero-initialized buffer).
	black := image.NewNRGBA(dev.Bounds())
	if err := dev.Draw(dev.Bounds(), black, image.Point{}); err != nil {
		t.Fatal(err)
	}
	if err := port.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNrgbaToRGB565(t *testing.T) {
	// Pure red: R=0xFF -> 0b11111 = 31, G=0, B=0
	// hi = (31 << 3) | 0 = 0xF8, lo = 0x00
	hi, lo := nrgbaToRGB565(0xFF, 0x00, 0x00)
	if hi != 0xF8 || lo != 0x00 {
		t.Fatalf("red: got 0x%02X 0x%02X, expected 0xF8 0x00", hi, lo)
	}

	// Pure green: R=0, G=0xFF -> 0b111111 = 63, B=0
	// hi = (0 << 3) | (63 >> 3) = 0x07, lo = (63 << 5) | 0 = 0xE0
	hi, lo = nrgbaToRGB565(0x00, 0xFF, 0x00)
	if hi != 0x07 || lo != 0xE0 {
		t.Fatalf("green: got 0x%02X 0x%02X, expected 0x07 0xE0", hi, lo)
	}

	// Pure blue: R=0, G=0, B=0xFF -> 0b11111 = 31
	// hi = 0, lo = 0 | 31 = 0x1F
	hi, lo = nrgbaToRGB565(0x00, 0x00, 0xFF)
	if hi != 0x00 || lo != 0x1F {
		t.Fatalf("blue: got 0x%02X 0x%02X, expected 0x00 0x1F", hi, lo)
	}

	// White: all 0xFF
	// hi = (31 << 3) | (63 >> 3) = 0xFF, lo = (63 << 5) | 31 = 0xFF
	hi, lo = nrgbaToRGB565(0xFF, 0xFF, 0xFF)
	if hi != 0xFF || lo != 0xFF {
		t.Fatalf("white: got 0x%02X 0x%02X, expected 0xFF 0xFF", hi, lo)
	}

	// Black: all 0
	hi, lo = nrgbaToRGB565(0x00, 0x00, 0x00)
	if hi != 0x00 || lo != 0x00 {
		t.Fatalf("black: got 0x%02X 0x%02X, expected 0x00 0x00", hi, lo)
	}
}

func TestDraw_gpio_fail(t *testing.T) {
	port := getPlayback(t)
	pin := &failPin{fail: false}
	dev, err := New(port, pin, nil, &DefaultOpts)
	if err != nil {
		t.Fatal(err)
	}
	// GPIO suddenly fails.
	pin.fail = true
	img := image.NewNRGBA(dev.Bounds())
	if err := dev.Draw(dev.Bounds(), img, image.Point{}); err == nil || err.Error() != "injected error" {
		t.Fatalf("expected injected error, got %v", err)
	}
}

// initOps returns the conntest.IO operations expected during initialization.
// Each command and its data are sent as separate SPI transactions.
func initOps() []conntest.IO {
	rotation := Rotation0
	madctl := madctlValues[rotation]

	ops := []conntest.IO{}
	type initCmd struct {
		c    byte
		data []byte
	}
	cmds := []initCmd{
		{_SWRESET, nil},
		{0xEF, nil},
		{0xEB, []byte{0x14}},
		{0xFE, nil},
		{0xEF, nil},
		{0xEB, []byte{0x14}},
		{0x84, []byte{0x40}},
		{0x85, []byte{0xFF}},
		{0x86, []byte{0xFF}},
		{0x87, []byte{0xFF}},
		{0x88, []byte{0x0A}},
		{0x89, []byte{0x21}},
		{0x8A, []byte{0x00}},
		{0x8B, []byte{0x80}},
		{0x8C, []byte{0x01}},
		{0x8D, []byte{0x01}},
		{0x8E, []byte{0xFF}},
		{0x8F, []byte{0xFF}},
		{0xB6, []byte{0x00, 0x00}},
		{_MADCTL, []byte{madctl}},
		{_COLMOD, []byte{0x05}},
		{0x90, []byte{0x08, 0x08, 0x08, 0x08}},
		{0xBD, []byte{0x06}},
		{0xBC, []byte{0x00}},
		{0xFF, []byte{0x60, 0x01, 0x04}},
		{0xC3, []byte{0x13}},
		{0xC4, []byte{0x13}},
		{0xC9, []byte{0x22}},
		{0xBE, []byte{0x11}},
		{0xE1, []byte{0x10, 0x0E}},
		{0xDF, []byte{0x21, 0x0C, 0x02}},
		{0xF0, []byte{0x45, 0x09, 0x08, 0x08, 0x26, 0x2A}},
		{0xF1, []byte{0x43, 0x70, 0x72, 0x36, 0x37, 0x6F}},
		{0xF2, []byte{0x45, 0x09, 0x08, 0x08, 0x26, 0x2A}},
		{0xF3, []byte{0x43, 0x70, 0x72, 0x36, 0x37, 0x6F}},
		{0xED, []byte{0x1B, 0x0B}},
		{0xAE, []byte{0x77}},
		{0xCD, []byte{0x63}},
		{0x70, []byte{0x07, 0x07, 0x04, 0x0E, 0x0F, 0x09, 0x07, 0x08, 0x03}},
		{0xE8, []byte{0x34}},
		{0x62, []byte{0x18, 0x0D, 0x71, 0xED, 0x70, 0x70, 0x18, 0x0F, 0x71, 0xEF, 0x70, 0x70}},
		{0x63, []byte{0x18, 0x11, 0x71, 0xF1, 0x70, 0x70, 0x18, 0x13, 0x71, 0xF3, 0x70, 0x70}},
		{0x64, []byte{0x28, 0x29, 0xF1, 0x01, 0xF1, 0x00, 0x07}},
		{0x66, []byte{0x3C, 0x00, 0xCD, 0x67, 0x45, 0x45, 0x10, 0x00, 0x00, 0x00}},
		{0x67, []byte{0x00, 0x3C, 0x00, 0x00, 0x00, 0x01, 0x54, 0x10, 0x32, 0x98}},
		{0x74, []byte{0x10, 0x85, 0x80, 0x00, 0x00, 0x4E, 0x00}},
		{0x98, []byte{0x3E, 0x07}},
		{0x35, nil},
		{_INVON, nil},
		{_SLPOUT, nil},
		{_DISPON, nil},
	}

	for _, c := range cmds {
		// Command byte (DC low).
		ops = append(ops, conntest.IO{W: []byte{c.c}})
		// Data bytes (DC high), if any.
		if len(c.data) > 0 {
			ops = append(ops, conntest.IO{W: c.data})
		}
	}
	return ops
}

func getPlayback(t *testing.T) *spitest.Playback {
	t.Helper()
	return &spitest.Playback{
		Playback: conntest.Playback{
			Ops: initOps(),
		},
	}
}

type configFail struct {
	spitest.Record
}

func (c *configFail) Connect(f physic.Frequency, mode spi.Mode, bits int) (spi.Conn, error) {
	return nil, errors.New("injected error")
}

type failPin struct {
	gpiotest.Pin
	fail bool
}

func (f *failPin) Out(l gpio.Level) error {
	if f.fail {
		return errors.New("injected error")
	}
	return nil
}
