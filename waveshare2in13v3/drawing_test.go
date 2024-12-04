// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v3

import (
	"bytes"
	"image"
	"image/draw"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"periph.io/x/devices/v3/ssd1306/image1bit"
)

func checkRectCanon(t *testing.T, got image.Rectangle) {
	if diff := cmp.Diff(got, got.Canon()); diff != "" {
		t.Errorf("Rectangle is not canonical (-got +want):\n%s", diff)
	}
}

func TestDrawSpec(t *testing.T) {
	type testCase struct {
		name string
		opts drawOpts
		want drawSpec
	}

	for _, tc := range []testCase{
		{
			name: "empty",
			opts: drawOpts{
				buffer: image1bit.NewVerticalLSB(image.Rectangle{}),
			},
		},
		{
			name: "smaller than display",
			opts: drawOpts{
				devSize: image.Pt(100, 200),
				buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 120, 210)),
				dstRect: image.Rect(17, 4, 25, 8),
			},
			want: drawSpec{
				bufferDstRect: image.Rect(17, 4, 25, 8),
				memDstRect:    image.Rect(17, 4, 25, 8),
				memRect:       image.Rect(2, 4, 4, 8),
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
				bufferDstRect: image.Rect(0, 50, 100, 200),
				memDstRect:    image.Rect(0, 50, 100, 200),
				memRect:       image.Rect(0, 50, 13, 200),
			},
		},
		func() testCase {
			tc := testCase{
				name: "origin top left full",
				opts: drawOpts{
					devSize: image.Pt(48, 96),
					origin:  TopLeft,
					buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 6*8, 12*8)),
					dstRect: image.Rect(0, 0, 48, 96),
				},
			}

			tc.want.bufferDstRect.Max = image.Pt(48, 96)
			tc.want.memDstRect.Max = image.Pt(48, 96)
			tc.want.memRect.Max = image.Pt(6, 96)

			return tc
		}(),
		func() testCase {
			tc := testCase{
				name: "origin top right, empty dest",
				opts: drawOpts{
					devSize: image.Pt(105, 50),
					origin:  TopRight,
					buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 12*8, 8*8)),
				},
			}

			tc.want.bufferDstOffset.Y = tc.opts.buffer.Bounds().Dy() - tc.opts.devSize.Y

			return tc
		}(),
		func() testCase {
			tc := testCase{
				name: "origin top right",
				opts: drawOpts{
					devSize: image.Pt(100, 50),
					origin:  TopRight,
					buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 12*8, 8*8)),
					dstRect: image.Rect(0, 0, 20, 30),
				},
			}

			tc.want.bufferDstOffset.Y = tc.opts.buffer.Bounds().Dy() - tc.opts.devSize.Y
			tc.want.bufferDstRect = image.Rectangle{
				Min: tc.want.bufferDstOffset,
				Max: image.Point{
					X: tc.opts.dstRect.Max.X,
					Y: tc.want.bufferDstOffset.Y + tc.opts.dstRect.Max.Y,
				},
			}
			tc.want.memDstRect = image.Rectangle{
				Min: image.Point{
					X: tc.opts.devSize.Y - tc.opts.dstRect.Max.Y,
				},
				Max: image.Point{
					X: tc.opts.devSize.Y,
					Y: tc.opts.dstRect.Max.X,
				},
			}
			tc.want.memRect = image.Rectangle{
				Min: image.Pt(2, tc.want.memDstRect.Min.Y),
				Max: image.Pt(7, tc.want.memDstRect.Max.Y),
			}

			return tc
		}(),
		func() testCase {
			tc := testCase{
				name: "origin top right full",
				opts: drawOpts{
					devSize: image.Pt(48, 96),
					origin:  TopRight,
					buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 6*8, 12*8)),
					dstRect: image.Rect(0, 0, 48, 96),
				},
			}

			tc.want.bufferDstRect.Max = image.Pt(48, 96)
			tc.want.memDstRect.Max = image.Pt(96, 48)
			tc.want.memRect.Max = image.Pt(12, 48)

			return tc
		}(),
		func() testCase {
			tc := testCase{
				name: "origin top right with offset",
				opts: drawOpts{
					devSize: image.Pt(101, 83),
					origin:  TopRight,
					buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 14*8, 11*8)),
					dstRect: image.Rect(9, 17, 19, 27),
				},
			}

			tc.want.bufferDstOffset.Y = tc.opts.buffer.Bounds().Dy() - tc.opts.devSize.Y
			tc.want.bufferDstRect = image.Rectangle{
				Min: image.Point{
					X: tc.opts.dstRect.Min.X,
					Y: tc.want.bufferDstOffset.Y + tc.opts.dstRect.Min.Y,
				},
				Max: image.Point{
					X: tc.opts.dstRect.Max.X,
					Y: tc.want.bufferDstOffset.Y + tc.opts.dstRect.Max.Y,
				},
			}
			tc.want.memDstRect = image.Rectangle{
				Min: image.Point{
					X: tc.opts.devSize.Y - tc.opts.dstRect.Max.Y,
					Y: tc.opts.dstRect.Min.X,
				},
				Max: image.Point{
					X: tc.opts.devSize.Y - tc.opts.dstRect.Min.Y,
					Y: tc.opts.dstRect.Max.X,
				},
			}
			tc.want.memRect = image.Rectangle{
				Min: image.Pt(7, tc.want.memDstRect.Min.Y),
				Max: image.Pt(9, tc.want.memDstRect.Max.Y),
			}

			return tc
		}(),
		func() testCase {
			tc := testCase{
				name: "origin bottom right full",
				opts: drawOpts{
					devSize: image.Pt(48, 96),
					origin:  BottomRight,
					buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 6*8, 12*8)),
					dstRect: image.Rect(0, 0, 48, 96),
				},
			}

			tc.want.bufferDstRect.Max = image.Pt(48, 96)
			tc.want.memDstRect.Max = image.Pt(48, 96)
			tc.want.memRect.Max = image.Pt(6, 96)

			return tc
		}(),
		func() testCase {
			tc := testCase{
				name: "origin bottom right with offset",
				opts: drawOpts{
					devSize: image.Pt(75, 103),
					origin:  BottomRight,
					buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 10*8, 14*8)),
					dstRect: image.Rect(9, 17, 19, 49),
				},
			}

			tc.want.bufferDstOffset = image.Point{
				X: tc.opts.buffer.Bounds().Dx() - tc.opts.devSize.X,
				Y: tc.opts.buffer.Bounds().Dy() - tc.opts.devSize.Y,
			}
			tc.want.bufferDstRect = image.Rectangle{
				Min: image.Point{
					X: tc.want.bufferDstOffset.X + tc.opts.dstRect.Min.X,
					Y: tc.want.bufferDstOffset.Y + tc.opts.dstRect.Min.Y,
				},
				Max: image.Point{
					X: tc.want.bufferDstOffset.X + tc.opts.dstRect.Max.X,
					Y: tc.want.bufferDstOffset.Y + tc.opts.dstRect.Max.Y,
				},
			}
			tc.want.memDstRect = image.Rectangle{
				Min: image.Point{
					X: tc.opts.devSize.X - tc.opts.dstRect.Max.X,
					Y: tc.opts.devSize.Y - tc.opts.dstRect.Max.Y,
				},
				Max: image.Point{
					X: tc.opts.devSize.X - tc.opts.dstRect.Min.X,
					Y: tc.opts.devSize.Y - tc.opts.dstRect.Min.Y,
				},
			}
			tc.want.memRect = image.Rectangle{
				Min: image.Pt(7, tc.want.memDstRect.Min.Y),
				Max: image.Pt(9, tc.want.memDstRect.Max.Y),
			}

			return tc
		}(),
		func() testCase {
			tc := testCase{
				name: "origin bottom left full",
				opts: drawOpts{
					devSize: image.Pt(48, 96),
					origin:  BottomLeft,
					buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 6*8, 12*8)),
					dstRect: image.Rect(0, 0, 48, 96),
				},
			}

			tc.want.bufferDstRect.Max = image.Pt(48, 96)
			tc.want.memDstRect.Max = image.Pt(96, 48)
			tc.want.memRect.Max = image.Pt(12, 48)

			return tc
		}(),
		func() testCase {
			tc := testCase{
				name: "origin bottom left with offset",
				opts: drawOpts{
					devSize: image.Pt(101, 81),
					origin:  BottomLeft,
					buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 15*8, 11*8)),
					dstRect: image.Rect(9, 17, 21, 49),
				},
			}

			tc.want.bufferDstOffset = image.Point{
				X: tc.opts.buffer.Bounds().Dx() - tc.opts.devSize.X,
				Y: tc.opts.buffer.Bounds().Dy() - tc.opts.devSize.Y,
			}
			tc.want.bufferDstRect = image.Rectangle{
				Min: image.Point{
					X: tc.want.bufferDstOffset.X + tc.opts.dstRect.Min.X,
					Y: tc.want.bufferDstOffset.Y + tc.opts.dstRect.Min.Y,
				},
				Max: image.Point{
					X: tc.want.bufferDstOffset.X + tc.opts.dstRect.Max.X,
					Y: tc.want.bufferDstOffset.Y + tc.opts.dstRect.Max.Y,
				},
			}
			tc.want.memDstRect = image.Rectangle{
				Min: image.Point{
					X: tc.opts.dstRect.Min.Y,
					Y: tc.opts.devSize.X - tc.opts.dstRect.Max.X,
				},
				Max: image.Point{
					X: tc.opts.dstRect.Max.Y,
					Y: tc.opts.devSize.X - tc.opts.dstRect.Min.X,
				},
			}
			tc.want.memRect = image.Rectangle{
				Min: image.Pt(2, tc.want.memDstRect.Min.Y),
				Max: image.Pt(7, tc.want.memDstRect.Max.Y),
			}

			return tc
		}(),
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkRectCanon(t, tc.opts.dstRect)

			got := tc.opts.spec()

			checkRectCanon(t, got.bufferDstRect)
			checkRectCanon(t, got.memRect)

			if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(drawSpec{})); diff != "" {
				t.Errorf("spec() difference (-got +want):\n%s", diff)
			}
		})
	}
}

func TestSendImage(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  byte
		opts drawOpts
		want []record
	}{
		{
			name: "empty",
			opts: drawOpts{
				buffer: image1bit.NewVerticalLSB(image.Rectangle{}),
			},
		},
		{
			name: "partial",
			cmd:  writeRAMBW,
			opts: drawOpts{
				devSize: image.Pt(64, 64),
				dstRect: image.Rect(16, 20, 32, 40),
				buffer:  image1bit.NewVerticalLSB(image.Rect(0, 0, 64, 64)),
			},
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{2, 4 - 1}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{20, 0, 40 - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{2}},
				{cmd: setRAMYAddressCounter, data: []byte{20, 0}},
				{
					cmd:  writeRAMBW,
					data: bytes.Repeat([]byte{0}, 2*(30-10)),
				},
			},
		},
		{
			name: "partial non-aligned",
			cmd:  writeRAMRed,
			opts: drawOpts{
				devSize: image.Pt(100, 64),
				dstRect: image.Rect(17, 4, 41, 8),
				buffer: func() *image1bit.VerticalLSB {
					img := image1bit.NewVerticalLSB(image.Rect(0, 0, 64, 64))
					draw.Src.Draw(img, image.Rect(17, 4, 41, 8), &image.Uniform{image1bit.On}, image.Point{})
					return img
				}(),
			},
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{2, 6 - 1}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{4, 0, 8 - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{2}},
				{cmd: setRAMYAddressCounter, data: []byte{4, 0}},
				{
					cmd:  writeRAMRed,
					data: bytes.Repeat([]byte{0x7f, 0xff, 0xff, 0x80}, 4),
				},
			},
		},
		{
			name: "full",
			cmd:  writeRAMBW,
			opts: drawOpts{
				devSize: image.Pt(80, 120),
				dstRect: image.Rect(0, 0, 80, 120),
				buffer: func() *image1bit.VerticalLSB {
					img := image1bit.NewVerticalLSB(image.Rect(0, 0, 80, 120))
					draw.Src.Draw(img, image.Rect(0, 0, 80, 120), &image.Uniform{image1bit.On}, image.Point{})
					return img
				}(),
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
		{
			name: "top left",
			cmd:  writeRAMBW,
			opts: drawOpts{
				devSize: image.Pt(100, 40),
				dstRect: image.Rect(20, 17-5, 44, 29+5),
				origin:  TopLeft,
				buffer: func() *image1bit.VerticalLSB {
					img := image1bit.NewVerticalLSB(image.Rect(0, 0, 100, 40))
					draw.Src.Draw(img, image.Rect(20, 17, 44, 29), &image.Uniform{image1bit.On}, image.Point{})
					return img
				}(),
			},
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{2, 5}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{17 - 5, 0, 29 + 5 - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{2}},
				{cmd: setRAMYAddressCounter, data: []byte{12, 0}},
				{
					cmd: writeRAMBW,
					data: append(
						append(
							bytes.Repeat([]byte{0x00, 0x00, 0x00, 0x00}, 5),
							bytes.Repeat([]byte{0x0f, 0xff, 0xff, 0xf0}, 29-17)...),
						bytes.Repeat([]byte{0x00, 0x00, 0x00, 0x00}, 5)...,
					),
				},
			},
		},
		{
			name: "top right",
			cmd:  writeRAMBW,
			opts: drawOpts{
				devSize: image.Pt(64, 48),
				dstRect: image.Rect(15-5, 16, 30+5, 40),
				origin:  TopRight,
				buffer: func() *image1bit.VerticalLSB {
					img := image1bit.NewVerticalLSB(image.Rect(0, 0, 64, 48))
					draw.Src.Draw(img, image.Rect(15, 20, 30, 36), &image.Uniform{image1bit.On}, image.Point{})
					return img
				}(),
			},
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{(48 - 40) / 8, ((48 - 16 + 7) / 8) - 1}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{15 - 5, 0, (30 + 5) - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{1}},
				{cmd: setRAMYAddressCounter, data: []byte{10, 0}},
				{
					cmd: writeRAMBW,
					data: append(
						append(
							bytes.Repeat([]byte{0x00, 0x00, 0x00}, 5),
							bytes.Repeat([]byte{0x0f, 0xff, 0xf0}, 30-15)...),
						bytes.Repeat([]byte{0x00, 0x00, 0x00}, 5)...,
					),
				},
			},
		},
		{
			name: "top right uneven size",
			cmd:  writeRAMBW,
			opts: drawOpts{
				devSize: image.Pt(61, 53),
				dstRect: image.Rect(15-5, 16, 30+5, 36),
				origin:  TopRight,
				buffer: func() *image1bit.VerticalLSB {
					img := image1bit.NewVerticalLSB(image.Rect(0, 0, 64, 99))
					yoff := img.Bounds().Dy() - 53 + 1
					draw.Src.Draw(img, image.Rect(15, yoff+16, 30, yoff+32), &image.Uniform{image1bit.On}, image.Point{})
					return img
				}(),
			},
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{(53 - 32) / 8, ((53 - 16 + 7) / 8) - 1}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{15 - 5, 0, (30 + 5) - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{2}},
				{cmd: setRAMYAddressCounter, data: []byte{10, 0}},
				{
					cmd: writeRAMBW,
					data: append(
						append(
							bytes.Repeat([]byte{0x00, 0x00, 0x00}, 5),
							bytes.Repeat([]byte{0x0f, 0xff, 0xf0}, 30-15)...),
						bytes.Repeat([]byte{0x00, 0x00, 0x00}, 5)...,
					),
				},
			},
		},
		{
			name: "bottom right",
			cmd:  writeRAMRed,
			opts: drawOpts{
				devSize: image.Pt(64, 48),
				dstRect: image.Rect(16, 15-5, 40, 30+5),
				origin:  BottomRight,
				buffer: func() *image1bit.VerticalLSB {
					img := image1bit.NewVerticalLSB(image.Rect(0, 0, 64, 48))
					draw.Src.Draw(img, image.Rect(20, 15, 36, 30), &image.Uniform{image1bit.On}, image.Point{})
					return img
				}(),
			},
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{(64 - 40) / 8, ((64 - 16 + 7) / 8) - 1}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{48 - (30 + 5), 0, 48 - (15 - 5) - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{3}},
				{cmd: setRAMYAddressCounter, data: []byte{48 - (30 + 5), 0}},
				{
					cmd: writeRAMRed,
					data: append(
						append(
							bytes.Repeat([]byte{0x00, 0x00, 0x00}, 5),
							bytes.Repeat([]byte{0x0f, 0xff, 0xf0}, 30-15)...),
						bytes.Repeat([]byte{0x00, 0x00, 0x00}, 5)...,
					),
				},
			},
		},
		{
			name: "bottom left",
			cmd:  writeRAMRed,
			opts: drawOpts{
				devSize: image.Pt(64, 48),
				dstRect: image.Rect(15-5, 16, 30+5, 40),
				origin:  BottomLeft,
				buffer: func() *image1bit.VerticalLSB {
					img := image1bit.NewVerticalLSB(image.Rect(0, 0, 64, 48))
					draw.Src.Draw(img, image.Rect(15, 20, 30, 36), &image.Uniform{image1bit.On}, image.Point{})
					return img
				}(),
			},
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{16 / 8, ((40 + 7) / 8) - 1}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{64 - (30 + 5), 0, 64 - (15 - 5) - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{2}},
				{cmd: setRAMYAddressCounter, data: []byte{64 - (30 + 5), 0}},
				{
					cmd: writeRAMRed,
					data: append(
						append(
							bytes.Repeat([]byte{0x00, 0x00, 0x00}, 5),
							bytes.Repeat([]byte{0x0f, 0xff, 0xf0}, 30-15)...),
						bytes.Repeat([]byte{0x00, 0x00, 0x00}, 5)...,
					),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			checkRectCanon(t, tc.opts.dstRect)

			spec := tc.opts.spec()

			tc.opts.sendImage(&got, tc.cmd, &spec)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("sendImage() difference (-got +want):\n%s", diff)
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
				buffer: image1bit.NewVerticalLSB(image.Rectangle{}),
			},
		},
		{
			name: "partial",
			opts: drawOpts{
				commands: []byte{writeRAMBW},
				devSize:  image.Pt(64, 64),
				buffer:   image1bit.NewVerticalLSB(image.Rect(0, 0, 64, 64)),
				dstRect:  image.Rect(17, 4, 41, 8),
				src:      &image.Uniform{image1bit.On},
				srcPts:   image.Pt(0, 0),
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
				{cmd: displayUpdateControl2, data: []byte{0x0f}},
				{cmd: masterActivation},
			},
		},
		{
			name: "full",
			opts: drawOpts{
				commands: []byte{writeRAMRed},
				devSize:  image.Pt(80, 120),
				buffer:   image1bit.NewVerticalLSB(image.Rect(0, 0, 80, 120)),
				dstRect:  image.Rect(0, 0, 80, 120),
				src:      &image.Uniform{image1bit.On},
				srcPts:   image.Pt(33, 44),
			},
			want: []record{
				{cmd: dataEntryModeSetting, data: []byte{0x3}},
				{cmd: setRAMXAddressStartEndPosition, data: []byte{0, 10 - 1}},
				{cmd: setRAMYAddressStartEndPosition, data: []byte{0, 0, 120 - 1, 0}},
				{cmd: setRAMXAddressCounter, data: []byte{0}},
				{cmd: setRAMYAddressCounter, data: []byte{0, 0}},
				{
					cmd:  writeRAMRed,
					data: bytes.Repeat([]byte{0xff}, 80/8*120),
				},
				{cmd: displayUpdateControl2, data: []byte{0x0f}},
				{cmd: masterActivation},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			drawImage(&got, &tc.opts, true)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("drawImage() difference (-got +want):\n%s", diff)
			}
		})
	}
}
