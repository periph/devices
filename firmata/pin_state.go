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

func (p PinStateResponse) String() string {
	return fmt.Sprintf("pin(%d) mode(%s) state(%d)", p.Pin, p.Mode, p.State)
}
