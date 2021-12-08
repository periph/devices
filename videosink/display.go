// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package videosink provides a display driver implementing an HTTP request
// handler. Client requests get an initial snapshot of the graphics buffer and
// are updated further on every change.
//
// The primary use case is the development of display outputs on a host
// machine. Additionally devices with network connectivity can use this driver
// to provide a copy of their local display via a web interface.
//
// The protocol used is "MJPEG" (https://en.wikipedia.org/wiki/Motion_JPEG)
// which is often used by IP cameras. Because of its better suitability for
// computer-drawn graphics the PNG image format is used by default. JPEG as
// a format can be selected via Options.Format or using the "format" URL
// parameter.
package videosink

import (
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"
	"sync"
	"time"

	"periph.io/x/conn/v3/display"
)

const defaultJPEGQuality = 95
const defaultKeepAliveInterval = time.Minute
const defaultMinFrameInterval = time.Second / 15

// Options for videosink devices.
type Options struct {
	// Width and height of the image buffer.
	Width, Height int

	// Format specifies the image format to send to clients.
	Format ImageFormat

	// JPEG controls options for the JPEG encoder.
	JPEG jpeg.Options

	// PNG controls options for the PNG encoder.
	PNG struct {
		// CompressionLevel is the amount of compression applied by the PNG
		// encoder. Defaults to png.DefaultCompression.
		CompressionLevel png.CompressionLevel
	}

	// MinFrameInterval is the amount of time which needs to pass before
	// sending a new image, thus implementing rate-limiting. Defaults to 15
	// frames per second.
	MinFrameInterval time.Duration

	// KeepAliveInterval is the amount of time after which to send a new
	// image regardless of whether any changes have been made. Defaults to
	// once per minute.
	KeepAliveInterval time.Duration
}

// Display is a virtual device receiving drawing operations and sending
// a stream of images to all connected HTTP clients.
type Display struct {
	defaultFormat       ImageFormat
	jpegOptions         jpeg.Options
	pngCompressionLevel png.CompressionLevel
	keepAliveInterval   time.Duration
	minFrameInterval    time.Duration

	mu       sync.Mutex
	buffer   *image.RGBA
	clients  map[*client]struct{}
	snapshot map[imageConfig][]byte
}

var _ display.Drawer = (*Display)(nil)
var _ http.Handler = (*Display)(nil)

// New creates a new videosink device instance.
func New(opt *Options) *Display {
	buffer := image.NewRGBA(image.Rect(0, 0, opt.Width, opt.Height))

	// By default the alpha channel is set to full transparency. The following
	// draw operation makes it opaque.
	draw.Draw(buffer, buffer.Bounds(), image.Black, image.Point{}, draw.Src)

	d := &Display{
		jpegOptions:         opt.JPEG,
		pngCompressionLevel: opt.PNG.CompressionLevel,
		keepAliveInterval:   opt.KeepAliveInterval,
		minFrameInterval:    opt.MinFrameInterval,

		buffer:        buffer,
		clients:       map[*client]struct{}{},
		snapshot:      map[imageConfig][]byte{},
		defaultFormat: opt.Format,
	}

	if d.jpegOptions.Quality == 0 {
		d.jpegOptions.Quality = defaultJPEGQuality
	}

	if d.keepAliveInterval == 0 {
		d.keepAliveInterval = defaultKeepAliveInterval
	}

	if d.minFrameInterval == 0 {
		d.minFrameInterval = defaultMinFrameInterval
	}

	return d
}

// String returns the name of the device.
func (d *Display) String() string {
	return "VideoSink"
}

// Halt implements conn.Resource and terminates all running client requests
// asynchronously.
func (d *Display) Halt() error {
	d.mu.Lock()
	d.terminateClientsLocked()
	d.mu.Unlock()

	return nil
}

// ColorModel implements display.Drawer.
func (d *Display) ColorModel() color.Model {
	return d.buffer.ColorModel()
}

// Bounds implements display.Drawer.
func (d *Display) Bounds() image.Rectangle {
	return d.buffer.Bounds()
}

// Draw implements display.Drawer.
func (d *Display) Draw(dstRect image.Rectangle, src image.Image, srcPts image.Point) error {
	d.mu.Lock()
	draw.Draw(d.buffer, dstRect, src, srcPts, draw.Src)
	d.bufferChangedLocked()
	d.mu.Unlock()

	return nil
}
