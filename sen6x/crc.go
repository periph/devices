// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"errors"
	"fmt"
)

const (
	crcInit       byte = 0xff
	crcPolynomial byte = 0x31
)

// crc8 computes the CRC-8-Dallas/Maxim checksum for a pair of data bytes.
func crc8(b0, b1 byte) byte {
	crc := crcInit
	for _, b := range [2]byte{b0, b1} {
		crc ^= b
		for i := 0; i < 8; i++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ crcPolynomial
			} else {
				crc <<= 1
			}
		}
	}

	return crc
}

// validateAndStripCRC validates the CRC byte after each 16-bit word in raw and returns
// the data bytes with CRC bytes removed. raw must have length divisible by 3 (each 16 bit
// word is followed by a CRC byte).
func validateAndStripCRC(raw []byte) ([]byte, error) {
	if len(raw)%3 != 0 {
		return nil, fmt.Errorf("sen6x: data length is not a multiple of 3: %d", len(raw))
	}

	data := make([]byte, 0, len(raw)/3*2)
	for i := 0; i < len(raw); i += 3 {
		if crc := crc8(raw[i], raw[i+1]); crc != raw[i+2] {
			return nil, fmt.Errorf("sen6x: CRC mismatch at word %d: got %#02x, want %#02x", i/3, crc, raw[i+2])
		}
		data = append(data, raw[i], raw[i+1])
	}

	return data, nil
}

// packWordsWithCRC packs 16-bit words with their CRCs, ready for transmission
// to the device.
func packWordsWithCRC(words []uint16) []byte {
	result := make([]byte, 0, len(words)*3)
	for _, w := range words {
		high := byte(w >> 8)
		low := byte(w)

		result = append(result, high, low, crc8(high, low))
	}

	return result
}

// packBytesWithCRC packs bytes with CRC values for each pair of bytes, ready
// for transmission to the device. The number of bytes in b must be even.
func packBytesWithCRC(b []byte) ([]byte, error) {
	if len(b)%2 != 0 {
		return nil, errors.New("sen6x: cannot pack bytes with CRC, number of bytes must be even")
	}

	result := make([]byte, 0, len(b)*3/2)
	for i := 0; i < len(b); i += 2 {
		// gosec emits CWE-118, "slice index out of range", for the two i+1
		// indexes below. This is a false positive: we know that the length of
		// b is even because of the check above. The loop body doesn't execute
		// if b's length is 0, a length of 1 is impossible, 2 works fine, 3 is
		// impossible, and so on.
		// #nosec CWE-118
		result = append(result, b[i], b[i+1], crc8(b[i], b[i+1]))
	}

	return result, nil
}
