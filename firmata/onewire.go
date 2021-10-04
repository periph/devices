package firmata

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/onewire"
)

type OneWireBus struct {
	c      ClientI
	q      *Pin
	mu     sync.Mutex
	closer func() error
	cid    uint16
}

func newOneWireBus(c ClientI, q *Pin, closer func() error) *OneWireBus {
	return &OneWireBus{
		c:      c,
		q:      q,
		closer: closer,
	}
}

func (b *OneWireBus) Close() error {
	return b.closer()
}

func (b *OneWireBus) String() string {
	panic("implement me")
}

func (b *OneWireBus) Search(alarmOnly bool) ([]onewire.Address, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan []byte)
	defer close(ch)

	release, err := b.c.SetOneWireListener(b.q.pin, ch)
	if err != nil {
		return nil, err
	}
	defer release()

	if alarmOnly {
		if _, err := b.c.SendSysEx(SysExOneWireData, byte(OneWireInstructionSearchAlarmed), b.q.pin); err != nil {
			return nil, err
		}
	} else {
		if _, err := b.c.SendSysEx(SysExOneWireData, byte(OneWireInstructionSearch), b.q.pin); err != nil {
			return nil, err
		}
	}

	data := <-ch

	ins := OneWireInstruction(data[0])
	pin := data[1]

	if ins != OneWireInstructionSearchAlarmedReply && ins != OneWireInstructionSearchReply {
		return nil, fmt.Errorf("%w: did not receive search reply", ErrUnhandledMessage)
	}
	if pin != b.q.pin {
		return nil, fmt.Errorf("%w: received message from wrong bus", ErrInvalidOneWirePin)
	}

	data = Decoder7Bit(data[2:])

	addresses := make([]onewire.Address, len(data)/8)
	for i := range addresses {
		addresses[i] = onewire.Address(binary.LittleEndian.Uint64(data[i*8:]))
	}
	return addresses, nil
}

func (b *OneWireBus) Tx(w, r []byte, power onewire.Pullup) error {
	if len(w) == 0 && len(r) == 0 {
		return nil
	}

	if len(r) > math.MaxUint16 {
		return fmt.Errorf("%w: cannot reach more than %d bytes", ErrValueOutOfRange, math.MaxUint16)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan []byte)
	defer close(ch)

	release, err := b.c.SetOneWireListener(b.q.pin, ch)
	if err != nil {
		return err
	}
	defer release()

	var powerVal byte = 0x00
	if power {
		powerVal = 0x01
	}

	if _, err := b.c.SendSysEx(SysExOneWireData, byte(OneWireInstructionConfigure), b.q.pin, powerVal); err != nil {
		return err
	}

	cmd := OneWireCommandReset

	if len(w) > 0 {
		switch w[0] {
		case 0xF0: // search rom
			if _, err := b.c.SendSysEx(SysExOneWireData, byte(OneWireInstructionSearch), b.q.pin); err != nil {
				return err
			}
		case 0xEC: // search rom (alarmed)
			if _, err := b.c.SendSysEx(SysExOneWireData, byte(OneWireInstructionSearchAlarmed), b.q.pin); err != nil {
				return err
			}
		case 0xCC: // skip rom
			panic("not implemented")
		case 0x33: // read rom
			panic("not implemented")
		case 0x55: // match rom
			cmd |= OneWireCommandSelect

			payload := w[1:9]

			if len(w) > 9 { // command length (1) + address length (8)
				cmd |= OneWireCommandWrite
			}
			if len(r) > 0 {
				cmd |= OneWireCommandRead
				// Write the amount of bytes to read.
				payload = append(payload, byte(len(r)&0xFF), byte((len(r)>>8)&0xFF))
				// Write the correlation id for sanity checking.
				payload = append(payload, byte(b.cid&0xFF), byte((b.cid>>8)&0xFF))
				b.cid++
			}

			cmd |= OneWireCommandDelay
			payload = append(payload, 10, 0, 0, 0)

			payload = append(payload, w[9:]...)
			payload = append([]byte{byte(cmd), b.q.pin}, Encoder7Bit(payload)...)
			if _, err := b.c.SendSysEx(SysExOneWireData, payload...); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: no onewire command matching 0x%02X", ErrUnsupportedFeature, w[0])
		}
	}

	if len(r) > 0 {
		data := <-ch

		ins := OneWireInstruction(data[0])
		pin := data[1]

		if ins != OneWireInstructionReadReply {
			return fmt.Errorf("%w: did not receive read reply", ErrUnhandledMessage)
		}
		if pin != b.q.pin {
			return fmt.Errorf("%w: received message from wrong bus", ErrInvalidOneWirePin)
		}

		data = Decoder7Bit(data[2:])

		// Drop the correlation id when copying.
		copy(r, data[2:])
	}

	return nil
}

func (b *OneWireBus) Q() gpio.PinIO {
	return b.q
}
