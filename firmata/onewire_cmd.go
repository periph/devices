package firmata

type OneWireInstruction uint8

const (
	OneWireInstructionSearch             OneWireInstruction = 0x40
	OneWireInstructionConfigure          OneWireInstruction = 0x41
	OneWireInstructionSearchReply        OneWireInstruction = 0x42
	OneWireInstructionReadReply          OneWireInstruction = 0x43
	OneWireInstructionSearchAlarmed      OneWireInstruction = 0x44
	OneWireInstructionSearchAlarmedReply OneWireInstruction = 0x45
)

type OneWireConnectorWrapper struct {
	client ClientI
}

type OneWireCommand uint8

const (
	OneWireCommandReset  OneWireCommand = 0b00000001
	OneWireCommandSkip   OneWireCommand = 0b00000010
	OneWireCommandSelect OneWireCommand = 0b00000100
	OneWireCommandRead   OneWireCommand = 0b00001000
	OneWireCommandDelay  OneWireCommand = 0b00010000
	OneWireCommandWrite  OneWireCommand = 0b00100000
)
