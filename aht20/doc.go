// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package aht20 controls an AHT20 device over I²C.
// The sensor is a temperature and humidity sensor with a typical accuracy of ±2% RH and ±0.3°C.
// The aht20.Dev type implements the physic.SenseEnv interface. The physic.Env measurement results
// contain a temperature, pressure and humidity value though the pressure is not set.
//
// **Datasheet:** http://www.aosong.com/userfiles/files/media/Data%20Sheet%20AHT20.pdf
package aht20
