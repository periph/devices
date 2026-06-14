// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import "encoding/binary"

// PMNumberConcentrations contains particulate matter measurements in particles/cm³ ("number concentration").
type PMNumberConcentrations struct {
	PM05, PM1, PM25, PM4, PM10 *uint16
}

// ReadNumberConcentrationValues reads the current particulate matter concentration
// measurements in particles/cm³ ("number concentration") rather than the more
// conventional μg/m³.
//
// [Dev.GetDataReady] may be used to check if new data is available since the
// last read operation. If no new data is available, the previous values will
// be returned. If no data is available at all (e.g. measurement not running
// for at least one second), all values will be nil.
func (d *Dev) ReadNumberConcentrationValues() (*PMNumberConcentrations, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdReadNumberConcentrationValues, nil)
	if err != nil {
		return nil, err
	}

	nc := &PMNumberConcentrations{}

	if rawPM05 := binary.BigEndian.Uint16(data[0:2]); rawPM05 != 0xffff {
		nc.PM05 = ptr(rawPM05)
	}

	if rawPM1 := binary.BigEndian.Uint16(data[2:4]); rawPM1 != 0xffff {
		nc.PM1 = ptr(rawPM1)
	}

	if rawPM25 := binary.BigEndian.Uint16(data[4:6]); rawPM25 != 0xffff {
		nc.PM25 = ptr(rawPM25)
	}

	if rawPM4 := binary.BigEndian.Uint16(data[6:8]); rawPM4 != 0xffff {
		nc.PM4 = ptr(rawPM4)
	}

	if rawPM10 := binary.BigEndian.Uint16(data[8:10]); rawPM10 != 0xffff {
		nc.PM10 = ptr(rawPM10)
	}

	return nc, nil
}
