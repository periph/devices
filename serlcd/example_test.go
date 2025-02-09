// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package serlcd_test

import (
	"errors"
	"fmt"
	"log"
	"time"

	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/display/displaytest"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/serlcd"
	"periph.io/x/host/v3"
)

func Example() {
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer bus.Close()

	conn := &i2c.Dev{Bus: bus, Addr: serlcd.DefaultI2CAddress}
	dev := serlcd.NewConn(conn, 4, 20)
	_ = dev.Clear()
	n, err := dev.WriteString("Hello")
	fmt.Printf("n=%d, err=%s\n", n, err)

	time.Sleep(10 * time.Second)

	fmt.Println("calling TestTextDisplay")

	errs := displaytest.TestTextDisplay(dev, true)
	fmt.Println("back from TestTextDsiplay")
	for _, e := range errs {
		if !errors.Is(e, display.ErrNotImplemented) {
			fmt.Println(e)
		}
	}

}
