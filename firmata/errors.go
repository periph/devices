package firmata

import (
	"errors"
)

var (
	ErrDeviceDisconnected                 = errors.New("device disconnected")
	ErrUnsupportedFeature                 = errors.New("unsupported feature")
	ErrUnhandledMessage                   = errors.New("message did not have an active listener")
	ErrInvalidMessageTypeStart            = errors.New("invalid message type start")
	ErrNoDataRead                         = errors.New("no data read")
	ErrUnexpectedSysExMessageTypeReceived = errors.New("unexpected sysex message type")
	ErrAlreadyStarted                     = errors.New("client already started")
	ErrValueOutOfRange                    = errors.New("value is out of range")
	ErrNoI2CListenerForAddress            = errors.New("no i2c listener registered for address")
	ErrInvalidFirmataI2CBus               = errors.New("firmata does not support multiple i2c buses")
	ErrInvalidAnalogPin                   = errors.New("analog pin is outside of range")
	ErrI2CNotEnabled                      = errors.New("i2c must started to use")
	ErrInvalidOneWirePin                  = errors.New("onewire pin number cannot exceed 0x7F")
	ErrAnalogIOMessagePinOutOfRange       = errors.New("analog io message pin number cannot exceed 0xF")
	ErrDigitalIOMessagePinOutOfRange      = errors.New("digital io message port number cannot exceed 0xF")
	ErrPinListenerNotReleased             = errors.New("pin listener is already set for pin")
)
