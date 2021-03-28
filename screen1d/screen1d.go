// Copyright 2017 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package screen1d implements a 1D display.Drawer that outputs to terminal
// (stdout) using ANSI color codes.
//
// Useful while you are waiting for your super nice APA-102 LED strip to come
// by mail.
package screen1d

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/maruel/ansi256"
	"github.com/mattn/go-colorable"
	"periph.io/x/conn/v3/display"
)

// Opts represents the options available for this display.
type Opts struct {
	X       int
	Palette *ansi256.Palette

	_ struct{}
}

// Dev is a 1D LED strip emulator that outputs to the console.
type Dev struct {
	w       io.Writer
	l       int
	palette ansi256.Palette

	pixels []byte
	buf    bytes.Buffer
}

// New returns a Dev that displays at the console.
//
// Permits to do local testing of LEDs animation.
func New(opts *Opts) *Dev {
	p := opts.Palette
	if p == nil {
		p = ansi256.Default
	}
	d := &Dev{
		w:       colorable.NewColorableStdout(),
		l:       opts.X,
		palette: *p,
		pixels:  make([]byte, 3*opts.X),
	}
	return d
}

func (d *Dev) String() string {
	return "Screen1D"
}

// Halt implements conn.Resource.
//
// It clears the display so it is not corrupted.
func (d *Dev) Halt() error {
	_, err := d.w.Write([]byte("\n\033[0m"))
	if err != nil {
		return err
	}
	return nil
}

// Write accepts a stream of raw RGB pixels and writes it to the console.
func (d *Dev) Write(pixels []byte) (int, error) {
	if len(pixels)%3 != 0 {
		return 0, errors.New("invalid RGB stream length")
	}
	copy(d.pixels, pixels)
	return d.refresh()
}

// ColorModel implements display.Drawer.
func (d *Dev) ColorModel() color.Model {
	return color.NRGBAModel
}

// Bounds implements display.Drawer.
func (d *Dev) Bounds() image.Rectangle {
	return image.Rectangle{Max: image.Point{X: d.l, Y: 1}}
}

// Draw implements display.Drawer.
func (d *Dev) Draw(r image.Rectangle, src image.Image, sp image.Point) error {
	r = r.Intersect(d.Bounds())
	srcR := src.Bounds()
	srcR.Min = srcR.Min.Add(sp)
	if dX := r.Dx(); dX < srcR.Dx() {
		srcR.Max.X = srcR.Min.X + dX
	}
	if dY := r.Dy(); dY < srcR.Dy() {
		srcR.Max.Y = srcR.Min.Y + dY
	}
	// TODO(maruel): Allow non-full screen drawing.
	// Generic version.
	deltaX3 := 3 * (r.Min.X - srcR.Min.X)
	for sX := srcR.Min.X; sX < srcR.Max.X; sX++ {
		r16, g16, b16, _ := src.At(sX, srcR.Min.Y).RGBA()
		dX3 := 3*sX + deltaX3
		d.pixels[dX3] = byte(r16 >> 8)
		d.pixels[dX3+1] = byte(g16 >> 8)
		d.pixels[dX3+2] = byte(b16 >> 8)
	}
	_, err := d.refresh()
	return err
}

func (d *Dev) refresh() (int, error) {
	// This code is designed to minimize the amount of memory allocated per call.
	d.buf.Reset()
	_, _ = d.buf.WriteString("\r\033[0m")
	for i := 0; i < len(d.pixels)/3; i++ {
		c := color.NRGBA{d.pixels[3*i], d.pixels[3*i+1], d.pixels[3*i+2], 255}
		_, _ = io.WriteString(&d.buf, d.palette.Block(c))
	}
	_, _ = d.buf.WriteString("\033[0m ")
	_, err := d.buf.WriteTo(d.w)
	return len(d.pixels), err
}

var _ display.Drawer = &Dev{}
var _ fmt.Stringer = &Dev{}
