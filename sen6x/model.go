// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

//go:generate -command stringer go run golang.org/x/tools/cmd/stringer@latest
//go:generate stringer -type=Model
package sen6x

// Model represents the various sensor models in the SEN6x family.
type Model int

const (
	SEN62 Model = iota
	SEN63C
	SEN65
	SEN66
	SEN68
	SEN69C
)

// hasCO2 returns true if the model has a CO2 sensor.
//
// Some CO2 capabilities behave differently on SEN63C and SEN69C than they do on
// SEN66. This package abstracts away the differences where possible and documents
// them otherwise.
//
// Based on the description in [datasheet] section 4.8.32, Perform CO2 Sensor Factory
// Reset, it would appear that SEN63C, SEN66, and SEN69C all use the [STCC4] CO2
// sensor. It seems, then, that it is the SEN6x firmware that accounts for
// the CO2 differences.
//
// However, section 1.5 of the [datasheet], CO2 Specifications, shows different
// specs for SEN63C and SEN69C vs. SEN66. One would expect the same specs if the
// same sensor were used. So it's not totally clear how the internals differ.
//
// Regardless, you'll notice that in some locations in the code we check sensor
// model to differentiate CO2-related actions.
//
// [datasheet]: https://sensirion.com/media/documents/FAFC548D/693FBB15/PS_DS_SEN6x.pdf
// [STCC4]: https://sensirion.com/media/documents/6AED4B15/69295E41/CD_DS_STCC4_D1.pdf
func (m Model) hasCO2() bool {
	switch m {
	case SEN63C, SEN66, SEN69C:
		return true
	}

	return false
}

// hasVOCNOx returns true if the model has VOC and NOx sensors.
func (m Model) hasVOCNOx() bool {
	switch m {
	case SEN65, SEN66, SEN68, SEN69C:
		return true
	}

	return false
}

// hasHCHO returns true if the model has a formaldehyde sensor.
func (m Model) hasHCHO() bool {
	switch m {
	case SEN68, SEN69C:
		return true
	}

	return false
}
