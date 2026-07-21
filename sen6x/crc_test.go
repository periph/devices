// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"bytes"
	"testing"
)

func TestCRC8(t *testing.T) {
	cases := []struct {
		name string
		b0   byte
		b1   byte
		want byte
	}{
		{
			// From the datasheet: crc8(0xbeef) = 0x92
			name: "datasheet example",
			b0:   0xbe,
			b1:   0xef,
			want: 0x92,
		},
		{
			// All zeros.
			name: "zeros",
			b0:   0x00,
			b1:   0x00,
			want: 0x81,
		},
		{
			// All ones.
			name: "ones",
			b0:   0xff,
			b1:   0xff,
			want: 0xac,
		},
		{
			// Single bit in b0.
			name: "single bit b0",
			b0:   0x01,
			b1:   0x00,
			want: 0x75,
		},
		{
			// Single bit in b1.
			name: "single bit b1",
			b0:   0x00,
			b1:   0x01,
			want: 0xb0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := crc8(tc.b0, tc.b1)
			if got != tc.want {
				t.Errorf("got %#02x, want %#02x", got, tc.want)
			}
		})
	}
}

func TestValidateAndStripCRC(t *testing.T) {
	cases := []struct {
		name    string
		raw     []byte
		want    []byte
		wantErr bool
	}{
		{
			name: "single word",
			// 0xbeef with CRC 0x92 from datasheet example.
			raw:  []byte{0xbe, 0xef, 0x92},
			want: []byte{0xbe, 0xef},
		},
		{
			name: "two words",
			raw:  []byte{0xbe, 0xef, 0x92, 0x00, 0x00, 0x81},
			want: []byte{0xbe, 0xef, 0x00, 0x00},
		},
		{
			name:    "wrong CRC on first word",
			raw:     []byte{0xbe, 0xef, 0x00},
			wantErr: true,
		},
		{
			name:    "wrong CRC on second word",
			raw:     []byte{0xbe, 0xef, 0x92, 0x00, 0x00, 0x00},
			wantErr: true,
		},
		{
			name:    "length not multiple of 3",
			raw:     []byte{0xbe, 0xef},
			wantErr: true,
		},
		{
			name: "empty input",
			raw:  []byte{},
			want: []byte{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateAndStripCRC(tc.raw)

			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}

			if err == nil && tc.wantErr {
				t.Fatal("expected error, got nil")
			}

			if !tc.wantErr && !bytes.Equal(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestPackWordsWithCRC(t *testing.T) {
	cases := []struct {
		name string
		raw  []uint16
		want []byte
	}{
		{
			name: "single word",
			// 0xbeef with CRC of 0x92 from datasheet example.
			raw:  []uint16{0xbeef},
			want: []byte{0xbe, 0xef, 0x92},
		},
		{
			name: "two words",
			raw:  []uint16{0xbeef, 0x0000},
			want: []byte{0xbe, 0xef, 0x92, 0x00, 0x00, 0x81},
		},
		{
			name: "empty",
			raw:  []uint16{},
			want: []byte{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := packWordsWithCRC(tc.raw)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestPackBytesWithCRC(t *testing.T) {
	cases := []struct {
		name    string
		b       []byte
		want    []byte
		wantErr bool
	}{
		{
			name: "single word",
			// 0xbeef with CRC of 0x92 from datasheet example.
			b:    []byte{0xbe, 0xef},
			want: []byte{0xbe, 0xef, 0x92},
		},
		{
			name: "two words",
			b:    []byte{0xbe, 0xef, 0x00, 0x00},
			want: []byte{0xbe, 0xef, 0x92, 0x00, 0x00, 0x81},
		},
		{
			name: "empty",
			b:    []byte{},
			want: []byte{},
		},
		{
			name:    "odd number of bytes",
			b:       []byte{0xbe},
			wantErr: true,
		},
		{
			name:    "odd number of bytes, more than one",
			b:       []byte{0xbe, 0xef, 0x00},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := packBytesWithCRC(tc.b)

			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}

			if err == nil && tc.wantErr {
				t.Fatal("expected error, got nil")
			}

			if !tc.wantErr && !bytes.Equal(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}
