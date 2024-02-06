// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v3

type controller interface {
	sendCommand(byte)
	sendData([]byte)
	waitUntilIdle()
}

func initDisplay(ctrl controller, opts *Opts) {
	ctrl.waitUntilIdle()
	ctrl.sendCommand(swReset)
	ctrl.waitUntilIdle()

	ctrl.sendCommand(driverOutputControl)
	ctrl.sendData([]byte{0xf9, 0x00, 0x00})

	ctrl.sendCommand(dataEntryModeSetting)
	ctrl.sendData([]byte{0x03})

	setWindow(ctrl, 0, 0, opts.Width-1, opts.Height-1)
	setCursor(ctrl, 0, 0)

	ctrl.sendCommand(borderWaveformControl)
	ctrl.sendData([]byte{0x05})

	ctrl.sendCommand(displayUpdateControl1)
	ctrl.sendData([]byte{0x00, 0x80})

	ctrl.sendCommand(tempSensorSelect)
	ctrl.sendData([]byte{0x80})

	ctrl.waitUntilIdle()

	setLut(ctrl, opts.FullUpdate)
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

	ctrl.sendCommand(writeDisplayOptionRegister)
	ctrl.sendData([]byte{0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00})

	// Start up the parts likely used by a draw operation soon.
	ctrl.sendCommand(displayUpdateControl2)
	ctrl.sendData([]byte{displayUpdateEnableClock | displayUpdateEnableAnalog})

	ctrl.sendCommand(masterActivation)
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

// new

// turnOnDisplay turns on the display if mode = true it does a partial display
func turnOnDisplay(ctrl controller, mode PartialUpdate) {
	var upMode byte = 0xC7
	if mode {
		upMode = 0x0f
	}
	ctrl.sendCommand(displayUpdateControl2)
	ctrl.sendData([]byte{upMode})
	ctrl.sendCommand(masterActivation)
	ctrl.waitUntilIdle()
}

func lookUpTable(ctrl controller, lut LUT) {
	ctrl.sendCommand(writeLutRegister)
	ctrl.sendData(lut[:153])
	ctrl.waitUntilIdle()
}

func setLut(ctrl controller, lut LUT) {
	lookUpTable(ctrl, lut)
	ctrl.sendCommand(endOptionEOPT)
	ctrl.sendData([]byte{lut[153]})
	ctrl.sendCommand(gateDrivingVoltageControl)
	ctrl.sendData([]byte{lut[154]})
	ctrl.sendCommand(sourceDrivingVoltageControl)
	ctrl.sendData(lut[155:157])
	ctrl.sendCommand(writeVcomRegister)
	ctrl.sendData([]byte{lut[158]})
}

func setWindow(ctrl controller, x_start int, y_start int, x_end int, y_end int) {
	ctrl.sendCommand(setRAMXAddressStartEndPosition)
	ctrl.sendData([]byte{byte((x_start >> 3) & 0xFF), byte((x_end >> 3) & 0xFF)})

	ctrl.sendCommand(setRAMYAddressStartEndPosition)
	ctrl.sendData([]byte{byte(y_start & 0xFF), byte((y_start >> 8) & 0xFF), byte(y_end & 0xFF), byte((y_end >> 8) & 0xFF)})
}

func setCursor(ctrl controller, x int, y int) {
	ctrl.sendCommand(setRAMXAddressCounter)
	// x point must be the multiple of 8 or the last 3 bits will be ignored
	ctrl.sendData([]byte{byte(x & 0xFF)})

	ctrl.sendCommand(setRAMYAddressCounter)
	ctrl.sendData([]byte{byte(y & 0xFF), byte((y >> 8) & 0xFF)})
}

func clear(ctrl controller, color byte, opts *Opts) {
	var linewidth int
	if opts.Width%8 == 0 {
		linewidth = int(opts.Width / 8)
	} else {
		linewidth = int(opts.Width/8) + 1
	}

	var buff []byte
	ctrl.sendCommand(writeRAMBW)
	for j := 0; j < opts.Height; j++ {
		for i := 0; i < linewidth; i++ {
			buff = append(buff, color)
		}
	}
	ctrl.sendData(buff)

	turnOnDisplay(ctrl, false)
}
