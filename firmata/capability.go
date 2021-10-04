package firmata

import (
	"bytes"
	"fmt"

	"periph.io/x/conn/v3/pin"
)

var pinModeOrder = []pin.Func{
	PinFuncDigitalInput,
	PinFuncDigitalOutput,
	PinFuncAnalogInput,
	PinFuncPWM,
	PinFuncServo,
	PinFuncShift,
	PinFuncI2C,
	PinFuncOneWire,
	PinFuncStepper,
	PinFuncEncoder,
	PinFuncSerial,
	PinFuncInputPullUp,
	PinFuncSPI,
	PinFuncSonar,
	PinFuncTone,
	PinFuncDHT,
}

const CapabilityResponsePinDelimiter = 0x7F

type CapabilityResponse struct {
	PinToModeToResolution []map[pin.Func]uint8
	SupportedPinModes     [][]pin.Func
}

func (c CapabilityResponse) String() string {
	str := bytes.Buffer{}
	for p, modeMap := range c.PinToModeToResolution {
		_, _ = fmt.Fprintf(&str, "pin %2v: [", p)
		if len(modeMap) > 0 {
			for _, mode := range pinModeOrder {
				if resolution, ok := modeMap[mode]; ok {
					_, _ = fmt.Fprintf(&str, "%s: %d, ", mode, resolution)
				}
			}
			str.Truncate(str.Len() - 2)
		}
		_, _ = fmt.Fprintf(&str, "]\n")
	}
	return str.String()
}
