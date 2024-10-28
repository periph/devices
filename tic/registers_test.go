// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package tic

import (
	"bytes"
	"testing"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2ctest"
)

func TestGetVar8(t *testing.T) {
	for _, test := range []struct {
		name      string
		offset    offset
		ops       []i2ctest.IO
		want      uint8
		expectErr bool
	}{
		{
			name:   "success",
			offset: 0xAA,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0xAA}},
				{Addr: I2CAddr, R: []byte{0xAB}},
			},
			want:      0xAB,
			expectErr: false,
		},
		{
			name:   "no bytes received",
			offset: 0xAA,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0xAA}},
				{Addr: I2CAddr, R: []byte{}},
			},
			expectErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			dev := Dev{
				c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
				variant: Tic36v4,
			}

			got, err := dev.getVar8(test.offset)
			if test.expectErr {
				if err == nil {
					t.Fatalf("expected error, got: %v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if got != test.want {
					t.Fatalf("wanted: %d, got: %d", test.want, got)
				}
			}
		})
	}
}

func TestGetVar16(t *testing.T) {
	for _, test := range []struct {
		name      string
		offset    offset
		ops       []i2ctest.IO
		want      uint16
		expectErr bool
	}{
		{
			name:   "success",
			offset: 0xAA,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0xAA}},
				{Addr: I2CAddr, R: []byte{0xCD, 0xAB}},
			},
			want:      0xABCD,
			expectErr: false,
		},
		{
			name:   "no bytes received",
			offset: 0xAA,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0xAA}},
				{Addr: I2CAddr, R: []byte{}},
			},
			expectErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			dev := Dev{
				c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
				variant: Tic36v4,
			}

			got, err := dev.getVar16(test.offset)
			if test.expectErr {
				if err == nil {
					t.Fatalf("expected error, got: %v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if got != test.want {
					t.Fatalf("wanted: %d, got: %d", test.want, got)
				}
			}
		})
	}
}

func TestGetVar32(t *testing.T) {
	for _, test := range []struct {
		name      string
		offset    offset
		ops       []i2ctest.IO
		want      uint32
		expectErr bool
	}{
		{
			name:   "success",
			offset: 0xAA,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0xAA}},
				{Addr: I2CAddr, R: []byte{0xEF, 0xBE, 0xAD, 0xDE}},
			},
			want:      0xDEADBEEF,
			expectErr: false,
		},
		{
			name:   "no bytes received",
			offset: 0xAA,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xA1, 0xAA}},
				{Addr: I2CAddr, R: []byte{}},
			},
			expectErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			dev := Dev{
				c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
				variant: Tic36v4,
			}

			got, err := dev.getVar32(test.offset)
			if test.expectErr {
				if err == nil {
					t.Fatalf("expected error, got: %v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if got != test.want {
					t.Fatalf("wanted: %d, got: %d", test.want, got)
				}
			}
		})
	}
}

func TestCommandQuick(t *testing.T) {
	const cmd = 0xAA

	b := i2ctest.Playback{
		Ops: []i2ctest.IO{
			{Addr: I2CAddr, W: []byte{cmd}},
		},
		DontPanic: true,
	}
	defer b.Close()

	dev := Dev{
		c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
		variant: Tic36v4,
	}

	err := dev.commandQuick(cmd)
	if err != nil {
		t.Error(err)
	}
}

func TestCommandW7(t *testing.T) {
	for _, test := range []struct {
		name string
		cmd  command
		val  uint8
		ops  []i2ctest.IO
	}{
		{
			name: "success",
			cmd:  0xAA,
			val:  0x0B,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xAA, 0x0B}},
			},
		},
		{
			name: "val MSB truncated",
			cmd:  0xAA,
			val:  0b1111_1111,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xAA, 0b0111_1111}},
			},
		},
	} {
		b := i2ctest.Playback{
			Ops:       test.ops,
			DontPanic: true,
		}
		defer b.Close()

		dev := Dev{
			c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
			variant: Tic36v4,
		}

		err := dev.commandW7(test.cmd, test.val)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCommandW32(t *testing.T) {
	const (
		cmd        = 0xAA
		val uint32 = 0xBBCCDDEE
	)

	b := i2ctest.Playback{
		Ops: []i2ctest.IO{
			{Addr: I2CAddr, W: []byte{0xAA, 0xEE, 0xDD, 0xCC, 0xBB}},
		},
		DontPanic: true,
	}
	defer b.Close()

	dev := Dev{
		c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
		variant: Tic36v4,
	}

	err := dev.commandW32(cmd, val)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSegment(t *testing.T) {
	for _, test := range []struct {
		name      string
		cmd       command
		offset    offset
		length    uint
		want      []uint8
		ops       []i2ctest.IO
		expectErr bool
	}{
		{
			name:   "read 1 byte",
			cmd:    0xAA,
			offset: 0xBB,
			length: 1,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xAA, 0xBB}},
				{Addr: I2CAddr, R: []byte{0xCC}},
			},
			want:      []byte{0xCC},
			expectErr: false,
		},
		{
			name:   "read 4 bytes",
			cmd:    0xAA,
			offset: 0xBB,
			length: 4,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xAA, 0xBB}},
				{Addr: I2CAddr, R: []byte{0xCC, 0xDD, 0xEE, 0xFF}},
			},
			want:      []byte{0xCC, 0xDD, 0xEE, 0xFF},
			expectErr: false,
		},
		{
			name:   "invalid length",
			cmd:    0xAA,
			offset: 0xBB,
			length: 0,
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{0xAA, 0xBB}},
			},
			expectErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			defer b.Close()

			dev := Dev{
				c:       &i2c.Dev{Bus: &b, Addr: I2CAddr},
				variant: Tic36v4,
			}

			got, err := dev.getSegment(test.cmd, test.offset, test.length)
			if test.expectErr {
				if err == nil {
					t.Fatalf("expected error, got: %v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, test.want) {
					t.Fatalf("wanted: %d, got: %d", test.want, got)
				}
			}
		})
	}
}
