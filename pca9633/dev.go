// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// The PCA9633 is a four-channel LED PWM controller. Additionally, it provides
// features for dimming and blink.
//
// # Datasheet
//
// https://www.nxp.com/docs/en/data-sheet/PCA9633.pdf
package pca9633

import (
	"fmt"
	"time"

	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/i2c"
)

type LEDStructure byte

const (
	// LEDs are connected in OpenDrain format
	STRUCT_OPENDRAIN LEDStructure = iota
	// LEDs are connected in TotemPole format.
	STRUCT_TOTEMPOLE
)

type LEDMode byte

const (
	MODE_FULL_OFF LEDMode = iota
	MODE_FULL_ON
	// The brightness of the LED is controlled by the PWM setting.
	MODE_PWM
	// The brightness of the LED is controlled by the PWM setting AND the group
	// PWM/blinking options.
	MODE_PWM_PLUS_GROUP
)

const (
	// Register offsets from the datasheet
	_DEV_MODE1 byte = iota
	_DEV_MODE2
	_PWM0
	_PWM1
	_PWM2
	_PWM3
	_GRPPWM
	_GRPFREQ
	_LED_MODE
)

const (
	_DEV_MODE_BLINK    byte = 0x20
	_DEV_MODE_TOTEM    byte = 0x08
	_DEV_MODE_INVERT   byte = 0x10
	_DEV_MODE2_DEFAULT byte = 0x05
	_DEV_MODE1_DEFAULT byte = 0x81
)

// Dev represents a PCA9633 LED PWM Controller.
type Dev struct {
	d     *i2c.Dev
	modes []LEDMode
	// bit settings for device mode register 2
	devMode2 byte
}

// New returns an initialized PCA9633 device ready for use.
func New(bus i2c.Bus, address uint16, ledStructure LEDStructure) (*Dev, error) {
	dev := &Dev{d: &i2c.Dev{Bus: bus, Addr: address},
		modes:    make([]LEDMode, 4),
		devMode2: _DEV_MODE2_DEFAULT}

	if ledStructure == STRUCT_TOTEMPOLE {
		dev.devMode2 |= _DEV_MODE_TOTEM
	}
	return dev, dev.init()
}

func (dev *Dev) init() error {
	// We have to write 0 to bit 5 to turn on the PWM oscillator...
	err := dev.d.Tx([]byte{_DEV_MODE1, _DEV_MODE1_DEFAULT}, nil)
	if err == nil {
		err = dev.d.Tx([]byte{_DEV_MODE2, dev.devMode2}, nil)
		if err == nil {
			err = dev.SetModes(MODE_FULL_OFF, MODE_FULL_OFF, MODE_FULL_OFF, MODE_FULL_OFF)
		}
	}
	return wrap(err)
}

func wrap(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("pca9633: %w", err)
}

// Halt stops all LED display by setting them all to MODE_FULL_OFF. Implements
// conn.Resource
func (dev *Dev) Halt() error {
	return dev.SetModes(MODE_FULL_OFF, MODE_FULL_OFF, MODE_FULL_OFF, MODE_FULL_OFF)
}

// Set the output intensity for LEDs. If intensity is 0, the LED is set to full
// off. If intensity==255, the LED is set to full on, otherwise the LED is PWMd
// to the desired intensity.
func (dev *Dev) Out(intensities ...display.Intensity) error {
	newModes := make([]LEDMode, len(dev.modes))
	copy(newModes, dev.modes)
	for ix := range len(intensities) {
		if intensities[ix] == 0 {
			newModes[ix] = MODE_FULL_OFF
		} else if intensities[ix] >= 0xff && dev.modes[ix] == MODE_FULL_OFF {
			newModes[ix] = MODE_FULL_ON
		} else {
			if dev.modes[ix] != MODE_PWM && dev.modes[ix] != MODE_PWM_PLUS_GROUP {
				newModes[ix] = MODE_PWM
			}
			err := dev.d.Tx([]byte{_PWM0 + byte(ix), byte(intensities[ix])}, nil)
			if err != nil {
				return wrap(err)
			}
		}
	}
	return dev.SetModes(newModes...)
}

// SetGroupPWMBlink sets the group level PWM value, and optionally, a blink
// duration. Blink duration can range from 41,666 uS to 10.625 S. If 0, blink
// is disabled.
//
// Refer to the datasheet on this functionality. If the mode is not blink,
// then it's group PWM, but group PWM is only applied if the individual led
// mode is MODE_PWM_PLUS_GROUP
func (dev *Dev) SetGroupPWMBlink(intensity display.Intensity, blinkDuration time.Duration) error {
	periodIncrement := 41_666 * time.Microsecond
	newDevMode := dev.devMode2
	if blinkDuration >= periodIncrement {
		// calculate the duration value.
		var blinkSetting int
		cnt := int(blinkDuration / periodIncrement)
		if cnt < 0 {
			blinkSetting = 0
		} else if cnt > 0xff {
			blinkSetting = 0xff
		} else {
			blinkSetting = cnt
		}
		if blinkSetting == 0 {
			newDevMode ^= _DEV_MODE_BLINK
		} else {
			err := dev.d.Tx([]byte{_GRPFREQ, byte(blinkSetting)}, nil)
			if err != nil {
				return wrap(err)
			}
			if dev.devMode2&_DEV_MODE_BLINK != _DEV_MODE_BLINK {
				newDevMode |= _DEV_MODE_BLINK
			}
		}
	} else {
		if dev.devMode2&_DEV_MODE_BLINK == _DEV_MODE_BLINK {
			newDevMode ^= _DEV_MODE_BLINK
		}
	}
	if newDevMode != dev.devMode2 {
		err := dev.d.Tx([]byte{_DEV_MODE2, newDevMode}, nil)
		if err != nil {
			return wrap(err)
		}
		dev.devMode2 = newDevMode
	}
	err := dev.d.Tx([]byte{_GRPPWM, byte(intensity)}, nil)
	return wrap(err)
}

// SetInvert allows you to easily invert the meaning of the PWM values. This
// is useful if you're driving LEDs with a transistor or other device that
// inverts the output.
func (dev *Dev) SetInvert(invert bool) error {
	if invert {
		dev.devMode2 |= _DEV_MODE_INVERT
	} else {
		dev.devMode2 ^= _DEV_MODE_INVERT
	}
	err := dev.d.Tx([]byte{_DEV_MODE2, dev.devMode2}, nil)
	return wrap(err)
}

// SetModes sets the output mode of LEDs. The value for modes should be
// one of the LEDMode constants.
func (dev *Dev) SetModes(modes ...LEDMode) error {
	var mode byte
	var changed bool
	for i := range len(modes) {
		changed = changed || (modes[i] != dev.modes[i])
		mode |= (byte(modes[i]) << (i * 2))
	}
	if !changed {
		return nil
	}
	copy(dev.modes, modes)
	err := dev.d.Tx([]byte{_LED_MODE, mode}, nil)
	return wrap(err)
}

func (dev *Dev) String() string {
	return fmt.Sprintf("PCA9633::%#v", dev.d)
}
