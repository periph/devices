// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package gc9a01 controls a GC9A01 240x240 round RGB LCD display over SPI.
//
// The GC9A01 is a single-chip driver for 240x240 resolution TFT LCD displays
// with 65K colors (RGB565). It communicates via 4-wire SPI.
//
// # Datasheet
//
// https://www.buydisplay.com/download/ic/GC9A01A.pdf
//
// # Wiring
//
// Connect SDA to SPI_MOSI, SCL to SPI_CLK, CS to SPI_CS, DC to a GPIO pin.
// Optionally connect RST to a GPIO pin for hardware reset.
package gc9a01
