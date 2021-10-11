package firmata

import (
	"context"
	"errors"
	"sync"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/pin"
)

var (
	ErrUnsupportedGPIOPull = errors.New("firmata: PullDown is not supported")
	ErrNoMatchingGPIOPull  = errors.New("firmata: pin was previously in a non-input mode")
)

type Pin struct {
	c          ClientI
	pin        uint8
	edge       gpio.Edge
	ch         chan gpio.Level
	release    func()
	done       chan struct{}
	valueLast  gpio.Level
	valueNew   gpio.Level
	edgeChange chan gpio.Edge
	mu         sync.Mutex
}

func newPin(c ClientI, num uint8) *Pin {
	p := &Pin{
		c:   c,
		pin: num,
		ch:  make(chan gpio.Level),
	}

	go p.run()

	return p
}

func (p *Pin) run() {
	p.done = make(chan struct{})

	for {
		select {
		case <-p.done:
			close(p.ch)
			return
		case v := <-p.ch:
			p.valueLast = p.valueNew
			p.valueNew = v

			func() {
				p.mu.Lock()
				defer p.mu.Unlock()

				if p.edgeChange == nil {
					return
				}

				if p.valueLast == true || p.valueNew == false {
					p.edgeChange <- gpio.FallingEdge
				}
				if p.valueLast == false || p.valueNew == true {
					p.edgeChange <- gpio.RisingEdge
				}
			}()
		}
	}
}

func (p *Pin) In(pull gpio.Pull, edge gpio.Edge) error {
	var mode = PinFuncDigitalInput
	switch pull {
	case gpio.PullDown:
		return ErrUnsupportedGPIOPull
	case gpio.PullNoChange:
		ch, err := p.c.PinStateQuery(p.pin)
		if err != nil {
			return err
		}

		s := <-ch
		mode = s.Mode

		switch mode {
		case PinFuncInputPullUp:
		case PinFuncDigitalInput:
		default:
			return ErrNoMatchingGPIOPull
		}
	case gpio.PullUp:
		mode = PinFuncInputPullUp
	case gpio.Float:
		mode = PinFuncDigitalInput
	}

	if err := p.c.SetPinMode(p.pin, mode); err != nil {
		return err
	}

	var err error
	if p.release, err = p.c.SetDigitalIOMessageListener(p.pin, p.ch); err != nil {
		return err
	}

	if err = p.c.SetDigitalPinReporting(p.pin, true); err != nil {
		return err
	}

	p.edge = edge

	return nil
}

func (p *Pin) Read() gpio.Level {
	return p.valueNew
}

func (p *Pin) WaitForEdge(timeout time.Duration) bool {
	defer func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		close(p.edgeChange)
		p.edgeChange = nil
	}()

	func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		p.edgeChange = make(chan gpio.Edge)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case change := <-p.edgeChange:
			if p.edge == gpio.BothEdges || change == p.edge {
				return true
			}
		case <-ctx.Done():
			return false
		}
	}
}

func (p *Pin) Pull() gpio.Pull {
	ch, err := p.c.PinStateQuery(p.pin)
	if err != nil {
		return gpio.PullNoChange
	}

	s := <-ch
	switch s.Mode {
	case PinFuncInputPullUp:
		return gpio.PullUp
	case PinFuncDigitalInput:
		return gpio.Float
	}

	return gpio.PullNoChange
}

func (p *Pin) DefaultPull() gpio.Pull {
	return gpio.PullNoChange
}

func (p *Pin) Out(l gpio.Level) error {
	// No need to set mode as firmata does so automatically
	return p.c.SetDigitalPinValue(p.pin, l)
}

const dutyMax gpio.Duty = 1<<8 - 1

// PWM ignores physic.Frequency as there is no way to set it through firmata
func (p *Pin) PWM(duty gpio.Duty, f physic.Frequency) error {
	// No need to set mode as firmata does so automatically
	// PWM duty scaled down from 24 to 8 bits
	return p.c.ExtendedReportAnalogPin(p.pin, uint8((duty>>16)&0xFF))
}

func (p *Pin) Func() pin.Func {
	ch, err := p.c.PinStateQuery(p.pin)
	if err != nil {
		return pin.FuncNone
	}

	s := <-ch

	return s.Mode
}

func (p *Pin) SetFunc(f pin.Func) error {
	return p.c.SetPinMode(p.pin, f)
}

func (p *Pin) SupportedFuncs() []pin.Func {
	return p.c.GetPinFunctions(p.pin)
}

func (p *Pin) Halt() error {
	close(p.done)
	p.release()
	if err := p.c.SetDigitalPinReporting(p.pin, false); err != nil {
		return err
	}
	if err := p.c.SetAnalogPinReporting(p.pin, false); err != nil {
		return err
	}
	if err := p.c.SetDigitalPinValue(p.pin, gpio.Low); err != nil {
		return err
	}
	return nil
}

func (p *Pin) Name() string {
	return p.c.GetPinName(p.pin)
}

func (p *Pin) String() string {
	return p.Name()
}

func (p *Pin) Number() int {
	return int(p.pin)
}

func (p *Pin) Function() string {
	return string(p.Func())
}
