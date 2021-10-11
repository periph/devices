package firmata

import (
	"fmt"
)

type FirmwareReport struct {
	Major byte
	Minor byte
	Name  []byte
}

func ParseFirmwareReport(data []byte) FirmwareReport {
	return FirmwareReport{
		Major: data[0],
		Minor: data[1],
		Name:  data[2:],
	}
}

func (r FirmwareReport) String() string {
	return fmt.Sprintf("%s [%d.%d]", TwoByteString(r.Name), r.Major, r.Minor)
}
