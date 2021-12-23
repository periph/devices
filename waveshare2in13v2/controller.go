// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

type controller interface {
	sendCommand(byte)
	sendData([]byte)
	waitUntilIdle()
}

func initDisplay(ctrl controller, opts *Opts) {
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

	ctrl.sendCommand(gateDrivingVoltageControl)
	ctrl.sendData([]byte{gateDrivingVoltage19V})

	ctrl.sendCommand(sourceDrivingVoltageControl)
	ctrl.sendData([]byte{sourceDrivingVoltageVSH1_15V, sourceDrivingVoltageVSH2_5V, sourceDrivingVoltageVSL_neg15V})

	ctrl.sendCommand(setDummyLinePeriod)
	ctrl.sendData([]byte{0x30})

	ctrl.sendCommand(setGateTime)
	ctrl.sendData([]byte{0x0A})
}

func configDisplayMode(ctrl controller, mode PartialUpdate, lut LUT) {
	var vcom byte
	var borderWaveformControlValue byte

	switch mode {
	case Full:
		vcom = 0x55
		borderWaveformControlValue = 0x03
	case Partial:
		vcom = 0x24
		borderWaveformControlValue = 0x01
	}

	ctrl.sendCommand(writeVcomRegister)
	ctrl.sendData([]byte{vcom})

	ctrl.sendCommand(borderWaveformControl)
	ctrl.sendData([]byte{borderWaveformControlValue})

	ctrl.sendCommand(writeLutRegister)
	ctrl.sendData(lut[:70])

	if mode == Partial {
		// Undocumented command used in vendor example code.
		ctrl.sendCommand(0x37)
		ctrl.sendData([]byte{0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00})

		ctrl.sendCommand(displayUpdateControl2)
		ctrl.sendData([]byte{
			displayUpdateEnableClock |
				displayUpdateEnableAnalog,
		})

		ctrl.sendCommand(masterActivation)
	}

	ctrl.waitUntilIdle()
}

func updateDisplay(ctrl controller, mode PartialUpdate) {
	var displayUpdateFlags byte

	if mode == Partial {
		// Make use of red buffer
		displayUpdateFlags = 0b1000_0000
	}

	ctrl.sendCommand(displayUpdateControl1)
	ctrl.sendData([]byte{displayUpdateFlags})

	ctrl.sendCommand(displayUpdateControl2)
	ctrl.sendData([]byte{
		displayUpdateDisableClock |
			displayUpdateDisableAnalog |
			displayUpdateDisplay |
			displayUpdateEnableClock |
			displayUpdateEnableAnalog,
	})

	ctrl.sendCommand(masterActivation)
	ctrl.waitUntilIdle()
}
