package firmata

type (
	SysExCmd         uint8
	SysExExtendedCmd uint16
)

// Base Features
const (
	SysExExtendedId            SysExCmd = 0x00 // A value of 0x00 indicates the next 2 bytes define the extended ID
	SysExAnalogMappingQuery    SysExCmd = 0x69 // ask for mapping of analog pin names to pin numbers
	SysExAnalogMappingResponse SysExCmd = 0x6A // reply with mapping info
	SysExCapabilityQuery       SysExCmd = 0x6B // ask for supported modes and resolution of all pins
	SysExCapabilityResponse    SysExCmd = 0x6C // reply with supported modes and resolution
	SysExPinStateQuery         SysExCmd = 0x6D // ask for a pin's current mode and state (different from value)
	SysExPinStateResponse      SysExCmd = 0x6E // reply with a pin's current mode and state (different from value)
	SysExExtendedAnalog        SysExCmd = 0x6F // analog write (PWM, Servo, etc.) to any pin
	SysExStringData            SysExCmd = 0x71 // a string message with 14-bits per char
	SysExReportFirmware        SysExCmd = 0x79 // report name and version of the firmware
	SysExSamplingInterval      SysExCmd = 0x7A // the interval at which analog input is sampled (default = 19ms)
	SysExNonRealtime           SysExCmd = 0x7E // MIDI Reserved for non-realtime messages
	SysExRealtime              SysExCmd = 0x7F // MIDI Reserved for realtime messages
)

// User Defined Feature Codes
// Assign these to whatever feature constant you define
const (
	UserFeature1 SysExCmd = 0x01
	UserFeature2 SysExCmd = 0x02
	UserFeature3 SysExCmd = 0x03
	UserFeature4 SysExCmd = 0x04
	UserFeature5 SysExCmd = 0x05
	UserFeature6 SysExCmd = 0x06
	UserFeature7 SysExCmd = 0x07
	UserFeature8 SysExCmd = 0x08
	UserFeature9 SysExCmd = 0x09
	UserFeatureA SysExCmd = 0x0A
	UserFeatureB SysExCmd = 0x0B
	UserFeatureC SysExCmd = 0x0C
	UserFeatureD SysExCmd = 0x0D
	UserFeatureE SysExCmd = 0x0E
	UserFeatureF SysExCmd = 0x0F
)

// Optionally Included Features
const (
	SysExRCOutputData     SysExCmd = 0x5C // https://github.com/firmata/protocol/blob/master/proposals/rcswitch-proposal.md
	SysExRCInputData      SysExCmd = 0x5D // https://github.com/firmata/protocol/blob/master/proposals/rcswitch-proposal.md
	SysExDeviceQuery      SysExCmd = 0x5E // https://github.com/finson-release/Luni/blob/master/extras/v0.9/v0.8-device-driver-C-firmata-messages.md
	SysExDeviceResponse   SysExCmd = 0x5F // https://github.com/finson-release/Luni/blob/master/extras/v0.9/v0.8-device-driver-C-firmata-messages.md
	SysExSerialDataV1     SysExCmd = 0x60 // https://github.com/firmata/protocol/blob/master/serial-1.0.md
	SysExEncoderData      SysExCmd = 0x61 // https://github.com/firmata/protocol/blob/master/encoder.md
	SysExAccelStepperData SysExCmd = 0x62 // https://github.com/firmata/protocol/blob/master/accelStepperFirmata.md
	SysExSerialDataV2     SysExCmd = 0x67 // https://github.com/firmata/protocol/blob/master/proposals/serial-2.0-proposal.md
	SysExSPIData          SysExCmd = 0x68 // https://github.com/firmata/protocol/blob/master/spi.md
	SysExServoConfig      SysExCmd = 0x70 // https://github.com/firmata/protocol/blob/master/servos.md
	SysExStepperData      SysExCmd = 0x72 // https://github.com/firmata/protocol/blob/master/stepper-legacy.md
	SysExOneWireData      SysExCmd = 0x73 // https://github.com/firmata/protocol/blob/master/onewire.md
	SysExDHTSensorData    SysExCmd = 0x74 // https://github.com/firmata/protocol/blob/master/dhtsensor.md
	SysExShiftData        SysExCmd = 0x75 // https://github.com/firmata/protocol/blob/master/proposals/shift-proposal.md
	SysExI2CRequest       SysExCmd = 0x76 // https://github.com/firmata/protocol/blob/master/i2c.md
	SysExI2CReply         SysExCmd = 0x77 // https://github.com/firmata/protocol/blob/master/i2c.md
	SysExI2CConfig        SysExCmd = 0x78 // https://github.com/firmata/protocol/blob/master/i2c.md
	SysExSchedulerData    SysExCmd = 0x7B // https://github.com/firmata/protocol/blob/master/scheduler.md
	SysExFrequencyCommand SysExCmd = 0x7D // https://github.com/firmata/protocol/blob/master/frequency.md
)

var sysExCmdToStringMap = map[SysExCmd]string{
	SysExExtendedId:            "ExtendedId",
	SysExAnalogMappingQuery:    "AnalogMappingQuery",
	SysExAnalogMappingResponse: "AnalogMappingResponse",
	SysExCapabilityQuery:       "CapabilityQuery",
	SysExCapabilityResponse:    "CapabilityResponse",
	SysExPinStateQuery:         "PinStateQuery",
	SysExPinStateResponse:      "PinStateResponse",
	SysExExtendedAnalog:        "ExtendedAnalog",
	SysExStringData:            "StringData",
	SysExReportFirmware:        "ReportFirmware",
	SysExSamplingInterval:      "SamplingInterval",
	SysExNonRealtime:           "NonRealtime",
	SysExRealtime:              "Realtime",
	SysExRCOutputData:          "RCOutputData",
	SysExRCInputData:           "RCInputData",
	SysExDeviceQuery:           "DeviceQuery",
	SysExDeviceResponse:        "DeviceResponse",
	SysExSerialDataV1:          "SerialDataV1",
	SysExEncoderData:           "EncoderData",
	SysExAccelStepperData:      "AccelStepperData",
	SysExSerialDataV2:          "SerialDataV2",
	SysExSPIData:               "SPIData",
	SysExServoConfig:           "ServoConfig",
	SysExStepperData:           "StepperData",
	SysExOneWireData:           "OneWireData",
	SysExDHTSensorData:         "DHTSensorData",
	SysExShiftData:             "ShiftData",
	SysExI2CRequest:            "I2CRequest",
	SysExI2CReply:              "I2CReply",
	SysExI2CConfig:             "I2CConfig",
	SysExSchedulerData:         "SchedulerData",
	SysExFrequencyCommand:      "FrequencyCommand",
}

func (s SysExCmd) String() string {
	if v, ok := sysExCmdToStringMap[s]; ok {
		return v
	}

	return "Unknown"
}
