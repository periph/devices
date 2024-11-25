// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.
//
// This package provides a driver for the Texas Instruments HDC3021/3022
// I2C Temperature/Humidity Sensors
//
// The datasheet is available at:
//
//      https://www.ti.com/lit/ds/symlink/hdc3022.pdf
package hdc302x

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
)

// NewI2C returns a new HDC302x sensor using the specified bus and address.
// If opts is not supplied, the configuration of the sensor is set to the
// default on startup.
func NewI2C(b i2c.Bus, addr uint16, opts *Opts) (*Dev, error) {
    if opts == nil {
        opts = &Opts{SampleRate: RateFourHertz, AlertSetting: ModeComparator}
    }
    d := &Dev{d: &i2c.Dev{Bus: b, Addr: addr}, opts: opts, shutdown: nil}
    return d, d.start()
}

// Halt shuts down the device. If a SenseContinuous operation is in progress,
// its aborted. Implements conn.Resource
func (dev *Dev) Halt() error {
}

// Sense reads temperature from the device and writes the value to the specified
// env variable. Implements physic.SenseEnv.
func (dev *Dev) Sense(env *physic.Env) error {
}

// SenseContinuous continuously reads from the device and writes the value to
// the returned channel. Implements physic.SenseEnv. To terminate the
// continuous read, call Halt().
func (dev *Dev) SenseContinuous(interval time.Duration) (<-chan physic.Env, error) {
}

// Precision returns the sensor's precision, or minimum value between steps the
// device can make. The specified precision is 0.0625 degrees Celsius. Note
// that the accuracy of the device is +/- 0.5 degrees Celsius.
func (dev *Dev) Precision(env *physic.Env) {
    env.Temperature = _DEGREES_RESOLUTION
    env.Pressure = 0
    env.Humidity = 0
}

func (dev *Dev) String() string {
    return fmt.Sprintf("hdc302x: %s", dev.d.String())
}





var _ conn.Resource = &Dev{}
var _ physic.SenseEnv = &Dev{}
