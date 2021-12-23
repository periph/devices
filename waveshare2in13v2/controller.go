// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

type controller interface {
	sendCommand(byte)
	sendData([]byte)
	waitUntilIdle()
}

func initDisplayFull(ctrl controller, opts *Opts) {
	ctrl.waitUntilIdle()
	ctrl.sendCommand(swReset)
	ctrl.waitUntilIdle()

	ctrl.sendCommand(setAnalogBlockControl)
	ctrl.sendData([]byte{0x54})

	ctrl.sendCommand(setDigitalBlockControl)
	ctrl.sendData([]byte{0x3B})

	ctrl.sendCommand(driverOutputControl)
	ctrl.sendData([]byte{
		byte((opts.Height - 1) % 0xFF),
		byte((opts.Height - 1) / 0xFF),
		0x00,
	})

	ctrl.sendCommand(borderWaveformControl)
	ctrl.sendData([]byte{0x03})

	ctrl.sendCommand(writeVcomRegister)
	ctrl.sendData([]byte{0x55})

	ctrl.sendCommand(gateDrivingVoltageControl)
	ctrl.sendData([]byte{gateDrivingVoltage19V})

	ctrl.sendCommand(sourceDrivingVoltageControl)
	ctrl.sendData([]byte{sourceDrivingVoltageVSH1_15V, sourceDrivingVoltageVSH2_5V, sourceDrivingVoltageVSL_neg15V})

	ctrl.sendCommand(setDummyLinePeriod)
	ctrl.sendData([]byte{0x30})

	ctrl.sendCommand(setGateTime)
	ctrl.sendData([]byte{0x0A})

	ctrl.sendCommand(writeLutRegister)
	ctrl.sendData(opts.FullUpdate[:70])
}

func initDisplayPartial(ctrl controller, opts *Opts) {
	ctrl.sendCommand(writeVcomRegister)
	ctrl.sendData([]byte{0x26})

	ctrl.waitUntilIdle()

	ctrl.sendCommand(writeLutRegister)
	ctrl.sendData(opts.PartialUpdate[:70])

	// Undocumented command used in vendor example code.
	ctrl.sendCommand(0x37)
	ctrl.sendData([]byte{0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00})

	ctrl.sendCommand(displayUpdateControl2)
	ctrl.sendData([]byte{0xC0})

	ctrl.sendCommand(masterActivation)

	ctrl.waitUntilIdle()

	ctrl.sendCommand(borderWaveformControl)
	ctrl.sendData([]byte{0x01})
}
