// Copyright 2016 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package ssd1306 controls a monochrome OLED display via a SSD1306, SH1106,
// or SH1107 controller. The driver automatically detects the variant and
// adjusts accordingly.
//
// The driver does differential updates: it only sends modified pixels for the
// smallest rectangle, to economize bus bandwidth. This is especially important
// when using I²C as the bus default speed (often 100kHz) is slow enough to
// saturate the bus at less than 10 frames per second.
//
// The device can be driven on either I²C or SPI with 4 wires. Changing
// between protocol is likely done through resistor soldering, for boards that
// support both.
//
// Some boards expose a RES / Reset pin. If present, it must be normally be
// High. When set to Low (Ground), it enables the reset circuitry. It can be
// used externally to this driver, if used, the driver must be reinstantiated.
//
// # More details
//
// See https://periph.io/device/ssd1306/ for more details about the device.
//
// # Datasheets
//
// Product page:
//
// SSD1306
//
// http://www.solomon-systech.com/en/product/display-ic/oled-driver-controller/ssd1306/
//
// https://cdn-shop.adafruit.com/datasheets/SSD1306.pdf
//
// "DM-OLED096-624": https://drive.google.com/file/d/0B5lkVYnewKTGaEVENlYwbDkxSGM/view
//
// SH1106
//
// https://cdn.velleman.eu/downloads/29/infosheets/sh1106_datasheet.pdf
//
// SH1107
//
// https://www.adafruit.com/product/5297
//
// https://www.displayfuture.com/Display/datasheet/controller/SH1107.pdf
package ssd1306
