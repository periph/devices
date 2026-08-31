package firmata

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"periph.io/x/conn/v3/physic"
)

var (
	Err10BitAddressingNotSupported = errors.New("10-bit addressing not supported")
)

type I2CBus struct {
	c  ClientI
	mu sync.Mutex
}

func newI2CBus(c ClientI) (*I2CBus, error) {
	b := &I2CBus{
		c: c,
	}

	err := c.SendI2CConfig(0)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (b *I2CBus) Close() error {
	panic("implement me")
}

func (b *I2CBus) String() string {
	panic("implement me")
}

func (b *I2CBus) Tx(addr uint16, w, r []byte) error {
	if addr >= 0b11111111 {
		return fmt.Errorf("%w: 0x%04X", Err10BitAddressingNotSupported, addr)
	}

	if len(r) > math.MaxUint16 {
		return fmt.Errorf("%w: cannot reach more than %d bytes", ErrValueOutOfRange, math.MaxUint16)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan I2CPacket)
	defer close(ch)

	address := uint8(addr)

	release, err := b.c.SetI2CAddressListener(address, ch)
	if err != nil {
		return err
	}
	defer release()

	if len(w) > 0 {
		if err := b.c.WriteI2CData(address, false, w); err != nil {
			return err
		}
	}

	if len(r) > 0 {
		if err := b.c.ReadI2CData(address, false, uint16(len(r))); err != nil {
			return err
		}

		pck := <-ch

		copy(r, pck.Data)
	}

	return nil
}

func (b *I2CBus) SetSpeed(f physic.Frequency) error {
	return fmt.Errorf("%w: firmata does not support setting bus frequency", ErrUnsupportedFeature)
}
