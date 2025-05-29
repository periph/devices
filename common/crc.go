// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package common contains functions used across multiple packages. For
// example, a CRC8 calculation
package common

// CRC8 calculates the 8-bit CRC of the byte slice parameter and returns the
// calculated value. CRC bytes are used in sensors from TI and Sensirion.
func CRC8(bytes []byte) byte {
	var crc byte = 0xff
	for _, val := range bytes {
		crc ^= val
		for range 8 {
			if (crc & 0x80) == 0 {
				crc <<= 1
			} else {
				crc = (byte)((crc << 1) ^ 0x31)
			}
		}
	}
	return crc
}
