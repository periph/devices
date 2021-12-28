// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package waveshare2in13v2 controls Waveshare 2.13 v2 e-paper displays.
//
// Datasheet:
// https://www.waveshare.com/w/upload/d/d5/2.13inch_e-Paper_Specification.pdf
//
// Product page:
// 2.13 inch version 2: https://www.waveshare.com/wiki/2.13inch_e-Paper_HAT
//
// The Waveshare 2.13in v2 display is a GoodDisplay GDEH0213B72. Its IL3897
// controller is compatible with the SSD1675A (sometimes also referred to as
// SSD1675). The SSD1675A should not be mixed up with the SSD1675B. They have
// different LUT formats (70 bytes for SSD1675A, 100 bytes for SSD1675B).
package waveshare2in13v2
