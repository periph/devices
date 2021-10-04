package firmata

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sync"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/onewire"
	"periph.io/x/conn/v3/pin"
)

// These max values are for data bytes as, within firmata, data is 7 bits long.
const (
	MaxUInt8  uint8  = (1<<8 - 1) >> 1
	MaxUInt16 uint16 = (1<<16 - 1) >> 2
)

var commandResponseMap = map[SysExCmd]SysExCmd{
	SysExAnalogMappingQuery: SysExAnalogMappingResponse,
	SysExCapabilityQuery:    SysExCapabilityResponse,
	SysExPinStateQuery:      SysExPinStateResponse,
}

type ClientI interface {
	SendSysEx(SysExCmd, ...byte) (chan []byte, error)
	SendReset() error
	ExtendedReportAnalogPin(uint8, int) error
	CapabilityQuery() (chan CapabilityResponse, error)
	PinStateQuery(uint8) (chan PinStateResponse, error)
	ReportFirmware() (chan FirmwareReport, error)
	SetPinMode(uint8, pin.Func) error
	SetAnalogPinReporting(uint8, bool) error
	SetDigitalPinReporting(uint8, bool) error
	SetDigitalPortReporting(uint8, bool) error
	SetSamplingInterval(uint16) error
	SetDigitalPinValue(p uint8, value gpio.Level) error
	SendAnalogMappingQuery() (chan AnalogMappingResponse, error)
	AnalogPinToDigitalPin(p uint8) (uint8, error)
	SetAnalogIOMessageListener(p uint8, ch chan uint16) (release func(), err error)
	SetDigitalIOMessageListener(p uint8, ch chan gpio.Level) (release func(), err error)
	SendAnalogIOMessage(uint8, uint16) error

	OpenI2CBus() (i2c.Bus, error)
	SetI2CAddressListener(addr uint8, ch chan I2CPacket) (release func(), err error)
	WriteI2CData(address uint8, restart bool, data []uint8) error
	ReadI2CData(address uint8, restart bool, len uint16) error
	ReadI2CRegister(address uint8, restart bool, register uint8, len uint16) error
	SendI2CConfig(delayMicroseconds uint8) error

	OpenOneWireBus(p uint8) (bus onewire.BusCloser, err error)
	SetOneWireListener(uint8, chan []byte) (release func(), err error)

	GetPinName(uint8) string
	GetPinFunctions(uint8) []pin.Func

	Close() error
}

type Client struct {
	board                 io.ReadWriteCloser
	responseChannels      map[SysExCmd][]chan []byte
	sysExListenerChannels map[SysExCmd]chan []byte

	i2cListeners map[uint8]chan I2CPacket
	i2cMU        sync.Mutex

	onewireListeners map[uint8]chan []byte
	onewireMU        sync.Mutex

	digitalIOMessageChannels map[uint8]chan gpio.Level
	digitalPinMU             sync.Mutex
	analogIOMessageChannels  map[uint8]chan uint16
	analogPinMU              sync.Mutex

	mu         sync.Mutex
	started    bool
	i2cStarted bool

	// We want to report these to the requester, but also save them for internal use.
	cr  CapabilityResponse
	amr AnalogMappingResponse
}

func NewClient(board io.ReadWriteCloser) *Client {
	return &Client{
		board:                 board,
		responseChannels:      map[SysExCmd][]chan []byte{},
		sysExListenerChannels: map[SysExCmd]chan []byte{},
		i2cListeners:          map[uint8]chan I2CPacket{},

		onewireListeners: map[uint8]chan []byte{},

		digitalIOMessageChannels: map[uint8]chan gpio.Level{},
		analogIOMessageChannels:  map[uint8]chan uint16{},
	}
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.started = false
	return c.board.Close()
}

type flusher interface {
	Flush()
}

type flusherErr interface {
	Flush() error
}

func (c *Client) Start() error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return ErrAlreadyStarted
	}
	c.started = true
	c.mu.Unlock()

	if b, ok := c.board.(flusher); ok {
		b.Flush()
	} else if b, ok := c.board.(flusherErr); ok {
		if err := b.Flush(); err != nil {
			return err
		}
	}

	firmChannel := make(chan []byte, 1)
	// Don't call ReportFirmware as it is automatic, but we want to register a listener for it.
	c.mu.Lock()
	c.responseChannels[SysExReportFirmware] = []chan []byte{firmChannel}
	c.mu.Unlock()
	report := c.parseReportFirmware(firmChannel)

	go func() {
		err := c.responseWatcher()
		if err != nil {
			panic(err)
		}
	}()

	fmt.Println("Firmware Info:", <-report)

	return nil
}

func (c *Client) write(payload []byte, withinMutex func()) error {
	// Cannot allow multiple writes at the same time.
	c.mu.Lock()
	defer c.mu.Unlock()

	//fmt.Println(SprintHexArray(payload))

	// Write to the board.
	_, err := c.board.Write(payload)
	if err != nil {
		return err
	}

	if withinMutex != nil {
		withinMutex()
	}

	return nil
}

func (c *Client) responseWatcher() (err error) {
	defer func() {
		if errors.Is(err, io.EOF) {
			err = ErrDeviceDisconnected
		}
	}()

	reader := bufio.NewReader(c.board)
	for {
		var data []byte
		b0, err := reader.ReadByte()
		if err != nil {
			return err
		}

		mt := MessageType(b0)
		switch {
		case mt == ProtocolVersion:
			var version [2]byte
			_, err := reader.Read(version[:])
			if err != nil {
				return err
			}

			fmt.Printf("Protocol Version: 0x%0.2X 0x%0.2X\n", version[0], version[1])
		case AnalogIOMessage <= mt && mt <= (AnalogIOMessage+0xF):
			v1, err := reader.ReadByte()
			if err != nil {
				return err
			}
			v2, err := reader.ReadByte()
			if err != nil {
				return err
			}

			c.analogIOMessageChannels[b0&0xF] <- TwoByteToWord(v1, v2)
		case DigitalIOMessage <= mt && mt <= (DigitalIOMessage+0xF):
			v1, err := reader.ReadByte()
			if err != nil {
				return err
			}
			v2, err := reader.ReadByte()
			if err != nil {
				return err
			}

			values := TwoByteToByte(v1, v2)

			port := b0 & 0xF
			pinMin := port * 8
			pinMax := (port+1)*8 - 1
			for p := pinMin; p <= pinMax; p++ {
				if ch, ok := c.digitalIOMessageChannels[p]; ok {
					lvl := gpio.Low
					if values>>p%8 > 0 {
						lvl = gpio.High
					}
					ch <- lvl
				}
			}
		case mt == StartSysEx:
			data, err = reader.ReadBytes(byte(EndSysEx))
			if err != nil {
				return err
			}

			if len(data) == 0 {
				return ErrNoDataRead
			}

			cmd := SysExCmd(data[0])
			data = data[1 : len(data)-1]

			switch {
			case cmd == SysExSerialDataV1:
				fallthrough
			case cmd == SysExSerialDataV2:
				return fmt.Errorf("%w: %s", ErrUnsupportedFeature, cmd)
			case cmd == SysExOneWireData:
				p := data[1]

				if l, ok := c.onewireListeners[p]; ok {
					l <- data
				} else {
					return fmt.Errorf("%w: onewire cmd:0x%02X pin 0x%02X", ErrUnhandledMessage, data[0], p)
				}
			case cmd == SysExI2CReply:
				address := TwoByteToByte(data[0], data[1])
				register := TwoByteToByte(data[2], data[3])
				ch, ok := c.i2cListeners[address]
				if !ok {
					return fmt.Errorf("%w: 0x%02X", ErrNoI2CListenerForAddress, address)
				}

				ch <- I2CPacket{
					Register: register,
					Data:     TwoByteRepresentationToByteSlice(data[4:]),
				}
			case c.sysExListenerChannels[cmd] != nil:
				c.sysExListenerChannels[cmd] <- data
			case len(c.responseChannels[cmd]) != 0:
				c.mu.Lock()
				resp := c.responseChannels[cmd][0]
				c.responseChannels[cmd] = c.responseChannels[cmd]
				c.mu.Unlock()

				resp <- data
				close(resp)
			case cmd == SysExStringData:
				fmt.Printf("device: [%s]\n", TwoByteString(data))
			default:
				str := ""
				if cmd == SysExStringData {
					str = TwoByteString(data)
				} else {
					for _, b := range data {
						str += fmt.Sprintf("%d", b)
					}
				}

				return fmt.Errorf("%w: 0x%0.2X: %s", ErrUnexpectedSysExMessageTypeReceived, byte(cmd), str)
			}
		default:
			return fmt.Errorf("%w: 0x%0.2X", ErrInvalidMessageTypeStart, b0)
		}
	}
}

func (c *Client) SendReset() error {
	return c.write([]byte{byte(SystemReset)}, nil)
}

func (c *Client) AnalogPinToDigitalPin(p uint8) (uint8, error) {
	if int(p) > len(c.amr.AnalogPinToDigital) {
		return 0, ErrInvalidAnalogPin
	}

	return c.amr.AnalogPinToDigital[p], nil
}

func (c *Client) SetPinMode(p uint8, mode pin.Func) error {
	return c.write([]uint8{uint8(SetPinMode), p, pinFuncToModeMap[mode]}, nil)
}

func (c *Client) SetDigitalPinValue(p uint8, value gpio.Level) error {
	v := byte(0)
	if value {
		v = 1
	}
	return c.write([]uint8{uint8(SetDigitalPinValue), p, v}, nil)
}

func (c *Client) SendSysEx(cmd SysExCmd, payload ...byte) (chan []byte, error) {
	// Create a response channel.
	var data chan []byte

	err := c.write(append([]byte{byte(StartSysEx), byte(cmd)}, append(payload, byte(EndSysEx))...), func() {
		// This assumes that SysEx commands of the same type are responded to in order.
		if resp, ok := commandResponseMap[cmd]; ok {
			data = make(chan []byte, 1)
			c.responseChannels[resp] = append(c.responseChannels[resp], data)
		}
	})
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *Client) CapabilityQuery() (chan CapabilityResponse, error) {
	future, err := c.SendSysEx(SysExCapabilityQuery)
	if err != nil {
		return nil, err
	}

	resp := c.parseCapabilityCommand(future)

	return resp, nil
}

func (c *Client) parseCapabilityCommand(future chan []byte) chan CapabilityResponse {
	resp := make(chan CapabilityResponse, 1)

	go func() {
		data := <-future
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

		c.cr = response
		resp <- response
		close(resp)
	}()

	return resp
}

func (c *Client) SendAnalogMappingQuery() (chan AnalogMappingResponse, error) {
	future, err := c.SendSysEx(SysExAnalogMappingQuery)
	if err != nil {
		return nil, err
	}

	resp := c.parseAnalogMappingQuery(future)

	return resp, nil
}

func (c *Client) parseAnalogMappingQuery(future chan []byte) chan AnalogMappingResponse {
	resp := make(chan AnalogMappingResponse, 1)

	go func() {
		data := <-future
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

		c.amr = response
		resp <- response
		close(resp)
	}()

	return resp
}

func (c *Client) ExtendedReportAnalogPin(p uint8, value int) error {
	if value > 0xFFFFFFFFFFFFFF {
		return fmt.Errorf("%w: 0x0 - 0xFFFFFFFFFFFFFF", ErrValueOutOfRange)
	}

	_, err := c.SendSysEx(SysExExtendedAnalog, p, uint8(value), uint8(value>>7), uint8(value>>14))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) PinStateQuery(p uint8) (chan PinStateResponse, error) {
	future, err := c.SendSysEx(SysExPinStateQuery, p)
	if err != nil {
		return nil, err
	}

	resp := c.parsePinStateQuery(future)

	return resp, nil
}

func (c *Client) parsePinStateQuery(future chan []byte) chan PinStateResponse {
	resp := make(chan PinStateResponse, 1)

	go func() {
		data := <-future
		var ps = PinStateResponse{
			Pin:   data[0],
			Mode:  pinModeToFuncMap[data[1]],
			State: 0,
		}

		for i, b := range data[2:] {
			ps.State |= int(b << (i * 7))
		}

		resp <- ps
		close(resp)
	}()

	return resp
}

func (c *Client) ReportFirmware() (chan FirmwareReport, error) {
	future, err := c.SendSysEx(SysExReportFirmware)
	if err != nil {
		return nil, err
	}

	resp := c.parseReportFirmware(future)

	return resp, nil
}

func (c *Client) parseReportFirmware(future chan []byte) chan FirmwareReport {
	resp := make(chan FirmwareReport, 1)

	go func() {
		data := <-future
		var rc = FirmwareReport{
			Major: data[0],
			Minor: data[1],
			Name:  data[2:],
		}

		resp <- rc
		close(resp)
	}()

	return resp
}

func (c *Client) SetAnalogPinReporting(analogPin uint8, report bool) error {
	v := byte(0)
	if report {
		v = 1
	}

	return c.write([]byte{byte(ReportAnalogPin) | (analogPin & 0xF), v}, nil)
}

func (c *Client) SetDigitalPinReporting(p uint8, report bool) error {
	return c.SetDigitalPortReporting(p%8, report)
}

func (c *Client) SetDigitalPortReporting(port uint8, report bool) error {
	v := byte(0)
	if report {
		v = 1
	}

	return c.write([]byte{byte(ReportDigitalPort) | (port & 0xF), v}, nil)
}

func (c *Client) SetSamplingInterval(ms uint16) error {
	if ms > MaxUInt16 {
		return fmt.Errorf("%w: 0x0 - 0x%X", ErrValueOutOfRange, MaxUInt16)
	}
	return c.write([]byte{byte(SysExSamplingInterval), byte(ms), byte(ms >> 7)}, nil)
}

// This function only supports 7-bit I2C addresses
func (c *Client) WriteI2CData(address uint8, restart bool, data []uint8) error {
	if !c.i2cStarted {
		return ErrI2CNotEnabled
	}

	byte2 := byte(I2CModeWrite)
	if restart {
		byte2 &= I2CRestartTransmission
	}

	payload := append([]byte{address, byte2}, ByteSliceToTwoByteRepresentation(data)...)
	_, err := c.SendSysEx(SysExI2CRequest, payload...)
	return err
}

// This function only supports 7-bit I2C addresses
func (c *Client) ReadI2CData(address uint8, restart bool, length uint16) error {
	if !c.i2cStarted {
		return ErrI2CNotEnabled
	}

	if length > MaxUInt16 {
		return fmt.Errorf("%w: 0x0 - 0xFFFFFFFFFFFFFF", ErrValueOutOfRange)
	}

	byte2 := byte(I2CModeRead)
	if restart {
		byte2 &= I2CRestartTransmission
	}

	lLSB, lMSB := WordToTwoByte(length)

	_, err := c.SendSysEx(SysExI2CRequest, address, byte2, lLSB, lMSB)
	return err
}

// This function only supports 7-bit I2C addresses
func (c *Client) ReadI2CRegister(address uint8, restart bool, register uint8, length uint16) error {
	if !c.i2cStarted {
		return ErrI2CNotEnabled
	}

	if length > MaxUInt16 {
		return fmt.Errorf("%w: 0x0 - 0xFFFFFFFFFFFFFF", ErrValueOutOfRange)
	}

	byte2 := byte(I2CModeRead)
	if restart {
		byte2 &= I2CRestartTransmission
	}

	rLSB, rMSB := ByteToTwoByte(register)
	lLSB, lMSB := WordToTwoByte(length)

	_, err := c.SendSysEx(SysExI2CRequest, address, byte2, rLSB, rMSB, lLSB, lMSB)
	return err
}

func (c *Client) SendI2CConfig(delayMicroseconds uint8) error {
	micLSB, micMSB := ByteToTwoByte(delayMicroseconds)
	_, err := c.SendSysEx(SysExI2CConfig, micLSB, micMSB)
	if err != nil {
		return err
	}
	c.i2cStarted = true
	return nil
}

func (c *Client) releaseI2CAddressListener(addr uint8) {
	c.i2cMU.Lock()
	defer c.i2cMU.Unlock()

	delete(c.i2cListeners, addr)
}

// This function only supports 7-bit I2C addresses
func (c *Client) SetI2CAddressListener(addr uint8, ch chan I2CPacket) (release func(), err error) {
	c.i2cMU.Lock()
	defer c.i2cMU.Unlock()

	if c.i2cListeners[addr] != nil {
		return nil, ErrI2CAddressListenerNotReleased
	}

	c.i2cListeners[addr] = ch

	return func() { c.releaseI2CAddressListener(addr) }, nil
}

func (c *Client) releaseAnalogIOMessageListener(p uint8) {
	c.analogPinMU.Lock()
	defer c.analogPinMU.Unlock()

	delete(c.analogIOMessageChannels, p)
}

func (c *Client) SetAnalogIOMessageListener(p uint8, ch chan uint16) (release func(), err error) {
	c.analogPinMU.Lock()
	defer c.analogPinMU.Unlock()

	if c.analogIOMessageChannels[p] != nil {
		return nil, ErrPinListenerNotReleased
	}

	c.analogIOMessageChannels[p] = ch

	return func() { c.releaseAnalogIOMessageListener(p) }, nil
}

func (c *Client) releaseDigitalIOMessageListener(p uint8) {
	c.digitalPinMU.Lock()
	defer c.digitalPinMU.Unlock()

	delete(c.digitalIOMessageChannels, p)
}

func (c *Client) SetDigitalIOMessageListener(p uint8, ch chan gpio.Level) (release func(), err error) {
	c.digitalPinMU.Lock()
	defer c.digitalPinMU.Unlock()

	if c.digitalIOMessageChannels[p] != nil {
		return nil, ErrPinListenerNotReleased
	}

	c.digitalIOMessageChannels[p] = ch

	return func() { c.releaseDigitalIOMessageListener(p) }, nil
}

func (c *Client) releaseOneWireListener(p uint8) {
	c.onewireMU.Lock()
	defer c.onewireMU.Unlock()

	delete(c.onewireListeners, p)
}

func (c *Client) SetOneWireListener(p uint8, ch chan []byte) (release func(), err error) {
	c.onewireMU.Lock()
	defer c.onewireMU.Unlock()

	if c.onewireListeners[p] != nil {
		return nil, ErrPinListenerNotReleased
	}

	c.onewireListeners[p] = ch

	return func() { c.releaseOneWireListener(p) }, nil
}

func (c *Client) OpenOneWireBus(p uint8) (bus onewire.BusCloser, err error) {
	c.onewireMU.Lock()
	defer c.onewireMU.Unlock()

	if c.onewireListeners[p] != nil {
		return nil, ErrPinListenerNotReleased
	}

	// Need to run configure or firmata will not initialize.
	if _, err := c.SendSysEx(SysExOneWireData, byte(OneWireInstructionConfigure), p, 0x00); err != nil {
		return nil, err
	}

	return newOneWireBus(c, newPin(c, p), func() error {
		c.releaseOneWireListener(p)
		return nil
	}), nil
}

func (c *Client) SendAnalogIOMessage(p uint8, value uint16) error {
	if p > 0xF {
		return ErrAnalogIOMessagePinOutOfRange
	}

	lsb, msb := WordToTwoByte(value)

	return c.write([]byte{byte(AnalogIOMessage) | p, lsb, msb}, nil)
}

func (c *Client) GetPinName(p uint8) string {
	if v, ok := c.amr.DigitalPinToAnalog[p]; ok {
		return fmt.Sprintf("A%d", v)
	}
	return fmt.Sprintf("%d", p)
}

func (c *Client) GetPinFunctions(p uint8) []pin.Func {
	return c.cr.SupportedPinModes[int(p)]
}

func (c *Client) OpenI2CBus() (i2c.Bus, error) {
	return newI2CBus(c)
}
