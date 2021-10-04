package firmata

import (
	"fmt"
)

type FirmwareReport struct {
	Major byte
	Minor byte
	Name  []byte
}

func (r FirmwareReport) String() string {
	return fmt.Sprintf("%s [%d.%d]", TwoByteString(r.Name), r.Major, r.Minor)
}
