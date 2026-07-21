// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package dht22

import (
	"errors"
	"testing"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpiotest"
)

type outErrorPin struct {
	*gpiotest.Pin
	err error
}

func (p *outErrorPin) Out(l gpio.Level) error {
	if l == gpio.High {
		return p.err
	}
	return p.Pin.Out(l)
}

func TestDev(t *testing.T) {
	t.Run("Read returns output error", func(t *testing.T) {
		errOutput := errors.New("output error")
		d := Dev{pin: &outErrorPin{Pin: &gpiotest.Pin{}, err: errOutput}}

		_, err := d.Read()
		if !errors.Is(err, errOutput) {
			t.Fatalf("Read() error = %v, want %v", err, errOutput)
		}
	})
}
