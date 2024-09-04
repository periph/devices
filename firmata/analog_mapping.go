package firmata

import (
	"bytes"
	"fmt"
)

type AnalogMappingResponse struct {
	AnalogPinToDigital []uint8
	DigitalPinToAnalog map[uint8]uint8
}

func ParseAnalogMappingResponse(data []byte) AnalogMappingResponse {
	var response = AnalogMappingResponse{
		AnalogPinToDigital: []uint8{},
		DigitalPinToAnalog: map[uint8]uint8{},
	}

	for i := 0; i < len(data); i++ {
		if data[i] != CapabilityResponsePinDelimiter {
			response.DigitalPinToAnalog[uint8(i)] = uint8(len(response.AnalogPinToDigital))
			response.AnalogPinToDigital = append(response.AnalogPinToDigital, uint8(i))
		}
	}

	return response
}

func (a AnalogMappingResponse) String() string {
	str := bytes.Buffer{}
	for analogPin, digitalPin := range a.AnalogPinToDigital {
		_, _ = fmt.Fprintf(&str, "A%d: %d\n", analogPin, digitalPin)
	}
	return str.String()
}

type ExtendedAnalogMappingResponse struct {
	Pin uint8
}

func (a ExtendedAnalogMappingResponse) String() string {
	return fmt.Sprintf("%d", a.Pin)
}
