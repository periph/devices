package firmata

import (
	"bytes"
	"fmt"
)

type AnalogMappingResponse struct {
	AnalogPinToDigital []uint8
	DigitalPinToAnalog map[uint8]uint8
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
