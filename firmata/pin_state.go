package firmata

import (
	"fmt"

	"periph.io/x/conn/v3/pin"
)

type PinStateResponse struct {
	Pin   uint8
	Mode  pin.Func
	State int
}

func ParsePinStateResponse(data []byte) PinStateResponse {
	var response = PinStateResponse{
		Pin:   data[0],
		Mode:  pinModeToFuncMap[data[1]],
		State: 0,
	}

	for i, b := range data[2:] {
		response.State |= int(b << (i * 7))
	}

	return response
}

func (p PinStateResponse) String() string {
	return fmt.Sprintf("pin(%d) mode(%s) state(%d)", p.Pin, p.Mode, p.State)
}
