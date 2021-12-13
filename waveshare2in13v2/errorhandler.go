// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

import (
	"time"

	"periph.io/x/conn/v3/gpio"
)

// errorHandler is a wrapper for error management.
type errorHandler struct {
	d   Dev
	err error
}

func (eh *errorHandler) rstOut(l gpio.Level) {
	if eh.err != nil {
		return
	}
	eh.err = eh.d.rst.Out(l)
}

func (eh *errorHandler) cTx(w []byte, r []byte) {
	if eh.err != nil {
		return
	}
	eh.err = eh.d.c.Tx(w, r)
}

func (eh *errorHandler) dcOut(l gpio.Level) {
	if eh.err != nil {
		return
	}
	eh.err = eh.d.dc.Out(l)
}

func (eh *errorHandler) csOut(l gpio.Level) {
	if eh.err != nil {
		return
	}
	eh.err = eh.d.cs.Out(l)
}

func (eh *errorHandler) waitUntilIdle() {
	for eh.d.busy.Read() == gpio.High {
		time.Sleep(100 * time.Millisecond)
	}
}

func (eh *errorHandler) sendCommand(cmd byte) {
	if eh.err != nil {
		return
	}

	eh.dcOut(gpio.Low)
	eh.csOut(gpio.Low)
	eh.cTx([]byte{cmd}, nil)
	eh.csOut(gpio.High)
}

func (eh *errorHandler) sendData(data []byte) {
	if eh.err != nil {
		return
	}

	eh.dcOut(gpio.High)
	eh.csOut(gpio.Low)
	eh.cTx(data, nil)
	eh.csOut(gpio.High)
}
