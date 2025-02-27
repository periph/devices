// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package nxp74hc595

import (
	"log"
	"testing"

	"periph.io/x/conn/v3/conntest"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spitest"
)

func TestBasic(t *testing.T) {
	pb := &spitest.Record{Ops: make([]conntest.IO, 0)}
	defer pb.Close()
	conn, err := pb.Connect(physic.MegaHertz, spi.Mode1, 8)
	if err != nil {
		log.Fatal(err)
	}

	dev, err := New(conn)
	if err != nil {
		log.Fatal(err)
	}

	gr, _ := dev.Group(6, 5, 4, 3)
	for i := range 16 {
		gr.Out(gpio.GPIOValue(i), 0)
	}
	singlePin := dev.Pins[7]
	for i := range 20 {
		err = singlePin.Out(gpio.Level(i%2 == 0))
		if err != nil {
			t.Error(err)
		}
		err = dev.Pins[0].Out(i%2 != 0)
		if err != nil {
			t.Error(err)
		}
	}
	err = dev.Pins[0].Out(gpio.Low)
	if err != nil {
		t.Error(err)
	}
	err = singlePin.Out(gpio.High)
	if err != nil {
		t.Error(err)
	}
}
