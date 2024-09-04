package firmata

type (
	MessageType uint8
)

const (
	AnalogIOMessage    MessageType = 0xE0 // pin #	LSB(bits 0-6)	MSB(bits 7-13)
	DigitalIOMessage   MessageType = 0x90 // port	LSB(bits 0-6)	MSB(bits 7-13)
	ReportAnalogPin    MessageType = 0xC0 // pin #	disable/enable(0/1)	- n/a -
	ReportDigitalPort  MessageType = 0xD0 // port	disable/enable(0/1)	- n/a -
	StartSysEx         MessageType = 0xF0 //
	SetPinMode         MessageType = 0xF4 // pin # (0-127)	pin mode
	SetDigitalPinValue MessageType = 0xF5 // pin # (0-127)	pin value(0/1)
	EndSysEx           MessageType = 0xF7 //
	ProtocolVersion    MessageType = 0xF9 // major version	minor version
	SystemReset        MessageType = 0xFF //
)

var messageTypeToStringMap = map[MessageType]string{
	AnalogIOMessage:    "AnalogIOMessage",
	DigitalIOMessage:   "DigitalIOMessage",
	ReportAnalogPin:    "ReportAnalogPin",
	ReportDigitalPort:  "ReportDigitalPort",
	StartSysEx:         "StartSysEx",
	SetPinMode:         "SetPinMode",
	SetDigitalPinValue: "SetDigitalPinValue",
	EndSysEx:           "EndSysEx",
	ProtocolVersion:    "ProtocolVersion",
	SystemReset:        "SystemReset",
}

func (m MessageType) String() string {
	switch {
	case AnalogIOMessage <= m && m <= (AnalogIOMessage+0xF):
		return messageTypeToStringMap[AnalogIOMessage]
	case DigitalIOMessage <= m && m <= (DigitalIOMessage+0xF):
		return messageTypeToStringMap[DigitalIOMessage]
	case ReportAnalogPin <= m && m <= (ReportAnalogPin+0xF):
		return messageTypeToStringMap[ReportAnalogPin]
	case ReportDigitalPort <= m && m <= (ReportDigitalPort+0xF):
		return messageTypeToStringMap[ReportDigitalPort]
	}

	if v, ok := messageTypeToStringMap[m]; ok {
		return v
	}

	return "Unknown"
}
