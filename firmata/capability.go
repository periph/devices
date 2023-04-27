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

func ParseCapabilityResponse(data []byte) CapabilityResponse {
	var response = CapabilityResponse{
		PinToModeToResolution: []map[pin.Func]uint8{{}},
		SupportedPinModes:     [][]pin.Func{{}},
	}

	var pindex = 0
	for i := 0; i < len(data); {
		if data[i] == CapabilityResponsePinDelimiter {
			response.PinToModeToResolution = append(response.PinToModeToResolution, map[pin.Func]uint8{})
			response.SupportedPinModes = append(response.SupportedPinModes, []pin.Func{})
			i += 1
			pindex++
		} else {
			pinFunc := pinModeToFuncMap[data[i]]
			response.PinToModeToResolution[pindex][pinFunc] = data[i+1]
			response.SupportedPinModes[pindex] = append(response.SupportedPinModes[pindex], pinFunc)
			i += 2
		}
	}

	return response
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
