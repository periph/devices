package firmata

import (
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
)

type I2CBus struct {
}

func (i *I2CBus) SCL() gpio.PinIO {
	panic("implement me")
}

func (i *I2CBus) SDA() gpio.PinIO {
	panic("implement me")
}

func (i *I2CBus) Close() error {
	panic("implement me")
}

func (i *I2CBus) String() string {
	panic("implement me")
}

func (i *I2CBus) Tx(addr uint16, w, r []byte) error {
	panic("implement me")
}

func (i *I2CBus) SetSpeed(f physic.Frequency) error {
	panic("implement me")
}
