// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Tests MPU9250 device initialization.

package mpu9250

import (
	"testing"

	"periph.io/x/conn/v3/i2c/i2ctest"
)

func TestNew(t *testing.T) {
	if dev, err := New(nil); err == nil || dev != nil {
		t.Fatal("nil transport was accepted")
	}
	bus := &i2ctest.Playback{Ops: []i2ctest.IO{{Addr: 0x68, W: []byte{0x75}, R: []byte{0x71}}}}
	tr, err := NewI2cTransport(bus, 0x68)
	if err != nil {
		t.Fatal(err)
	}
	dev, err := New(tr)
	if err != nil {
		t.Fatal(err)
	}
	id, err := dev.GetDeviceID()
	if err != nil {
		t.Fatal(err)
	}
	if id != 0x71 {
		t.Fatalf("device ID = %x, want 71", id)
	}
	if err := bus.Close(); err != nil {
		t.Fatal(err)
	}
}
