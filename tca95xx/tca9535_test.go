// Copyright 2022 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package tca95xx

import (
	"reflect"
	"testing"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/conn/v3/physic"
)

func TestTCA9535_out(t *testing.T) {
	const address uint16 = 0x20
	scenario := &i2ctest.Playback{
		Ops: []i2ctest.IO{
			// iodir is read on creation
			{Addr: address, W: []byte{0x06}, R: []byte{0xFF}},
			{Addr: address, W: []byte{0x07}, R: []byte{0xFF}},
			// iodir is set to output
			{Addr: address, W: []byte{0x06, 0xFE}, R: nil},
			// output is read
			{Addr: address, W: []byte{0x02}, R: []byte{0x00}},

			// writing back unchanged value is omitted
			// writing high output
			{Addr: address, W: []byte{0x02, 0x01}, R: nil},
			// writing low output
			{Addr: address, W: []byte{0x02, 0x00}, R: nil},
		},
	}

	dev, err := New(scenario, TCA9535, address)
	if err != nil {
		t.Fatal(err)
	}
	if dev == nil {
		t.Fatal("dev is nil")
	}
	defer dev.Close()

	p0 := gpioreg.ByName("TCA9535_20_P0_0")
	if nil == p0 {
		t.Fatal("p0 is nil")
	}
	p0.Out(gpio.Low)
	p0.Out(gpio.High)
	p0.Out(gpio.Low)
}

func TestTCA9535_in(t *testing.T) {
	const address uint16 = 0x20
	scenario := &i2ctest.Playback{
		Ops: []i2ctest.IO{
			// iodir is read on creation
			{Addr: address, W: []byte{0x06}, R: []byte{0xFF}},
			{Addr: address, W: []byte{0x07}, R: []byte{0xFF}},
			// not written, since it didn't change
			// input is read
			{Addr: address, W: []byte{0x00}, R: []byte{0x01}},
		},
	}

	dev, err := New(scenario, TCA9535, address)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	p0 := gpioreg.ByName("TCA9535_20_P0_0")

	p0.In(gpio.Float, gpio.NoEdge)
	l := p0.Read()
	if l != gpio.High {
		t.Errorf("Input should be High")
	}
}

func TestTCA9535_inInverted(t *testing.T) {
	const address uint16 = 0x20
	scenario := &i2ctest.Playback{
		Ops: []i2ctest.IO{
			// iodir is read on creation
			{Addr: address, W: []byte{0x06}, R: []byte{0xFF}},
			{Addr: address, W: []byte{0x07}, R: []byte{0xFF}},
			// not written, since it didn't change
			// polarity is set
			{Addr: address, W: []byte{0x04}, R: []byte{0x01}},
			// gpio is read high
			{Addr: address, W: []byte{0x00}, R: []byte{0x01}},
			// gpio is read low
			{Addr: address, W: []byte{0x00}, R: []byte{0x00}},
		},
	}

	dev, err := New(scenario, TCA9535, address)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	p0 := gpioreg.ByName("TCA9535_20_P0_0").(Pin)

	p0.In(gpio.Float, gpio.NoEdge)
	p0.SetPolarityInverted(true)
	l := p0.Read()
	if l != gpio.High {
		t.Errorf("Input should be High")
	}
	l = p0.Read()
	if l != gpio.Low {
		t.Errorf("Input should be Low")
	}
	inverted, err := p0.IsPolarityInverted()
	if inverted != true || err != nil {
		t.Errorf("polarity should return as inverted")
	}
}

func TestTCA9535_Tx(t *testing.T) {
	tests := []struct {
		description string
		scenario    i2ctest.Playback
		output      bool
		t           []byte
		r           []byte
		expectErr   bool
	}{
		{
			description: "working write 2 characters",
			output:      true,
			t:           []byte{0xa5, 0x5a},
			scenario: i2ctest.Playback{
				Ops: []i2ctest.IO{
					// iodir is read on creation
					{Addr: 0x20, W: []byte{0x06}, R: []byte{0xFF}},
					{Addr: 0x20, W: []byte{0x07}, R: []byte{0xFF}},
					// iodir is set to output
					{Addr: 0x20, W: []byte{0x06, 0xFE}, R: nil},
					// output is read
					{Addr: 0x20, W: []byte{0x02}, R: []byte{0x00}},
					{Addr: 0x20, W: []byte{0x06, 0xFC}, R: nil},
					{Addr: 0x20, W: []byte{0x06, 0xF8}, R: nil},
					{Addr: 0x20, W: []byte{0x06, 0xF0}, R: nil},
					{Addr: 0x20, W: []byte{0x06, 0xE0}, R: nil},
					{Addr: 0x20, W: []byte{0x06, 0xC0}, R: nil},
					{Addr: 0x20, W: []byte{0x06, 0x80}, R: nil},
					{Addr: 0x20, W: []byte{0x06, 0x00}, R: nil},

					// output is set
					{Addr: 0x20, W: []byte{0x02, 0xa5}, R: nil},
					// output is set
					{Addr: 0x20, W: []byte{0x02, 0x5a}, R: nil},
				},
			},
		}, {
			description: "working read 2 characters",
			r:           []byte{0xa5, 0x5a},
			scenario: i2ctest.Playback{
				Ops: []i2ctest.IO{
					// iodir is read on creation
					{Addr: 0x20, W: []byte{0x06}, R: []byte{0xFF}},
					{Addr: 0x20, W: []byte{0x07}, R: []byte{0xFF}},
					// read the inputs
					{Addr: 0x20, W: []byte{0x00}, R: []byte{0xa5}},
					{Addr: 0x20, W: []byte{0x00}, R: []byte{0x5a}},
				},
			},
		}, {
			description: "Invalid, only r or w may be set.",
			r:           []byte{0xa5, 0x5a},
			t:           []byte{0xa5, 0x5a},
			scenario: i2ctest.Playback{
				Ops: []i2ctest.IO{
					// iodir is read on creation
					{Addr: 0x20, W: []byte{0x06}, R: []byte{0xFF}},
					{Addr: 0x20, W: []byte{0x07}, R: []byte{0xFF}},
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			dev, err := New(&tc.scenario, TCA9535, uint16(0x20))
			if err != nil {
				t.Fatal(err)
			}
			if dev == nil {
				t.Fatal("dev must not be nil")
			}
			defer dev.Close()

			if tc.output {
				// Set the port for output
				for _, pin := range dev.Pins[0] {
					pin.Out(gpio.Low)
				}
			} else {
				// Set the port for input
				for _, pin := range dev.Pins[0] {
					pin.In(gpio.Float, gpio.NoEdge)
				}
			}

			r := make([]byte, len(tc.r))
			err = dev.Conns[0].Tx(tc.t, r)
			if tc.expectErr {
				if err == nil {
					t.Fatal(err)
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if len(tc.r) != len(r) || len(tc.r) > 0 {
				if !reflect.DeepEqual(tc.r, r) {
					t.Fatal("r buffers don't match")
				}
			}
		})
	}
}

func TestTCA9535_fixedValues(t *testing.T) {
	const address uint16 = 0x20
	scenario := &i2ctest.Playback{
		Ops: []i2ctest.IO{
			// iodir is read on creation
			{Addr: address, W: []byte{0x06}, R: []byte{0xFF}},
			{Addr: address, W: []byte{0x07}, R: []byte{0xFF}},
		},
	}

	dev, err := New(scenario, TCA9535, address)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	if conn.Half != dev.Conns[0].Duplex() {
		t.Errorf("Duplex() should return conn.Half")
	}

	if "TCA9535_20_P0" != dev.Conns[0].String() {
		t.Errorf("String() should return 'TCA9535_20_P0'")
	}

	if "TCA9535_20_P1" != dev.Conns[1].String() {
		t.Errorf("String() should return 'TCA9535_20_P1'")
	}

	if "TCA9535_20_P0_1" != dev.Pins[0][1].String() {
		t.Errorf("String() should return 'TCA9535_20_P0_1'")
	}

	if 1 != dev.Pins[0][1].Number() {
		t.Errorf("Number() should return '1'")
	}

	if 6 != dev.Pins[0][6].Number() {
		t.Errorf("Number() should return '6'")
	}

	if false != dev.Pins[0][6].WaitForEdge(10*time.Second) {
		t.Errorf("WaitForEdge() should return 'false'")
	}

	if gpio.Float != dev.Pins[0][5].Pull() {
		t.Errorf("Pull() should return 'gpio.Float'")
	}

	if gpio.Float != dev.Pins[0][5].DefaultPull() {
		t.Errorf("DefaultPull() should return 'gpio.Float'")
	}

	err = dev.Pins[0][0].PWM(gpio.DutyHalf, physic.Hertz)
	if err == nil {
		t.Errorf("PWM should return an error")
	}
}
