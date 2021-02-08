// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package ep0099

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"periph.io/x/conn/v3/i2c/i2ctest"
)

const (
	testDefaultValidAddress = 0x10
)

func TestNewBuildsInstanceSuccessfully(t *testing.T) {
	bus := initTestBus()

	dev, err := New(bus, testDefaultValidAddress)
	if err != nil {
		t.Fatal("New should not return error, got: ", err)
	}

	if bus.Ops[0].Addr != testDefaultValidAddress {
		t.Fatal("Expected operations on address ", testDefaultValidAddress, " got ", bus.Ops[0].Addr)
	}

	checkDevReset(t, dev, bus)
}

func TestNewReturnsInvalidAddress(t *testing.T) {
	bus := initTestBus()

	_, err := New(bus, 0x00)
	if !errors.Is(err, errInvalidAddress) {
		t.Fatal("New should return address validation error, got: ", err)
	}
}

func TestAvailableChannels(t *testing.T) {
	bus := initTestBus()
	expected := []uint8{0x01, 0x02, 0x03, 0x04}

	dev, _ := New(bus, testDefaultValidAddress)
	channels := dev.AvailableChannels()

	if len(channels) != len(expected) {
		t.Fatal("Available channels len should be ", len(expected), ", got ", len(channels))
	}

	for i := 0; i < len(expected); i++ {
		if channels[i] != expected[i] {
			t.Fatal("Available channels should be ", expected, " got ", channels)
		}
	}
}

func TestHalt(t *testing.T) {
	bus := initTestBus()
	dev, _ := New(bus, testDefaultValidAddress)

	resetTestBusOps(bus)

	dev.Halt()
	checkDevReset(t, dev, bus)
}

func TestOn(t *testing.T) {
	bus := initTestBus()
	dev, _ := New(bus, testDefaultValidAddress)

	resetTestBusOps(bus)

	err := dev.On(3)

	if err != nil {
		t.Fatal("Should not return error, got ", err)
	}

	checkBusHasWrite(t, bus, []byte{3, byte(StateOn)})
	checkChannelState(t, dev, 3, StateOn)
}

func TestOff(t *testing.T) {
	bus := initTestBus()
	dev, _ := New(bus, testDefaultValidAddress)

	resetTestBusOps(bus)

	err := dev.Off(4)

	if err != nil {
		t.Fatal("Should not return error, got ", err)
	}

	checkBusHasWrite(t, bus, []byte{4, byte(StateOff)})
	checkChannelState(t, dev, 4, StateOff)
}

func TestReturnErrorForInvalidChannel(t *testing.T) {
	bus := initTestBus()
	dev, _ := New(bus, testDefaultValidAddress)

	if err := dev.On(98); err != errInvalidChannel {
		t.Fatal("On should return invalid channel error, got ", err)
	}

	if err := dev.Off(98); err != errInvalidChannel {
		t.Fatal("Off should return invalid channel error, got ", err)
	}
}

func initTestBus() *i2ctest.Record {
	return &i2ctest.Record{
		Bus: nil,
		Ops: []i2ctest.IO{},
	}
}

func resetTestBusOps(bus *i2ctest.Record) {
	bus.Ops = []i2ctest.IO{}
}

func checkChannelState(t *testing.T, dev *Dev, channel uint8, state State) {
	if actual, _ := dev.State(channel); actual != state {
		msg := fmt.Sprintf("Channel %d should have state %s, got: %s", channel, state, actual)
		t.Fatal(msg)
	}
}

func checkBusHasWrite(t *testing.T, bus *i2ctest.Record, data []byte) {
	for _, op := range bus.Ops {
		if bytes.Equal(op.W, data) {
			return
		}
	}
	t.Fatal("Expected data ", data, " to be written but it never did")
}

func checkDevReset(t *testing.T, dev *Dev, bus *i2ctest.Record) {
	checkBusHasWrite(t, bus, []byte{1, byte(StateOff)})
	checkChannelState(t, dev, 1, StateOff)

	checkBusHasWrite(t, bus, []byte{2, byte(StateOff)})
	checkChannelState(t, dev, 2, StateOff)

	checkBusHasWrite(t, bus, []byte{3, byte(StateOff)})
	checkChannelState(t, dev, 3, StateOff)

	checkBusHasWrite(t, bus, []byte{4, byte(StateOff)})
	checkChannelState(t, dev, 4, StateOff)
}
