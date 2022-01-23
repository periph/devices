// Copyright 2022 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

import (
	"image"
	"testing"

	"github.com/google/go-cmp/cmp"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpiotest"
	"periph.io/x/conn/v3/spi/spitest"
	"periph.io/x/devices/v3/ssd1306/image1bit"
)

func TestNew(t *testing.T) {
	for _, tc := range []struct {
		name             string
		opts             Opts
		wantString       string
		wantBounds       image.Rectangle
		wantBufferBounds image.Rectangle
	}{
		{
			name:       "empty",
			wantString: "epd.Dev{playback, (0), Width: 0, Height: 0}",
		},
		{
			name:             "EPD2in13v2",
			opts:             EPD2in13v2,
			wantBounds:       image.Rect(0, 0, 122, 250),
			wantBufferBounds: image.Rect(0, 0, 128, 250),
			wantString:       "epd.Dev{playback, (0), Width: 122, Height: 250}",
		},
		{
			name: "EPD2in13v2, top right",
			opts: func() Opts {
				opts := EPD2in13v2
				opts.Origin = TopRight
				return opts
			}(),
			wantBounds:       image.Rect(0, 0, 250, 122),
			wantBufferBounds: image.Rect(0, 0, 250, 128),
			wantString:       "epd.Dev{playback, (0), Width: 250, Height: 122}",
		},
		{
			name: "EPD2in13v2, bottom left",
			opts: func() Opts {
				opts := EPD2in13v2
				opts.Origin = BottomLeft
				return opts
			}(),
			wantBounds:       image.Rect(0, 0, 250, 122),
			wantBufferBounds: image.Rect(0, 0, 250, 128),
			wantString:       "epd.Dev{playback, (0), Width: 250, Height: 122}",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dev, err := New(&spitest.Playback{}, &gpiotest.Pin{}, &gpiotest.Pin{}, &gpiotest.Pin{}, &gpiotest.Pin{
				EdgesChan: make(chan gpio.Level, 1),
			}, &tc.opts)
			if err != nil {
				t.Errorf("New() failed: %v", err)
			}

			if diff := cmp.Diff(dev.String(), tc.wantString); diff != "" {
				t.Errorf("String() difference (-got +want):\n%s", diff)
			}

			if diff := cmp.Diff(dev.Bounds(), tc.wantBounds); diff != "" {
				t.Errorf("Bounds() difference (-got +want):\n%s", diff)
			}

			if diff := cmp.Diff(dev.buffer.Bounds(), tc.wantBufferBounds); diff != "" {
				t.Errorf("buffer.Bounds() difference (-got +want):\n%s", diff)
			}

			if !dev.buffer.Bounds().Empty() {
				for _, pos := range []image.Point{
					image.Pt(0, 0),
					image.Pt(dev.buffer.Bounds().Max.X-1, 0),
					image.Pt(dev.buffer.Bounds().Max.X-1, dev.buffer.Bounds().Max.Y-1),
					image.Pt(0, dev.buffer.Bounds().Max.Y-1),
					image.Pt(dev.buffer.Bounds().Dx()/2, dev.buffer.Bounds().Dy()/2),
				} {
					if diff := cmp.Diff(dev.buffer.BitAt(pos.X, pos.Y), image1bit.On); diff != "" {
						t.Errorf("buffer.BitAt(%v) difference (-got +want):\n%s", pos, diff)
					}
				}
			}
		})
	}
}
