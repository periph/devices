// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package tic

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
)

// I2CAddr is the default I²C address for the Tic.
const I2CAddr uint16 = 0x0E

// InputNull represents a null or missing value for some of the Tic's 16-bit
// input variables.
const InputNull uint16 = 0xFFFF

var (
	// ErrConnectionFailed is returned when the driver fails to connect.
	ErrConnectionFailed = errors.New("failed to connect to Tic")

	// ErrInvalidSetting is returned when you provide an invalid value.
	ErrInvalidSetting = errors.New("invalid setting")

	// ErrUnsupportedVariant is returned when a method or setting isn't
	// supported by the Tic variant.
	ErrUnsupportedVariant = errors.New("invalid command for Tic variant")

	// ErrIncorrectPlanningMode is returned when you call a method that isn't
	// compatible with the Tic's current planning mode.
	ErrIncorrectPlanningMode = errors.New("incorrect planning mode")
)

// Variant represents the specific Tic controller variant.
type Variant string

const (
	TicT825 Variant = "Tic T825"
	TicT834 Variant = "Tic T834"
	TicT500 Variant = "Tic T500"
	TicT249 Variant = "Tic T249"
	Tic36v4 Variant = "Tic 36v4"
)

// Dev is a handle to a Tic motor controller device.
type Dev struct {
	c       conn.Conn
	variant Variant
}

// NewI2C returns an object that communicates with a Tic motor controller over
// I²C.
//
// The default address is tic.I2CAddr.
func NewI2C(b i2c.Bus, variant Variant, addr uint16) (*Dev, error) {
	// Check the variant is valid.
	switch variant {
	case TicT825, TicT834, TicT500, TicT249, Tic36v4:
	default:
		return nil, errors.New("device variant is invalid")
	}

	d := Dev{
		c:       &i2c.Dev{Bus: b, Addr: addr},
		variant: variant,
	}

	// Test the connection by doing an I²C transaction. Throw away the result.
	_, err := d.GetStepMode()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnectionFailed, err)
	}

	return &d, nil
}

// String returns the device name in a readable format.
//
// String implements conn.Resource.
func (d *Dev) String() string {
	return string(d.variant)
}

// Halt stops the motor abruptly without respecting the deceleration limit.
//
// Halt implements conn.Resource.
func (d *Dev) Halt() error {
	return d.HaltAndHold()
}

// GetTargetPosition gets the target position, in microsteps.
//
// This is only possible if the planning mode from GetPlanningMode() is
// tic.PlanningModeTargetPosition.
func (d *Dev) GetTargetPosition() (int32, error) {
	mode, err := d.GetPlanningMode()
	if err != nil {
		return 0, err
	}
	if mode != PlanningModeTargetPosition {
		return 0, ErrIncorrectPlanningMode
	}

	v, err := d.getVar32(OffsetTargetPosition)
	return int32(v), err
}

// SetTargetPosition sets the target position of the Tic, in microsteps.
//
// This function sends a "Set target position" to the Tic. The Tic will enter
// target position planning mode and start moving the motor to reach the target
// position.
func (d *Dev) SetTargetPosition(position int32) error {
	return d.commandW32(cmdSetTargetPosition, uint32(position))
}

// GetTargetVelocity gets the target velocity, in microsteps per 10000 seconds.
//
// The default step mode is 1 microstep = 1 full step.
//
// This is only possible if the planning mode from GetPlanningMode() is
// tic.PlanningModeTargetVelocity.
func (d *Dev) GetTargetVelocity() (int32, error) {
	mode, err := d.GetPlanningMode()
	if err != nil {
		return 0, err
	}
	if mode != PlanningModeTargetVelocity {
		return 0, ErrIncorrectPlanningMode
	}

	v, err := d.getVar32(OffsetTargetVelocity)
	return int32(v), err
}

// SetTargetVelocity sets the target velocity of the Tic, in microsteps per
// 10000 seconds.
//
// The default step mode is 1 microstep = 1 full step.
//
// This function sends a "Set target velocity" command to the Tic. The Tic will
// enter target velocity planning mode and start accelerating or decelerating to
// reach the target velocity.
func (d *Dev) SetTargetVelocity(velocity int32) error {
	return d.commandW32(cmdSetTargetVelocity, uint32(velocity))
}

// HaltAndSetPosition stops the motor abruptly without respecting the
// deceleration limit and sets the "Current position" variable, which represents
// where the Tic currently thinks the motor's output is.
//
// This function sends a "Halt and set position" command to the Tic. Besides
// stopping the motor and setting the current position, this command also
// clears the "Position uncertain" flag, sets the "Input state" to "halt", and
// clears the "Input after scaling" variable.
func (d *Dev) HaltAndSetPosition(position int32) error {
	return d.commandW32(cmdHaltAndSetPosition, uint32(position))
}

// HaltAndHold stops the motor abruptly without respecting the deceleration
// limit.
//
// This function sends a "Halt and hold" command to the Tic. Besides stopping
// the motor, this command also sets the "Position uncertain" flag (because
// the abrupt stop might cause steps to be missed), sets the "Input state" to
// "halt", and clears the "Input after scaling" variable.
func (d *Dev) HaltAndHold() error {
	return d.commandQuick(cmdHaltAndHold)
}

// GoHomeReverse tells the Tic to start its homing procedure in the reverse
// direction.
//
// See the "Homing" section of the Tic user's guide for details.
func (d *Dev) GoHomeReverse() error {
	return d.commandW7(cmdGoHome, 0)
}

// GoHomeForward tells the Tic to start its homing procedure in the forward
// direction.
//
// See the "Homing" section of the Tic user's guide for details.
func (d *Dev) GoHomeForward() error {
	return d.commandW7(cmdGoHome, 1)
}

// ResetCommandTimeout prevents the "Command timeout" error from happening for
// some time.
//
// The Tic's default command timeout period is 1000 ms, but it can be changed or
// disabled in the Tic Control Center.
//
// This function sends a "Reset command timeout" command to the Tic.
func (d *Dev) ResetCommandTimeout() error {
	return d.commandQuick(cmdResetCommandTimeout)
}

// Deenergize de-energizes the stepper motor coils.
//
// This function sends a De-energize command to the Tic, causing it to disable
// its stepper motor driver. The motor will stop moving and consuming power. The
// Tic will set the "Intentionally de-energized" error bit, turn on its red LED,
// and drive its ERR line high. This command also sets the "Position uncertain"
// flag (because the Tic is no longer in control of the motor's position).
//
// Note that the Energize command, which can be sent with Energize(), will undo
// the effect of this command (except it will leave the "Position uncertain"
// flag set) and could make the system start up again.
func (d *Dev) Deenergize() error {
	return d.commandQuick(cmdDeenergize)
}

// Energize sends the Energize command.
//
// This function sends an Energize command to the Tic, clearing the
// "Intentionally de-energized" error bit. If there are no other errors,
// this allows the system to start up.
func (d *Dev) Energize() error {
	return d.commandQuick(cmdEnergize)
}

// ExitSafeStart sends the "Exit safe start" command.
//
// This command causes the safe start violation error to be cleared for 200 ms.
// If there are no other errors, this allows the system to start up.
func (d *Dev) ExitSafeStart() error {
	return d.commandQuick(cmdExitSafeStart)
}

// EnterSafeStart sends the "Enter safe start" command.
//
// This command has no effect if safe-start is disabled in the Tic's settings.
//
// This command causes the Tic to stop the motor and set its safe start
// violation error bit. An "Exit safe start" command is required before the Tic
// will move the motor again.
//
// See the Tic user's guide for information about what this command does in
// the other control modes.
func (d *Dev) EnterSafeStart() error {
	return d.commandQuick(cmdEnterSafeStart)
}

// Reset sends the Reset command.
//
// This command makes the Tic forget most parts of its current state. For
// more information, see the Tic user's guide.
func (d *Dev) Reset() error {
	err := d.commandQuick(cmdReset)

	// The Tic's I²C interface will be unreliable for a brief period after the
	// Tic receives the Reset command, so delay 10 ms here.
	time.Sleep(10 * time.Millisecond)

	return err
}

// ClearDriverError attempts to clear a motor driver error.
//
// This function sends a "Clear driver error" command to the Tic. For more
// information, see the Tic user's guide.
func (d *Dev) ClearDriverError() error {
	return d.commandQuick(cmdClearDriverError)
}

// GetMaxSpeed gets the current maximum speed, in microsteps per 10000 seconds.
//
// This is the current value, which could differ from the value in the Tic's
// settings.
func (d *Dev) GetMaxSpeed() (uint32, error) {
	return d.getVar32(OffsetSpeedMax)
}

// SetMaxSpeed sets the maximum speed, in units of steps per 10000 seconds.
//
// Example:
//
//	err := dev.SetMaxSpeed(5550000) // 555 steps per second
//
// This function sends a "Set max speed" command to the Tic. For more
// information, see the Tic user's guide.
func (d *Dev) SetMaxSpeed(speed uint32) error {
	return d.commandW32(cmdSetSpeedMax, speed)
}

// GetStartingSpeed gets the starting speed in microsteps per 10000 seconds.
//
// This is the current value, which could differ from the value in the Tic's
// settings.
func (d *Dev) GetStartingSpeed() (uint32, error) {
	return d.getVar32(OffsetStartingSpeed)
}

// SetStartingSpeed sets the starting speed, in units of steps per 10000
// seconds.
//
// Example:
//
//	err := dev.SetStartingSpeed(500000) // 50 steps per second
//
// This function sends a "Set starting speed" command to the Tic. For more
// information, see the Tic user's guide.
func (d *Dev) SetStartingSpeed(speed uint32) error {
	return d.commandW32(cmdSetStartingSpeed, speed)
}

// GetMaxAccel gets the maximum acceleration, in microsteps per second per 100
// seconds.
//
// This is the current value, which could differ from the value in the Tic's
// settings.
func (d *Dev) GetMaxAccel() (uint32, error) {
	return d.getVar32(OffsetAccelMax)
}

// SetMaxAccel sets the maximum acceleration, in units of steps per second per
// 100 seconds.
//
// Example:
//
//	err := dev.SetMaxAccel(10000) // 100 steps per second per second
//
// This function sends a "Set max acceleration" command to the Tic. For more
// information, see the Tic user's guide.
func (d *Dev) SetMaxAccel(accel uint32) error {
	return d.commandW32(cmdSetAccelMax, accel)
}

// GetMaxDecel gets the maximum deceleration, in microsteps per second per 100
// seconds.
//
// This is the current value, which could differ from the value in the Tic's
// settings.
func (d *Dev) GetMaxDecel() (uint32, error) {
	return d.getVar32(OffsetDecelMax)
}

// SetMaxDecel sets the maximum deceleration, in units of steps per second per
// 100 seconds.
//
// Example:
//
//	err := dev.SetMaxDecel(10000) // 100 steps per second per second
//
// This function sends a "Set max deceleration" command to the Tic. For more
// information, see the Tic user's guide.
func (d *Dev) SetMaxDecel(decel uint32) error {
	return d.commandW32(cmdSetDecelMax, decel)
}

// StepMode describes how many microsteps add up to one fulls step.
type StepMode uint8

const (
	// StepModeFull is 1 microstep per step.
	StepModeFull StepMode = 0
	// StepModeHalf is 2 microsteps per step.
	StepModeHalf StepMode = 1
	// StepModeMicrostep4 is 4 microsteps per step.
	StepModeMicrostep4 StepMode = 2
	// StepModeMicrostep8 is 8 microsteps per step.
	StepModeMicrostep8 StepMode = 3
	// StepModeMicrostep16 is 16 microsteps per step. Valid for Tic T834, Tic
	// T825 and Tic 36v4 only.
	StepModeMicrostep16 StepMode = 4
	// StepModeMicrostep32 is 32 microsteps per step. Valid for Tic T834, Tic
	// T825 and Tic 36v4 only.
	StepModeMicrostep32 StepMode = 5
	// StepModeMicrostep2_100p is 2 microsteps per step at 100% coil current.
	// Valid for Tic T249 only.
	StepModeMicrostep2_100p StepMode = 6
	// StepModeMicrostep64 is 64 microsteps per step. Valid for Tic 36v4 only.
	StepModeMicrostep64 StepMode = 7
	// StepModeMicrostep128 is 128 microsteps per step. Valid for Tic 36v4 only.
	StepModeMicrostep128 StepMode = 8
	// StepModeMicrostep256 is 256 microsteps per step. Valid for Tic 36v4 only.
	StepModeMicrostep256 StepMode = 9
)

// GetStepMode gets the current step mode of the stepper motor.
//
// Example:
//
//	mode, err := dev.GetStepMode()
//	if mode == tic.StepModeMicrostep8 {
//	 	// The Tic is currently using 1/8 microsteps.
//	}
func (d *Dev) GetStepMode() (StepMode, error) {
	v, err := d.getVar8(OffsetStepMode)
	return StepMode(v), err
}

// SetStepMode sets the stepper motor's step mode, which defines how many
// microsteps correspond to one full step.
//
// Example:
//
//	err := dev.SetStepMode(tic.StepModeMicrostep8)
//
// This function sends a "Set step mode" command to the Tic. For more
// information, see the Tic user's guide.
func (d *Dev) SetStepMode(mode StepMode) error {
	if mode > StepModeMicrostep256 {
		return ErrInvalidSetting
	}

	// Check that the variant supports the step mode.
	switch d.variant {
	case TicT825, TicT834:
		if mode > StepModeMicrostep32 {
			return ErrUnsupportedVariant
		}
	case TicT500:
		if mode > StepModeMicrostep8 {
			return ErrUnsupportedVariant
		}
	case TicT249:
		if mode > StepModeMicrostep2_100p {
			return ErrUnsupportedVariant
		}
	case Tic36v4:
		if mode == StepModeMicrostep2_100p {
			return ErrUnsupportedVariant
		}
	}

	return d.commandW7(cmdSetStepMode, uint8(mode))
}

// ticT500CurrentTable is used to convert TicT500 current codes to current.
var ticT500CurrentTable = [33]physic.ElectricCurrent{
	0 * physic.MilliAmpere,
	1 * physic.MilliAmpere,
	174 * physic.MilliAmpere,
	343 * physic.MilliAmpere,
	495 * physic.MilliAmpere,
	634 * physic.MilliAmpere,
	762 * physic.MilliAmpere,
	880 * physic.MilliAmpere,
	990 * physic.MilliAmpere,
	1092 * physic.MilliAmpere,
	1189 * physic.MilliAmpere,
	1281 * physic.MilliAmpere,
	1368 * physic.MilliAmpere,
	1452 * physic.MilliAmpere,
	1532 * physic.MilliAmpere,
	1611 * physic.MilliAmpere,
	1687 * physic.MilliAmpere,
	1762 * physic.MilliAmpere,
	1835 * physic.MilliAmpere,
	1909 * physic.MilliAmpere,
	1982 * physic.MilliAmpere,
	2056 * physic.MilliAmpere,
	2131 * physic.MilliAmpere,
	2207 * physic.MilliAmpere,
	2285 * physic.MilliAmpere,
	2366 * physic.MilliAmpere,
	2451 * physic.MilliAmpere,
	2540 * physic.MilliAmpere,
	2634 * physic.MilliAmpere,
	2734 * physic.MilliAmpere,
	2843 * physic.MilliAmpere,
	2962 * physic.MilliAmpere,
	3093 * physic.MilliAmpere,
}

// ticT249CurrentUnits is used by the library to convert between milliamps and
// the native current unit of the Tic T249, which is 40 mA.
const ticT249CurrentUnits uint16 = 40

// ticCurrentUnits is used by the library to convert between milliamps and the
// native current unit of the T825 and Tic T834, which is 32 mA.
const ticCurrentUnits uint16 = 32

// GetCurrentLimit gets the stepper motor coil current limit.
//
// This is the value being used now, which could differ from the value in the
// Tic's settings.
func (d *Dev) GetCurrentLimit() (physic.ElectricCurrent, error) {
	code, err := d.getVar8(OffsetCurrentLimit)
	if err != nil {
		return 0, err
	}

	switch d.variant {
	case TicT500:
		const maxCode = uint8(len(ticT500CurrentTable) - 1)
		if code > maxCode {
			code = maxCode
		}
		return ticT500CurrentTable[code], nil

	case TicT249:
		milliamps := uint16(code) * ticT249CurrentUnits
		return physic.ElectricCurrent(milliamps) * physic.MilliAmpere, nil

	case Tic36v4:
		milliamps := (uint32(55000)*uint32(code) + 384) / 768
		return physic.ElectricCurrent(milliamps) * physic.MilliAmpere, nil

	default:
		// Tic T825 or Tic T834.
		milliamps := uint16(code) * ticCurrentUnits
		return physic.ElectricCurrent(milliamps) * physic.MilliAmpere, nil
	}
}

// SetCurrentLimit sets the stepper motor coil current limit. If the desired
// current limit is not available, this function uses the closest current limit
// option that is lower than the desired one.
//
// Example:
//
//	err := dev.SetCurrentLimit(500 * physic.MilliAmpere)
//
// This command temporarily sets the stepper motor coil current limit of the
// driver. The provided value will override the corresponding setting from the
// Tic’s non-volatile memory until the next Reset command or power cycle.
//
// This function sends a "Set current limit" command to the Tic. For more
// information about this command and how to choose a good current limit, see
// the Tic user's guide.
func (d *Dev) SetCurrentLimit(limit physic.ElectricCurrent) error {
	milliamps := uint16(limit / physic.MilliAmpere)

	var code uint8
	switch d.variant {
	case TicT500:
		for i := range ticT500CurrentTable {
			if ticT500CurrentTable[i] <= limit {
				code = uint8(i)
			} else {
				break
			}
		}

	case TicT249:
		code = uint8(milliamps / ticT249CurrentUnits)

	case Tic36v4:
		// The Tic 36v4 represents current limits using numbers between 0
		// and 127 that are linearly proportional to the current limit. All
		// numbers within this range are valid current limits.
		const (
			tic36v4MinCurrentLimit = 72 * physic.MilliAmpere
			tic36v4MaxCurrentLimit = 9095 * physic.MilliAmpere
		)
		switch {
		case limit < tic36v4MinCurrentLimit:
			code = 0
		case limit >= tic36v4MaxCurrentLimit:
			code = 127
		default:
			code = uint8((uint32(milliamps)*768 - 55000/2) / 55000)
			if (code < 127) &&
				((55000*(uint32(code)+1)+384)/768) <= uint32(milliamps) {
				code++
			}
		}

	default:
		code = uint8(milliamps / ticCurrentUnits)
	}

	return d.commandW7(cmdSetCurrentLimit, code)
}

// DecayMode describes the possible decay modes. These are valid for the Tic
// T825, T834 and 36v4 only.
type DecayMode uint8

const (
	// DecayModeMixed specifies "Mixed" decay mode on the Tic T825 and
	// "Mixed 50%" on the Tic T834.
	DecayModeMixed DecayMode = 0
	// DecayModeSlow specifies "Slow" decay mode.
	DecayModeSlow DecayMode = 1
	// DecayModeFast specifies "Fast" decay mode.
	DecayModeFast DecayMode = 2
	// DecayModeMixed50 is the same as Mixed, but better expresses your
	// intent if you want to use "Mixed 50%" mode on a Tic T834.
	DecayModeMixed50 DecayMode = 0
	// DecayModeMixed25 specifies "Mixed 25%" decay mode on the Tic T834 and
	// is the same as Mixed on the Tic T825.
	DecayModeMixed25 DecayMode = 3
	// This specifies "Mixed 75%" decay mode on the Tic T834 and is the same as
	// Mixed on the Tic T825.
	DecayModeMixed75 DecayMode = 4
)

// GetDecayMode gets the current decay mode of the stepper motor driver.
//
// Example:
//
//	mode, err := dev.GetDecayMode()
//	if mode == tic.DecayModeSlow {
//	 	// The Tic is in slow decay mode.
//	}
func (d *Dev) GetDecayMode() (DecayMode, error) {
	v, err := d.getVar8(OffsetDecayMode)
	return DecayMode(v), err
}

// SetDecayMode sets the stepper motor driver's decay mode.
//
// Example:
//
//	err := dev.SetDecayMode(DecayModeSlow)
//
// The decay modes are documented in the Tic user's guide.
func (d *Dev) SetDecayMode(mode DecayMode) error {
	if mode > DecayModeMixed75 {
		return ErrInvalidSetting
	}

	// Check that the variant supports the decay mode.
	switch d.variant {
	case TicT825:
		if mode > DecayModeFast {
			return ErrUnsupportedVariant
		}
	case TicT834:
	case Tic36v4:
	default:
		return ErrUnsupportedVariant
	}

	return d.commandW7(cmdSetDecayMode, uint8(mode))
}

// AGCMode describes possible Active Gain Control modes.
type AGCMode uint8

const (
	AGCModeOff       AGCMode = 0
	AGCModeOn        AGCMode = 1
	AGCModeActiveOff AGCMode = 2
)

// GetAGCMode gets the Active Gain Control mode.
//
// This is only valid for the Tic T249.
func (d *Dev) GetAGCMode() (AGCMode, error) {
	if d.variant != TicT249 {
		return 0, ErrUnsupportedVariant
	}

	v, err := d.getVar8(OffsetAGCMode)
	return AGCMode(v & 0xF), err
}

// SetAGCMode sets the Active Gain Control mode.
//
// This is only valid for the Tic T249.
func (d *Dev) SetAGCMode(mode AGCMode) error {
	if d.variant != TicT249 {
		return ErrUnsupportedVariant
	}
	if mode > AGCModeActiveOff {
		return ErrInvalidSetting
	}

	return d.commandW7(cmdSetAGCOption, uint8(mode)&0xF)
}

// AGCBottomCurrentLimit describes the possible Active Gain Control bottom
// current limit percentages.
type AGCBottomCurrentLimit uint8

const (
	AGCBottomCurrentLimitP45 AGCBottomCurrentLimit = 0
	AGCBottomCurrentLimitP50 AGCBottomCurrentLimit = 1
	AGCBottomCurrentLimitP55 AGCBottomCurrentLimit = 2
	AGCBottomCurrentLimitP60 AGCBottomCurrentLimit = 3
	AGCBottomCurrentLimitP65 AGCBottomCurrentLimit = 4
	AGCBottomCurrentLimitP70 AGCBottomCurrentLimit = 5
	AGCBottomCurrentLimitP75 AGCBottomCurrentLimit = 6
	AGCBottomCurrentLimitP80 AGCBottomCurrentLimit = 7
)

// GetAGCBottomCurrentLimit gets the Active Gain Control bottom current limit.
//
// This is only valid for the Tic T249.
func (d *Dev) GetAGCBottomCurrentLimit() (AGCBottomCurrentLimit, error) {
	if d.variant != TicT249 {
		return 0, ErrUnsupportedVariant
	}

	v, err := d.getVar8(OffsetAGCBottomCurrentLimit)
	return AGCBottomCurrentLimit(v & 0xF), err
}

// SetAGCBottomCurrentLimit sets the Active Gain Control bottom current limit.
//
// This is only valid for the Tic T249.
func (d *Dev) SetAGCBottomCurrentLimit(limit AGCBottomCurrentLimit) error {
	if d.variant != TicT249 {
		return ErrUnsupportedVariant
	}
	if limit > AGCBottomCurrentLimitP80 {
		return ErrInvalidSetting
	}

	return d.commandW7(cmdSetAGCOption, 0x10|(uint8(limit)&0xF))
}

// AGCCurrentBoostSteps describes the possible Active Gain Control current boost
// steps values.
type AGCCurrentBoostSteps uint8

const (
	AGCCurrentBoostStepsS5  AGCCurrentBoostSteps = 0
	AGCCurrentBoostStepsS7  AGCCurrentBoostSteps = 1
	AGCCurrentBoostStepsS9  AGCCurrentBoostSteps = 2
	AGCCurrentBoostStepsS11 AGCCurrentBoostSteps = 3
)

// GetAGCCurrentBoostSteps gets the Active Gain Control current boost steps.
//
// This is only valid for the Tic T249.
func (d *Dev) GetAGCCurrentBoostSteps() (AGCCurrentBoostSteps, error) {
	if d.variant != TicT249 {
		return 0, ErrUnsupportedVariant
	}

	v, err := d.getVar8(OffsetAGCCurrentBoostSteps)
	return AGCCurrentBoostSteps(v & 0xF), err
}

// SetAGCCurrentBoostSteps sets the Active Gain Control current boost steps.
//
// This is only valid for the Tic T249.
func (d *Dev) SetAGCCurrentBoostSteps(steps AGCCurrentBoostSteps) error {
	if d.variant != TicT249 {
		return ErrUnsupportedVariant
	}
	if steps > AGCCurrentBoostStepsS11 {
		return ErrInvalidSetting
	}

	return d.commandW7(cmdSetAGCOption, 0x20|(uint8(steps)&0xF))
}

// AGCFrequencyLimit describes the possible Active Gain Control frequency limit
// values.
type AGCFrequencyLimit uint8

const (
	AGCFrequencyLimitOff    AGCFrequencyLimit = 0
	AGCFrequencyLimitF225Hz AGCFrequencyLimit = 1
	AGCFrequencyLimitF450Hz AGCFrequencyLimit = 2
	AGCFrequencyLimitF675Hz AGCFrequencyLimit = 3
)

// GetAGCFrequencyLimit gets the Active Gain Control frequency limit.
//
// This is only valid for the Tic T249.
func (d *Dev) GetAGCFrequencyLimit() (AGCFrequencyLimit, error) {
	if d.variant != TicT249 {
		return 0, ErrUnsupportedVariant
	}

	v, err := d.getVar8(OffsetAGCFrequencyLimit)
	return AGCFrequencyLimit(v & 0xF), err
}

// SetAGCFrequencyLimit sets the Active Gain Control frequency limit.
//
// This is only valid for the Tic T249.
func (d *Dev) SetAGCFrequencyLimit(limit AGCFrequencyLimit) error {
	if d.variant != TicT249 {
		return ErrUnsupportedVariant
	}
	if limit > AGCFrequencyLimitF675Hz {
		return ErrInvalidSetting
	}

	return d.commandW7(cmdSetAGCOption, 0x30|(uint8(limit)&0xF))
}

// OperationState describes the possible operation states for the Tic.
type OperationState uint8

const (
	OperationStateReset             OperationState = 0
	OperationStateDeenergized       OperationState = 2
	OperationStateSoftError         OperationState = 4
	OperationStateWaitingForErrLine OperationState = 6
	OperationStateStartingUp        OperationState = 8
	OperationStateNormal            OperationState = 10
)

// GetOperationState gets the Tic's current operation state, which indicates
// whether it is operating normally or in an error state.
//
// Example:
//
//	state, err := dev.GetOperationState()
//	if state != tic.OperationStateNormal {
//		// There is an error, or the Tic is starting up.
//	}
//
// For more information, see the "Error handling" section of the Tic user's
// guide.
func (d *Dev) GetOperationState() (OperationState, error) {
	v, err := d.getVar8(OffsetOperationState)
	return OperationState(v), err
}

// ticMiscFlags1 describes the bits in the Tic's Misc Flags 1 register.
type ticMiscFlags1 uint8

const (
	ticMiscFlags1Energized          ticMiscFlags1 = 0
	ticMiscFlags1PositionUncertain  ticMiscFlags1 = 1
	ticMiscFlags1ForwardLimitActive ticMiscFlags1 = 2
	ticMiscFlags1ReverseLimitActive ticMiscFlags1 = 3
	ticMiscFlags1HomingActive       ticMiscFlags1 = 4
)

// IsEnergized returns true if the motor driver is energized (trying to send
// current to its outputs).
func (d *Dev) IsEnergized() (bool, error) {
	v, err := d.getVar8(OffsetMiscFlags1)
	return ((v >> uint8(ticMiscFlags1Energized)) & 1) != 0, err
}

// IsPositionUncertain gets a flag that indicates whether there has been
// external confirmation that the value of the Tic's "Current position" variable
// is correct.
//
// For more information, see the "Error handling" section of the Tic user's
// guide.
func (d *Dev) IsPositionUncertain() (bool, error) {
	v, err := d.getVar8(OffsetMiscFlags1)
	return ((v >> uint8(ticMiscFlags1PositionUncertain)) & 1) != 0, err
}

// IsForwardLimitActive returns true if one of the forward limit switches is
// active.
func (d *Dev) IsForwardLimitActive() (bool, error) {
	v, err := d.getVar8(OffsetMiscFlags1)
	return ((v >> uint8(ticMiscFlags1ForwardLimitActive)) & 1) != 0, err
}

// IsReverseLimitActive returns true if one of the reverse limit switches is
// active.
func (d *Dev) IsReverseLimitActive() (bool, error) {
	v, err := d.getVar8(OffsetMiscFlags1)
	return ((v >> uint8(ticMiscFlags1ReverseLimitActive)) & 1) != 0, err
}

// IsHomingActive returns true if the Tic's homing procedure is running.
func (d *Dev) IsHomingActive() (bool, error) {
	v, err := d.getVar8(OffsetMiscFlags1)
	return ((v >> uint8(ticMiscFlags1HomingActive)) & 1) != 0, err
}

// ErrorBit describes the Tic's error bits. See the "Error handling" section of
// the Tic user's guide for more information about what these errors mean.
type ErrorBit uint32

const (
	ErrorBitIntentionallyDeenergized ErrorBit = 0
	ErrorBitMotorDriverError         ErrorBit = 1
	ErrorBitLowVin                   ErrorBit = 2
	ErrorBitKillSwitch               ErrorBit = 3
	ErrorBitRequiredInputInvalid     ErrorBit = 4
	ErrorBitSerialError              ErrorBit = 5
	ErrorBitCommandTimeout           ErrorBit = 6
	ErrorBitSafeStartViolation       ErrorBit = 7
	ErrorBitErrLineHigh              ErrorBit = 8
	ErrorBitSerialFraming            ErrorBit = 16
	ErrorBitRxOverrun                ErrorBit = 17
	ErrorBitFormat                   ErrorBit = 18
	ErrorBitCRC                      ErrorBit = 19
	ErrorBitEncoderSkip              ErrorBit = 20
)

// GetErrorStatus gets the errors that are currently stopping the motor.
//
// Each bit in the returned register represents a different error. The bits are
// defined by the tic.ErrorBit constants.
//
// Example:
//
//	 status, err := dev.GetErrorStatus()
//	 if status&(1<<tic.ErrorBitLowVin) != 0 {
//			// Handle loss of power.
//		}
//
// HasError may be used instead to check for specific errors.
func (d *Dev) GetErrorStatus() (uint16, error) {
	return d.getVar16(OffsetErrorStatus)
}

// HasError returns true if the Tic is in the error state described by the given
// error bit.
//
// Example:
//
//	 isLowVin, err := dev.HasError(tic.ErrorBitLowVin)
//	 if isLowVin {
//			// Handle loss of power.
//	 }
func (d *Dev) HasError(bit ErrorBit) (bool, error) {
	status, err := d.GetErrorStatus()
	return status&(1<<bit) != 0, err
}

// GetErrorsOccurred gets the errors that have occurred since the last time this
// function was called.
//
// Note that the Tic Control Center constantly clears the bits in this
// register, so if you are running the Tic Control Center then you will not
// be able to reliably detect errors with this function.
//
// Each bit in the returned register represents a different error. The bits
// are defined by the tic.Errorbit constants.
//
// Example:
//
//	errors, err := dev.GetErrorsOccurred()
//	if errors&(1<<tic.ErrorBitMotorDriverError) != 0 {
//		// Handle the motor driver error.
//	}
func (d *Dev) GetErrorsOccurred() (uint32, error) {
	const length = 4
	buffer, err := d.getSegment(
		cmdGetVariableAndClearErrors, OffsetErrorsOccurred, length,
	)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(buffer), nil
}

// PlanningMode describes the possible planning modes for the Tic's step
// generation code.
type PlanningMode uint8

const (
	PlanningModeOff            PlanningMode = 0
	PlanningModeTargetPosition PlanningMode = 1
	PlanningModeTargetVelocity PlanningMode = 2
)

// GetPlanningMode returns the current planning mode for the Tic's step
// generation code.
//
// This tells us whether the Tic is sending steps, and if it is sending steps,
// tells us whether it is in Target Position or Target Velocity mode.
//
// Example:
//
//	mode, err := dev.GetPlanningMode()
//	if mode == tic.PlanningModeTargetPosition {
//		// The Tic is moving the stepper motor to a target position, or has
//	    // already reached it and is at rest.
//	}
func (d *Dev) GetPlanningMode() (PlanningMode, error) {
	v, err := d.getVar8(OffsetPlanningMode)
	return PlanningMode(v), err
}

// GetCurrentPosition gets the current position of the stepper motor, in
// microsteps.
//
// Note that this just tracks steps that the Tic has commanded the stepper
// driver to take, which could be different from the actual position of the
// motor.
func (d *Dev) GetCurrentPosition() (int32, error) {
	v, err := d.getVar32(OffsetCurrentPosition)
	return int32(v), err
}

// GetCurrentVelocity gets the current velocity of the stepper motor, in
// microsteps per 10000 seconds.
//
// Note that this is just the velocity used in the Tic's step planning
// algorithms, and it might not correspond to the actual velocity of the motor.
func (d *Dev) GetCurrentVelocity() (int32, error) {
	v, err := d.getVar32(OffsetCurrentVelocity)
	return int32(v), err
}

// GetActingTargetPosition gets the acting target position, in microsteps.
//
// This is a variable used in the Tic's target position step planning algorithm,
// and it could be invalid while the motor is stopped.
//
// This is mainly intended for getting insight into how the Tic's algorithms
// work or troubleshooting issues, and most people should not use this.
func (d *Dev) GetActingTargetPosition() (uint32, error) {
	return d.getVar32(OffsetActingTargetPosition)
}

// GetTimeSinceLastStep gets the time since the last step, in timer ticks.
//
// Each timer tick represents one third of a microsecond. The Tic only updates
// this variable every 5 milliseconds or so, and it could be invalid while the
// motor is stopped.
//
// This is mainly intended for getting insight into how the Tic's algorithms
// work or troubleshooting issues, and most people should not use this.
func (d *Dev) GetTimeSinceLastStep() (uint32, error) {
	return d.getVar32(OffsetTimeSinceLastStep)
}

// ResetCause describes the possible causes of a full microcontroller reset for
// the Tic.
type ResetCause uint8

const (
	ResetCausePowerUp        = 0
	ResetCauseBrownout       = 1
	ResetCauseResetLine      = 2
	ResetCauseWatchdog       = 4
	ResetCauseSoftware       = 8
	ResetCauseStackOverflow  = 16
	ResetCauseStackUnderflow = 32
)

// GetDeviceReset gets the cause of the controller's last full microcontroller
// reset.
//
// Example:
//
//	reset, err := dev.GetDeviceReset()
//	if reset == tic.ResetCauseBrownout {
//		// There was a brownout reset - the power supply could not keep up.
//	}
//
// The Reset command (Reset()) does not affect this variable.
func (d *Dev) GetDeviceReset() (ResetCause, error) {
	v, err := d.getVar8(OffsetDeviceReset)
	return ResetCause(v), err
}

// GetVoltageIn gets the current measurement of the VIN voltage, in millivolts.
func (d *Dev) GetVoltageIn() (physic.ElectricPotential, error) {
	mv, err := d.getVar16(OffsetVoltageIn)
	return physic.ElectricPotential(mv) * physic.MilliVolt, err
}

// GetUpTime gets the time since the last full reset of the Tic's
// microcontroller, in milliseconds.
//
// A Reset command (Reset()) does not count.
func (d *Dev) GetUpTime() (time.Duration, error) {
	ms, err := d.getVar32(OffsetUpTime)
	return time.Duration(ms) * time.Millisecond, err
}

// GetEncoderPosition gets the raw encoder count measured from the Tic's RX and
// TX lines.
func (d *Dev) GetEncoderPosition() (int32, error) {
	v, err := d.getVar32(OffsetEncoderPosition)
	return int32(v), err
}

// GetRCPulseWidth gets the raw pulse width measured on the Tic's RC input, in
// units of twelfths of a microsecond.
//
// Returns tic.InputNull if the RC input is missing or invalid.
//
// Example:
//
//	width, err := dev.GetRCPulseWidth()
//	if width != tic.InputNull && width > 1500*12 {
//		// Pulse width is greater than 1500 microseconds.
//	}
func (d *Dev) GetRCPulseWidth() (uint16, error) {
	return d.getVar16(OffsetRCPulseWidth)
}

// Pin describes a Tic control pin.
type Pin uint8

const (
	PinSCL Pin = 0
	PinSDA Pin = 1
	PinTX  Pin = 2
	PinRX  Pin = 3
	PinRC  Pin = 4
)

// GetAnalogReading gets the analog reading from the specified pin.
//
// The reading is left-justified, so 0xFFFF represents a voltage equal to the
// Tic's 5V pin (approximately 4.8 V).
//
// Returns tic.InputNull if the analog reading is disabled or not ready.
//
// Example:
//
//	reading, err := dev.GetAnalogReading(tic.PinSDA)
//	if reading != tic.InputNull && reading < 32768 {
//		// The reading is less than about 2.4 V.
//	}
func (d *Dev) GetAnalogReading(pin Pin) (uint16, error) {
	return d.getVar16(OffsetAnalogReadingSCL + 2*offset(pin))
}

// IsDigitalReading gets a digital reading from the specified pin.
//
// Returns true for high and false for low.
func (d *Dev) IsDigitalReading(pin Pin) (bool, error) {
	readings, err := d.getVar8(OffsetDigitalReadings)
	return ((readings >> pin) & 1) != 0, err
}

// PinState describes a Tic's pin state.
type PinState uint8

const (
	PinStateHighImpedance PinState = 0
	PinStateInputPullUp   PinState = 1
	PinStateOutputLow     PinState = 2
	PinStateOutputHigh    PinState = 3
)

// GetPinState gets the current state of a pin, i.e. what kind of input or
// output it is.
//
// Note that the state might be misleading if the pin is being used as an I²C
// or serial pin.
//
// Example:
//
//	state, err := dev.GetPinState(PinSCL)
//	if state == tic.PinStateOutputHigh {
//		// SCL is driving high.
//	}
func (d *Dev) GetPinState(pin Pin) (PinState, error) {
	if pin > PinRX {
		// State not available for PinRC.
		return 0, ErrInvalidSetting
	}
	states, err := d.getVar8(OffsetPinStates)
	return PinState((states >> (2 * uint8(pin))) & 0b11), err
}

// InputState describes the possible states of the Tic's main input.
type InputState uint8

const (
	// The input is not ready yet. More samples are needed, or a command has not
	// been received yet.
	InputStateNotReady InputState = 0
	// The input is invalid.
	InputStateInvalid InputState = 1
	// The input is valid and is telling the Tic to halt the motor.
	InputStateHalt InputState = 2
	// The input is valid and is telling the Tic to go to a target position,
	// which you can get with GetInputAfterScaling().
	InputStatePosition InputState = 3
	// The input is valid and is telling the Tic to go to a target velocity,
	// which you can get with GetInputAfterScaling().
	InputStateVelocity InputState = 4
)

// GetInputState gets the current state of the Tic's main input.
//
// Example:
//
//	state, err := dev.GetInputState()
//	if state == tic.InputStatePosition {
//	 	// The Tic's input is specifying a target position.
//	}
func (d *Dev) GetInputState() (InputState, error) {
	v, err := d.getVar8(OffsetInputState)
	return InputState(v), err
}

// GetInputAfterAveraging gets a variable used in the process that converts raw
// RC and analog values into a motor position or speed. This is mainly for
// debugging your input scaling settings in RC or analog mode.
//
// A value of tic.InputNull means the input value is not available.
func (d *Dev) GetInputAfterAveraging() (uint16, error) {
	return d.getVar16(OffsetInputAfterAveraging)
}

// GetInputAfterHysteresis gets a variable used in the process that converts raw
// RC and analog values into a motor position or speed. This is mainly for
// debugging your input scaling settings in RC or analog mode.
//
// A value of tic.InputNull means the input value is not available.
func (d *Dev) GetInputAfterHysteresis() (uint16, error) {
	return d.getVar16(OffsetInputAfterHysteresis)
}

// GetInputAfterScaling gets the value of the Tic's main input after scaling has
// been applied.
//
// If the input is valid, this number is the target position or target velocity
// specified by the input.
func (d *Dev) GetInputAfterScaling() (int32, error) {
	v, err := d.getVar32(OffsetInputAfterScaling)
	return int32(v), err
}

// MotorDriverError describes the possible motor driver errors for the Tic T249.
type MotorDriverError uint8

const (
	MotorDriverErrorNone            MotorDriverError = 0
	MotorDriverErrorOverCurrent     MotorDriverError = 1
	MotorDriverErrorOverTemperature MotorDriverError = 2
)

// GetLastMotorDriverError gets the cause of the last motor driver error.
//
// This is only valid for the Tic T249.
func (d *Dev) GetLastMotorDriverError() (MotorDriverError, error) {
	if d.variant != TicT249 {
		return 0, ErrUnsupportedVariant
	}

	v, err := d.getVar8(OffsetLastMotorDriverError)
	return MotorDriverError(v), err
}

// GetLastHPDriverErrors gets the "Last HP driver errors" variable.
//
// Each bit in this register represents an error. If the bit is 1, the error was
// one of the causes of the Tic's last motor driver error.
//
// This is only valid for the Tic 36v4.
func (d *Dev) GetLastHPDriverErrors() (uint8, error) {
	if d.variant != Tic36v4 {
		return 0, ErrUnsupportedVariant
	}

	return d.getVar8(OffsetLastHPDriverErrors)
}

// GetSetting gets a contiguous block of settings from the Tic's EEPROM.
//
// The maximum length that can be fetched is 15 bytes.
//
// This library does not attempt to interpret the settings and say what they
// mean. If you are interested in how the settings are encoded in the Tic's
// EEPROM, see the "Settings reference" section of the Tic user's guide.
func (d *Dev) GetSetting(offset offset, length uint) ([]uint8, error) {
	if length > 15 {
		return nil, errors.New("maximum length exceeded")
	}

	return d.getSegment(cmdGetSetting, offset, length)
}

// offset represents where settings are stored in the Tic's EEPROM memory. See
// the "Variable reference" section of the Tic user's guide for details.
type offset uint8

const (
	OffsetOperationState        offset = 0x00 // uint8 return type
	OffsetMiscFlags1            offset = 0x01 // uint8 return type
	OffsetErrorStatus           offset = 0x02 // uint16 return type
	OffsetErrorsOccurred        offset = 0x04 // uint32 return type
	OffsetPlanningMode          offset = 0x09 // uint8 return type
	OffsetTargetPosition        offset = 0x0A // int32 return type
	OffsetTargetVelocity        offset = 0x0E // int32 return type
	OffsetStartingSpeed         offset = 0x12 // uint32 return type
	OffsetSpeedMax              offset = 0x16 // uint32 return type
	OffsetDecelMax              offset = 0x1A // uint32 return type
	OffsetAccelMax              offset = 0x1E // uint32 return type
	OffsetCurrentPosition       offset = 0x22 // int32 return type
	OffsetCurrentVelocity       offset = 0x26 // int32 return type
	OffsetActingTargetPosition  offset = 0x2A // int32 return type
	OffsetTimeSinceLastStep     offset = 0x2E // uint32 return type
	OffsetDeviceReset           offset = 0x32 // uint8 return type
	OffsetVoltageIn             offset = 0x33 // uint16 return type
	OffsetUpTime                offset = 0x35 // uint32 return type
	OffsetEncoderPosition       offset = 0x39 // int32 return type
	OffsetRCPulseWidth          offset = 0x3D // uint16 return type
	OffsetAnalogReadingSCL      offset = 0x3F // uint16 return type
	OffsetAnalogReadingSDA      offset = 0x41 // uint16 return type
	OffsetAnalogReadingTX       offset = 0x43 // uint16 return type
	OffsetAnalogReadingRX       offset = 0x45 // uint16 return type
	OffsetDigitalReadings       offset = 0x47 // uint8 return type
	OffsetPinStates             offset = 0x48 // uint8 return type
	OffsetStepMode              offset = 0x49 // uint8 return type
	OffsetCurrentLimit          offset = 0x4A // uint8 return type
	OffsetDecayMode             offset = 0x4B // uint8 return type
	OffsetInputState            offset = 0x4C // uint8 return type
	OffsetInputAfterAveraging   offset = 0x4D // uint16 return type
	OffsetInputAfterHysteresis  offset = 0x4F // uint16 return type
	OffsetInputAfterScaling     offset = 0x51 // uint16 return type
	OffsetLastMotorDriverError  offset = 0x55 // uint8 return type
	OffsetAGCMode               offset = 0x56 // uint8 return type
	OffsetAGCBottomCurrentLimit offset = 0x57 // uint8 return type
	OffsetAGCCurrentBoostSteps  offset = 0x58 // uint8 return type
	OffsetAGCFrequencyLimit     offset = 0x59 // uint8 return type
	OffsetLastHPDriverErrors    offset = 0xFF // uint8 return type
)

// command represents Tic command codes which are used for its I²C interface.
// See the "Command reference" section of the Tic user's guide for details.
type command uint8

const (
	cmdSetTargetPosition         command = 0xE0
	cmdSetTargetVelocity         command = 0xE3
	cmdHaltAndSetPosition        command = 0xEC
	cmdHaltAndHold               command = 0x89
	cmdGoHome                    command = 0x97
	cmdResetCommandTimeout       command = 0x8C
	cmdDeenergize                command = 0x86
	cmdEnergize                  command = 0x85
	cmdExitSafeStart             command = 0x83
	cmdEnterSafeStart            command = 0x8F
	cmdReset                     command = 0xB0
	cmdClearDriverError          command = 0x8A
	cmdSetSpeedMax               command = 0xE6
	cmdSetStartingSpeed          command = 0xE5
	cmdSetAccelMax               command = 0xEA
	cmdSetDecelMax               command = 0xE9
	cmdSetStepMode               command = 0x94
	cmdSetCurrentLimit           command = 0x91
	cmdSetDecayMode              command = 0x92
	cmdSetAGCOption              command = 0x98
	cmdGetVariable               command = 0xA1
	cmdGetVariableAndClearErrors command = 0xA2
	cmdGetSetting                command = 0xA8
)

var _ conn.Resource = &Dev{}
var _ fmt.Stringer = &Dev{}
