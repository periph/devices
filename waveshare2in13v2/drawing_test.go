// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

import (
	"bytes"
	"image"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"periph.io/x/devices/v3/ssd1306/image1bit"
)

func TestDrawSpec(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts drawOpts
		want drawSpec
	}{
		{
			name: "empty",
		},
		{
			name: "smaller than display",
			opts: drawOpts{
				devSize: image.Pt(100, 200),
				buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 120, 210)),
				dstRect: image.Rect(17, 4, 25, 8),
			},
			want: drawSpec{
				DstRect: image.Rect(17, 4, 25, 8),
				MemRect: image.Rect(2, 4, 4, 8),
			},
		},
		{
			name: "larger than display",
			opts: drawOpts{
				devSize: image.Pt(100, 200),
				buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 100, 200)),
				dstRect: image.Rect(-20, 50, 125, 300),
			},
			want: drawSpec{
				DstRect: image.Rect(0, 50, 100, 200),
				MemRect: image.Rect(0, 50, 13, 200),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.opts.spec()

			if diff := cmp.Diff(got, tc.want, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("spec() difference (-got +want):\n%s", diff)
			}
		})
	}
}

func TestDrawImage(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts drawOpts
		want []record
	}{
		{
			name: "empty",
			opts: drawOpts{
				src: &image.Uniform{image1bit.On},
			},
		},
		{
			name: "partial",
			opts: drawOpts{
				cmd:     writeRAMBW,
				devSize: image.Pt(64, 64),
				buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 64, 64)),
				dstRect: image.Rect(17, 4, 41, 8),
				src:     &image.Uniform{image1bit.On},
				srcPts:  image.Pt(0, 0),
			},
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{2, 6 - 1}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{4, 0, 8 - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{2}},
				{cmd: setRAMYAddressCounter, data: []byte{4, 0}},
				{
					cmd:  writeRAMBW,
					data: bytes.Repeat([]byte{0x7f, 0xff, 0xff, 0x80}, 4),
				},
			},
		},
		{
			name: "full",
			opts: drawOpts{
				cmd:     writeRAMBW,
				devSize: image.Pt(80, 120),
				buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 80, 120)),
				dstRect: image.Rect(0, 0, 80, 120),
				src:     &image.Uniform{image1bit.On},
				srcPts:  image.Pt(33, 44),
			},
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{0, 10 - 1}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{0, 0, 120 - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{0}},
				{cmd: setRAMYAddressCounter, data: []byte{0, 0}},
				{
					cmd:  writeRAMBW,
					data: bytes.Repeat([]byte{0xff}, 80/8*120),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			drawImage(&got, &tc.opts)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("drawImage() difference (-got +want):\n%s", diff)
			}
		})
	}
}

func TestClearDisplay(t *testing.T) {
	for _, tc := range []struct {
		name  string
		size  image.Point
		color image1bit.Bit
		want  []record
	}{
		{
			name: "empty",
		},
		{
			name:  "off",
			size:  image.Pt(100, 10),
			color: image1bit.Off,
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{0, (100+7)/8 - 1}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{0, 0, 10 - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{0}},
				{cmd: setRAMYAddressCounter, data: []byte{0, 0}},
				{
					cmd:  writeRAMBW,
					data: bytes.Repeat([]byte{0}, 13*10),
				},
			},
		},
		{
			name:  "on",
			size:  image.Pt(32, 20),
			color: image1bit.On,
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{0, 32/8 - 1}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{0, 0, 20 - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{0}},
				{cmd: setRAMYAddressCounter, data: []byte{0, 0}},
				{
					cmd:  writeRAMBW,
					data: bytes.Repeat([]byte{0xff}, 4*20),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			clearDisplay(&got, tc.size, tc.color)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("clearDisplay() difference (-got +want):\n%s", diff)
			}
		})
	}
}
