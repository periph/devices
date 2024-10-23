// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v4

type controller interface {
	sendCommand(byte)
	sendData([]byte)
	sendByte(byte)
	readBusy()
}

func initDisplay(ctrl controller, opts *Opts) {
	// self.ReadBusy()
	ctrl.readBusy()
	// self.send_command(0x12)  #SWRESET
	ctrl.sendCommand(swReset)
	// self.ReadBusy()
	ctrl.readBusy()

	// self.send_command(0x01) #Driver output control
	ctrl.sendCommand(driverOutputControl)
	// self.send_data(0xf9)
	// self.send_data(0x00)
	// self.send_data(0x00)
	ctrl.sendData([]byte{0xF9, 0x00, 0x00})

	// self.send_command(0x11) #data entry mode
	ctrl.sendCommand(dataEntryModeSetting)
	// self.send_data(0x03)
	ctrl.sendByte(0x03)

	// self.SetWindow(0, 0, self.width-1, self.height-1)
	setWindow(ctrl, 0, 0, opts.Width-1, opts.Height-1)
	// self.SetCursor(0, 0)
	setCursor(ctrl, 0, 0)

	// self.send_command(0x3c)
	ctrl.sendCommand(borderWaveformControl)
	// self.send_data(0x05)
	ctrl.sendByte(0x05)

	// self.send_command(0x21) #  Display update control
	ctrl.sendCommand(displayUpdateControl1)
	// self.send_data(0x00)
	// self.send_data(0x80)
	ctrl.sendData([]byte{0x80, 0x80})

	// self.send_command(0x18)
	ctrl.sendCommand(tempSensorSelect)
	// self.send_data(0x80)
	ctrl.sendByte(0x80)

	// self.ReadBusy()
	ctrl.readBusy()
}

func initDisplayFast(ctrl controller, opts *Opts) {
	// self.send_command(0x12)  #SWRESET
	ctrl.sendCommand(swReset)
	// self.ReadBusy()
	ctrl.readBusy()

	// self.send_command(0x18)
	ctrl.sendCommand(tempSensorSelect)
	// self.send_data(0x80)
	ctrl.sendByte(0x80)

	// self.send_command(0x11) #data entry mode
	ctrl.sendCommand(dataEntryModeSetting)
	// self.send_data(0x03)
	ctrl.sendByte(0x03)

	// self.SetWindow(0, 0, self.width-1, self.height-1)
	setWindow(ctrl, 0, 0, opts.Width-1, opts.Height-1)
	// self.SetCursor(0, 0)
	setCursor(ctrl, 0, 0)

	// self.send_command(0x22) # Load temperature value
	ctrl.sendCommand(displayUpdateControl2)
	// self.send_data(0xB1)
	ctrl.sendByte(0x81)
	// self.send_command(0x20)
	ctrl.sendCommand(masterActivation)
	// self.ReadBusy()
	ctrl.readBusy()

	// self.send_command(0x1A) # Write to temperature register
	ctrl.sendCommand(tempSensorRegWrite)
	// self.send_data(0x64)
	// self.send_data(0x00)
	ctrl.sendData([]byte{0x64, 0x00})

	// self.send_command(0x22) # Load temperature value
	ctrl.sendCommand(displayUpdateControl2)
	// self.send_data(0x91)
	ctrl.sendByte(0x91)
	// self.send_command(0x20)
	ctrl.sendCommand(masterActivation)
	// self.ReadBusy()
	ctrl.readBusy()
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
	ctrl.readBusy()
}

// new

// turnOnDisplay turns on the display.
func turnOnDisplay(ctrl controller) {
	ctrl.sendCommand(displayUpdateControl2)
	ctrl.sendByte(0xf7)
	ctrl.sendCommand(masterActivation)
	ctrl.readBusy()
}

// turnOnDisplayFast turns on the display fast.
func turnOnDisplayFast(ctrl controller) {
	ctrl.sendCommand(displayUpdateControl2)
	ctrl.sendByte(0xC7)
	ctrl.sendCommand(masterActivation)
	ctrl.readBusy()
}

// turnOnDisplayPart turns on the display for a partial update.
func turnOnDisplayPart(ctrl controller) {
	ctrl.sendCommand(displayUpdateControl2)
	ctrl.sendByte(0xFF)
	ctrl.sendCommand(masterActivation)
	ctrl.readBusy()
}

// setWindow sets the display window size.
func setWindow(ctrl controller, x_start int, y_start int, x_end int, y_end int) {
	ctrl.sendCommand(setRAMXAddressStartEndPosition)
	ctrl.sendData([]byte{byte((x_start >> 3) & 0xFF), byte((x_end >> 3) & 0xFF)})

	ctrl.sendCommand(setRAMYAddressStartEndPosition)
	ctrl.sendData([]byte{byte(y_start & 0xFF), byte((y_start >> 8) & 0xFF), byte(y_end & 0xFF), byte((y_end >> 8) & 0xFF)})
}

// setCursor positions the cursor.
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
	// new
	ctrl.sendCommand(writeRAMRed)
	ctrl.sendData(buff)

	turnOnDisplay(ctrl)
}
