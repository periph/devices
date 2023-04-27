package firmata

import (
	"periph.io/x/conn/v3/pin"
)

const (
	PinFuncDigitalInput  pin.Func = "Digital Input"
	PinFuncDigitalOutput pin.Func = "Digital Output"
	PinFuncAnalogInput   pin.Func = "Analog Input"
	PinFuncPWM           pin.Func = "PWM"
	PinFuncServo         pin.Func = "Servo"
	PinFuncShift         pin.Func = "Shift"
	PinFuncI2C           pin.Func = "I2C"
	PinFuncOneWire       pin.Func = "OneWire"
	PinFuncStepper       pin.Func = "Stepper"
	PinFuncEncoder       pin.Func = "Encoder"
	PinFuncSerial        pin.Func = "Serial"
	PinFuncInputPullUp   pin.Func = "Input Pull-Up"
	PinFuncSPI           pin.Func = "SPI"
	PinFuncSonar         pin.Func = "Sonar"
	PinFuncTone          pin.Func = "Tone"
	PinFuncDHT           pin.Func = "DHT"
)

var pinFuncToModeMap = map[pin.Func]uint8{
	PinFuncDigitalInput:  0x0,
	PinFuncDigitalOutput: 0x1,
	PinFuncAnalogInput:   0x2,
	PinFuncPWM:           0x3,
	PinFuncServo:         0x4,
	PinFuncShift:         0x5,
	PinFuncI2C:           0x6,
	PinFuncOneWire:       0x7,
	PinFuncStepper:       0x8,
	PinFuncEncoder:       0x9,
	PinFuncSerial:        0xA,
	PinFuncInputPullUp:   0xB,
	PinFuncSPI:           0xC,
	PinFuncSonar:         0xD,
	PinFuncTone:          0xE,
	PinFuncDHT:           0xF,
}

var pinModeToFuncMap = map[uint8]pin.Func{
	0x0: PinFuncDigitalInput,
	0x1: PinFuncDigitalOutput,
	0x2: PinFuncAnalogInput,
	0x3: PinFuncPWM,
	0x4: PinFuncServo,
	0x5: PinFuncShift,
	0x6: PinFuncI2C,
	0x7: PinFuncOneWire,
	0x8: PinFuncStepper,
	0x9: PinFuncEncoder,
	0xA: PinFuncSerial,
	0xB: PinFuncInputPullUp,
	0xC: PinFuncSPI,
	0xD: PinFuncSonar,
	0xE: PinFuncTone,
	0xF: PinFuncDHT,
}
