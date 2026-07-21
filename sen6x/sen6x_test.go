// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"testing"
	"time"

	"periph.io/x/conn/v3/i2c/i2ctest"
)

func TestNew(t *testing.T) {
	dev := New(&i2ctest.Playback{}, SEN66)

	if dev == nil {
		t.Fatalf("dev is nil")
	}

	if dev.model != SEN66 {
		t.Fatalf("got model %v, want %v", dev.model, SEN66)
	}
}

func TestDevString(t *testing.T) {
	d := newTestDev(t, nil, SEN66)
	got := d.String()
	want := "SEN66"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// sen66MeasurementOps returns the i2ctest.IO entries for a single SEN66
// measurement cycle. If withGetDataReady is true then the returned ops will
// start with GetDataReady returning ready=true.
func sen66MeasurementOps(t *testing.T, withGetDataReady bool) []i2ctest.IO {
	t.Helper()

	// SEN66 measurement data.
	data := []byte{
		0x00, 0x04, 0x45, // PM1.0
		0x00, 0x05, 0x74, // PM2.5
		0x00, 0x06, 0x27, // PM4.0
		0x00, 0x06, 0x27, // PM10.0
		0x14, 0xf0, 0xee, // RH
		0x11, 0x6b, 0x4a, // Temp
		0x00, 0x64, 0xfe, // VOC
		0x00, 0x46, 0x1a, // NOx
		0x01, 0xf4, 0x33, // CO2
	}

	ops := []i2ctest.IO{}

	if withGetDataReady {
		ops = append(ops,
			// GetDataReady write.
			i2ctest.IO{Addr: i2cAddr, W: []byte{0x02, 0x02}},
			// GetDataReady read.
			i2ctest.IO{Addr: i2cAddr, R: []byte{0x00, 0x01, crc8(0x00, 0x01)}},
		)
	}

	ops = append(ops,
		// ReadMeasuredValues write.
		i2ctest.IO{Addr: i2cAddr, W: []byte{0x03, 0x00}},
		// ReadMeasuredValues read.
		i2ctest.IO{Addr: i2cAddr, R: data},
	)

	return ops
}

func TestSenseContinuous(t *testing.T) {
	// Set the device's sampling interval to a small value so that tests run
	// quickly. Set it back after the tests finish.
	originalSamplingInterval := deviceSamplingInterval
	defer func() {
		deviceSamplingInterval = originalSamplingInterval
	}()
	deviceSamplingInterval = 2 * time.Millisecond

	// We'll use intervals shorter and longer than the device's native interval.
	shortTestInterval := deviceSamplingInterval - time.Millisecond
	longTestInterval := deviceSamplingInterval + time.Millisecond

	cases := []senseContinuousTestCase{
		{
			name:  "success short interval",
			model: SEN66,
			// An interval shorter than the device's internal sampling interval
			// will cause GetDataReady to be polled.
			interval:                 shortTestInterval,
			expectedMeasurementCount: 3,
			ops: func() []i2ctest.IO {
				// StartContinuousMeasurement write.
				ops := []i2ctest.IO{
					{Addr: i2cAddr, W: []byte{0x00, 0x21}},
				}

				// Three measurements.
				for range 3 {
					ops = append(ops, sen66MeasurementOps(t, true)...)
				}

				// StopMeasurement from Halt.
				ops = append(ops,
					i2ctest.IO{Addr: i2cAddr, W: []byte{0x01, 0x04}},
				)

				return ops
			}(),
		},
		{
			name:  "success long interval",
			model: SEN66,
			// An interval longer than the device's internal sampling interval will
			// result in GetDataReady not being polled before reading measurements.
			interval:                 longTestInterval,
			expectedMeasurementCount: 3,
			ops: func() []i2ctest.IO {
				// StartContinuousMeasurement write.
				ops := []i2ctest.IO{
					{Addr: i2cAddr, W: []byte{0x00, 0x21}},
				}

				// Three measurements.
				for range 3 {
					ops = append(ops, sen66MeasurementOps(t, false)...)
				}

				// StopMeasurement from Halt.
				ops = append(ops,
					i2ctest.IO{Addr: i2cAddr, W: []byte{0x01, 0x04}},
				)

				return ops
			}(),
		},
		{
			name:                     "fails to start",
			model:                    SEN66,
			interval:                 shortTestInterval,
			expectedMeasurementCount: 0,
			ops: []i2ctest.IO{
				// StartContinuousMeasurement write has no matching op
				// so i2ctest will return an error, simulating a bus error.

				// StopMeasurement from Halt.
				{Addr: i2cAddr, W: []byte{0x01, 0x04}},
			},
			dontPanic: true,
		},
		{
			name:                     "waitOnDataReady fails",
			model:                    SEN66,
			interval:                 shortTestInterval,
			expectedMeasurementCount: 0,
			ops: []i2ctest.IO{
				// StartContinuousMeasurement write.
				{Addr: i2cAddr, W: []byte{0x00, 0x21}},
				// GetDataReady write.
				{Addr: i2cAddr, W: []byte{0x02, 0x02}},
				// No GetDataReady read. This simulates a bus error in waitOnDataReady.

				// StopMeasurement from Halt.
				{Addr: i2cAddr, W: []byte{0x01, 0x04}},
			},
			dontPanic: true,
		},

		{
			name:                     "doReadMeasuredValues fails",
			model:                    SEN66,
			interval:                 shortTestInterval,
			expectedMeasurementCount: 0,
			ops: []i2ctest.IO{
				// StartContinuousMeasurement write.
				{Addr: i2cAddr, W: []byte{0x00, 0x21}},

				// A partial waitOnDataReady cycle that omits the measurement itself.
				// GetDataReady write.
				{Addr: i2cAddr, W: []byte{0x02, 0x02}},
				// GetDataReady read.
				{Addr: i2cAddr, R: []byte{0x00, 0x01, crc8(0x00, 0x01)}},
				// ReadMeasuredValues write.
				{Addr: i2cAddr, W: []byte{0x03, 0x00}},
				// No ReadMeasuredValues read. This simulates a bus error in doReadMeasuredValues.

				// StopMeasurement from Halt.
				{Addr: i2cAddr, W: []byte{0x01, 0x04}},
			},
			dontPanic: true,
		},
	}

	runSenseContinuousTests(t, cases)
}

func TestDevStartContinuousMeasurement(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx:    []byte{0x00, 0x21},
		},
	}

	runWriteTests(t, cases, (*Dev).StartContinuousMeasurement)
}

func TestDevStopMeasurement(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx:    []byte{0x01, 0x04},
		},
	}

	runWriteTests(t, cases, (*Dev).StopMeasurement)
}

func TestDevGetDataReady(t *testing.T) {
	cmd := []byte{0x02, 0x02}

	cases := []writeAndReadTestCase[bool]{
		{
			name:  "false",
			model: SEN66,
			tx:    cmd,
			rx:    []byte{0x00, 0x00, 0x81},
			want:  false,
		},
		{
			name:  "true",
			model: SEN66,
			tx:    cmd,
			rx:    []byte{0x00, 0x01, 0xb0},
			want:  true,
		},
		{
			name:    "bad CRC",
			model:   SEN66,
			tx:      cmd,
			rx:      []byte{0x00, 0x01, 0xff},
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).GetDataReady)
}

func TestDevReadMeasuredValues(t *testing.T) {
	cases := []writeAndReadTestCase[*SensorValues]{
		{
			name:  "SEN62",
			model: SEN62,
			tx:    []byte{0x04, 0xA3},
			rx: []byte{
				0x00, 0x04, 0x45, // PM1.0
				0x00, 0x05, 0x74, // PM2.5
				0x00, 0x06, 0x27, // PM4.0
				0x00, 0x06, 0x27, // PM10.0
				0x14, 0xf0, 0xee, // RH
				0x11, 0x6b, 0x4a, // Temp
			},
			want: &SensorValues{
				PM1:  ptr(float32(0.4)),
				PM25: ptr(float32(0.5)),
				PM4:  ptr(float32(0.6)),
				PM10: ptr(float32(0.6)),
				RH:   ptr(float32(53.6)),
				Temp: ptr(float32(22.295)),
			},
		},
		{
			name:  "SEN63C",
			model: SEN63C,
			tx:    []byte{0x04, 0x71},
			rx: []byte{
				0x00, 0x04, 0x45, // PM1.0
				0x00, 0x05, 0x74, // PM2.5
				0x00, 0x06, 0x27, // PM4.0
				0x00, 0x06, 0x27, // PM10.0
				0x14, 0xf0, 0xee, // RH
				0x11, 0x6b, 0x4a, // Temp
				0x01, 0xf4, 0x33, // CO2
			},
			want: &SensorValues{
				PM1:  ptr(float32(0.4)),
				PM25: ptr(float32(0.5)),
				PM4:  ptr(float32(0.6)),
				PM10: ptr(float32(0.6)),
				RH:   ptr(float32(53.6)),
				Temp: ptr(float32(22.295)),
				CO2:  ptr(int16(500)),
			},
		},
		{
			name:  "SEN65",
			model: SEN65,
			tx:    []byte{0x04, 0x46},
			rx: []byte{
				0x00, 0x04, 0x45, // PM1.0
				0x00, 0x05, 0x74, // PM2.5
				0x00, 0x06, 0x27, // PM4.0
				0x00, 0x06, 0x27, // PM10.0
				0x14, 0xf0, 0xee, // RH
				0x11, 0x6b, 0x4a, // Temp
				0x00, 0x64, 0xfe, // VOC
				0x00, 0x46, 0x1a, // NOx
			},
			want: &SensorValues{
				PM1:  ptr(float32(0.4)),
				PM25: ptr(float32(0.5)),
				PM4:  ptr(float32(0.6)),
				PM10: ptr(float32(0.6)),
				RH:   ptr(float32(53.6)),
				Temp: ptr(float32(22.295)),
				VOC:  ptr(float32(10)),
				NOx:  ptr(float32(7)),
			},
		},
		{
			name:  "SEN66",
			model: SEN66,
			tx:    []byte{0x03, 0x00},
			rx: []byte{
				0x00, 0x04, 0x45, // PM1.0
				0x00, 0x05, 0x74, // PM2.5
				0x00, 0x06, 0x27, // PM4.0
				0x00, 0x06, 0x27, // PM10.0
				0x14, 0xf0, 0xee, // RH
				0x11, 0x6b, 0x4a, // Temp
				0x00, 0x64, 0xfe, // VOC
				0x00, 0x46, 0x1a, // NOx
				0x01, 0xf4, 0x33, // CO2
			},
			want: &SensorValues{
				PM1:  ptr(float32(0.4)),
				PM25: ptr(float32(0.5)),
				PM4:  ptr(float32(0.6)),
				PM10: ptr(float32(0.6)),
				RH:   ptr(float32(53.6)),
				Temp: ptr(float32(22.295)),
				VOC:  ptr(float32(10)),
				NOx:  ptr(float32(7)),
				CO2:  ptr(int16(500)),
			},
		},
		{
			name:  "SEN68",
			model: SEN68,
			tx:    []byte{0x04, 0x67},
			rx: []byte{
				0x00, 0x04, 0x45, // PM1.0
				0x00, 0x05, 0x74, // PM2.5
				0x00, 0x06, 0x27, // PM4.0
				0x00, 0x06, 0x27, // PM10.0
				0x14, 0xf0, 0xee, // RH
				0x11, 0x6b, 0x4a, // Temp
				0x00, 0x64, 0xfe, // VOC
				0x00, 0x46, 0x1a, // NOx
				0x01, 0xf4, 0x33, // HCHO
			},
			want: &SensorValues{
				PM1:  ptr(float32(0.4)),
				PM25: ptr(float32(0.5)),
				PM4:  ptr(float32(0.6)),
				PM10: ptr(float32(0.6)),
				RH:   ptr(float32(53.6)),
				Temp: ptr(float32(22.295)),
				VOC:  ptr(float32(10)),
				NOx:  ptr(float32(7)),
				HCHO: ptr(float32(50)),
			},
		},
		{
			name:  "SEN69C",
			model: SEN69C,
			tx:    []byte{0x04, 0xb5},
			rx: []byte{
				0x00, 0x04, 0x45, // PM1.0
				0x00, 0x05, 0x74, // PM2.5
				0x00, 0x06, 0x27, // PM4.0
				0x00, 0x06, 0x27, // PM10.0
				0x14, 0xf0, 0xee, // RH
				0x11, 0x6b, 0x4a, // Temp
				0x00, 0x64, 0xfe, // VOC
				0x00, 0x46, 0x1a, // NOx
				0x01, 0xf4, 0x33, // HCHO
				0x01, 0xf4, 0x33, // CO2
			},
			want: &SensorValues{
				PM1:  ptr(float32(0.4)),
				PM25: ptr(float32(0.5)),
				PM4:  ptr(float32(0.6)),
				PM10: ptr(float32(0.6)),
				RH:   ptr(float32(53.6)),
				Temp: ptr(float32(22.295)),
				VOC:  ptr(float32(10)),
				NOx:  ptr(float32(7)),
				HCHO: ptr(float32(50)),
				CO2:  ptr(int16(500)),
			},
		},
		{
			// This covers the uint16 CO2 encoding used by the SEN66.
			name:  "SEN66 all values unknown",
			model: SEN66,
			tx:    []byte{0x03, 0x00},
			rx: []byte{
				0xff, 0xff, 0xac, // PM1.0
				0xff, 0xff, 0xac, // PM2.5
				0xff, 0xff, 0xac, // PM4.0
				0xff, 0xff, 0xac, // PM10.0
				0x7f, 0xff, 0x8f, // RH
				0x7f, 0xff, 0x8f, // Temp
				0x7f, 0xff, 0x8f, // VOC
				0x7f, 0xff, 0x8f, // NOx
				0xff, 0xff, 0xac, // CO2 as uint
			},
			want: &SensorValues{},
		},
		{
			// This covers the unset/unknown value for all measurements in the
			// SEN6x family, but notably it uses the int16 CO2 encoding also used
			// by the SEN63C.
			name:  "SEN69C all values unknown",
			model: SEN69C,
			tx:    []byte{0x04, 0xb5},
			rx: []byte{
				0xff, 0xff, 0xac, // PM1.0
				0xff, 0xff, 0xac, // PM2.5
				0xff, 0xff, 0xac, // PM4.0
				0xff, 0xff, 0xac, // PM10.0
				0x7f, 0xff, 0x8f, // RH
				0x7f, 0xff, 0x8f, // Temp
				0x7f, 0xff, 0x8f, // VOC
				0x7f, 0xff, 0x8f, // NOx
				0xff, 0xff, 0xac, // HCHO
				0x7f, 0xff, 0x8f, // CO2 as int
			},
			want: &SensorValues{},
		},
		{
			name:  "bad crc",
			model: SEN62,
			tx:    []byte{0x04, 0xA3},
			rx: []byte{
				0x00, 0x04, 0x45, // PM1.0
				0x00, 0x05, 0x74, // PM2.5
				0x00, 0x06, 0x27, // PM4.0
				0x00, 0x06, 0xff, // PM10.0 with incorrect CRC (should be 0x27)
				0x14, 0xf0, 0xee, // RH
				0x11, 0x6b, 0x4a, // Temp
			},
			wantErr: true,
		},
		{
			name:    "unknown model",
			model:   Model(-1),
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).ReadMeasuredValues)
}

func TestDevReadMeasuredRawValues(t *testing.T) {
	cases := []writeAndReadTestCase[*RawSensorValues]{
		{
			name:  "SEN62",
			model: SEN62,
			tx:    []byte{0x04, 0x92},
			rx: []byte{
				0x14, 0x0e, 0x73, // RH
				0x11, 0xea, 0x01, // Temp
			},
			want: &RawSensorValues{
				RH:   ptr(float32(51.34)),
				Temp: ptr(float32(22.93)),
			},
		},
		{
			name:  "SEN63C",
			model: SEN63C,
			tx:    []byte{0x04, 0x92},
			rx: []byte{
				0x14, 0x0e, 0x73, // RH
				0x11, 0xea, 0x01, // Temp
			},
			want: &RawSensorValues{
				RH:   ptr(float32(51.34)),
				Temp: ptr(float32(22.93)),
			},
		},
		{
			name:  "SEN65",
			model: SEN65,
			tx:    []byte{0x04, 0x55},
			rx: []byte{
				0x14, 0x0e, 0x73, // RH
				0x11, 0xea, 0x01, // Temp
				0x72, 0xc9, 0xac, // VOC
				0x49, 0x45, 0x03, // NOx
			},
			want: &RawSensorValues{
				RH:   ptr(float32(51.34)),
				Temp: ptr(float32(22.93)),
				VOC:  ptr(uint16(29385)),
				NOx:  ptr(uint16(18757)),
			},
		},
		{
			name:  "SEN66",
			model: SEN66,
			tx:    []byte{0x04, 0x05},
			rx: []byte{
				0x14, 0x0e, 0x73, // RH
				0x11, 0xea, 0x01, // Temp
				0x72, 0xc9, 0xac, // VOC
				0x49, 0x45, 0x03, // NOx
				0x01, 0xc8, 0x8b, // CO2
			},
			want: &RawSensorValues{
				RH:   ptr(float32(51.34)),
				Temp: ptr(float32(22.93)),
				VOC:  ptr(uint16(29385)),
				NOx:  ptr(uint16(18757)),
				CO2:  ptr(uint16(456)),
			},
		},
		{
			name:  "SEN68",
			model: SEN68,
			tx:    []byte{0x04, 0x55},
			rx: []byte{
				0x14, 0x0e, 0x73, // RH
				0x11, 0xea, 0x01, // Temp
				0x72, 0xc9, 0xac, // VOC
				0x49, 0x45, 0x03, // NOx
			},
			want: &RawSensorValues{
				RH:   ptr(float32(51.34)),
				Temp: ptr(float32(22.93)),
				VOC:  ptr(uint16(29385)),
				NOx:  ptr(uint16(18757)),
			},
		},
		{
			name:  "SEN69C",
			model: SEN69C,
			tx:    []byte{0x04, 0x55},
			rx: []byte{
				0x14, 0x0e, 0x73, // RH
				0x11, 0xea, 0x01, // Temp
				0x72, 0xc9, 0xac, // VOC
				0x49, 0x45, 0x03, // NOx
			},
			want: &RawSensorValues{
				RH:   ptr(float32(51.34)),
				Temp: ptr(float32(22.93)),
				VOC:  ptr(uint16(29385)),
				NOx:  ptr(uint16(18757)),
			},
		},
		{
			// SEN66 covers all raw measurements.
			name:  "SEN66 all values unknown",
			model: SEN66,
			tx:    []byte{0x04, 0x05},
			rx: []byte{
				0x7f, 0xff, 0x8f, // RH
				0x7f, 0xff, 0x8f, // Temp
				0xff, 0xff, 0xac, // VOC
				0xff, 0xff, 0xac, // NOx
				0xff, 0xff, 0xac, // CO2
			},
			want: &RawSensorValues{},
		},
		{
			name:  "SEN66",
			model: SEN66,
			tx:    []byte{0x04, 0x05},
			rx: []byte{
				0x14, 0x0e, 0x73, // RH
				0x11, 0xea, 0x01, // Temp
				0x72, 0xc9, 0xac, // VOC
				0x49, 0x45, 0xff, // NOx with incorrect CRC (should be 0x03)
				0x01, 0xc8, 0x8b, // CO2
			},
			wantErr: true,
		},
		{
			name:    "unknown model",
			model:   Model(-1),
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).ReadMeasuredRawValues)
}

func TestDevGetProductName(t *testing.T) {
	cmd := []byte{0xd0, 0x14}

	cases := []writeAndReadTestCase[string]{
		{
			name:  "SEN66",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x53, 0x45, 0x83, // "SE"
				0x4e, 0x36, 0x06, // "N6"
				0x36, 0x00, 0x69, // "6\0"
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
			},
			want: "SEN66",
		},
		{
			name:  "bad CRC",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x53, 0x45, 0x83, // "SE"
				0x4e, 0x36, 0xff, // "N6" with incorrect CRC (should be 0x06)
				0x36, 0x00, 0x69, // "6\0"
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
			},
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).GetProductName)
}

func TestDevGetSerialNumber(t *testing.T) {
	cmd := []byte{0xd0, 0x33}

	cases := []writeAndReadTestCase[string]{
		{
			name:  "serial number from device",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x32, 0x34, 0xeb, // "24"
				0x43, 0x34, 0x24, // "C4"
				0x45, 0x45, 0xb7, // "EE"
				0x31, 0x37, 0x95, // "17"
				0x46, 0x41, 0x5e, // "FA"
				0x43, 0x37, 0x77, // "C7"
				0x43, 0x35, 0x15, // "C5"
				0x43, 0x42, 0x7a, // "CB"
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
			},
			want: "24C4EE17FAC7C5CB",
		},
		{
			name:  "bad CRC",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x32, 0x34, 0xeb, // "24"
				0x43, 0x34, 0x24, // "C4"
				0x45, 0x45, 0xb7, // "EE"
				0x31, 0x37, 0xff, // "17" with incorrect CRC (should be 0x95)
				0x46, 0x41, 0x5e, // "FA"
				0x43, 0x37, 0x77, // "C7"
				0x43, 0x35, 0x15, // "C5"
				0x43, 0x42, 0x7a, // "CB"
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
				0x00, 0x00, 0x81,
			},
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).GetSerialNumber)
}

func TestDevGetVersion(t *testing.T) {
	cases := []struct {
		name      string
		response  []byte
		wantMajor uint8
		wantMinor uint8
		wantErr   bool
	}{
		{
			name:      "version 4.0",
			response:  []byte{0x04, 0x00, 0x2},
			wantMajor: 4,
			wantMinor: 0,
		},
		{
			name:      "version 1.2",
			response:  []byte{0x01, 0x02, crc8(0x01, 0x02)},
			wantMajor: 1,
			wantMinor: 2,
		},
		{
			name:     "bad CRC",
			response: []byte{0x04, 0x00, 0xff},
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bus := i2ctest.Playback{
				Ops: []i2ctest.IO{
					{Addr: i2cAddr, W: []byte{0xd1, 0x00}},
					{Addr: i2cAddr, R: tc.response},
				},
			}
			defer func() {
				if err := bus.Close(); err != nil {
					t.Error(err)
				}
			}()
			d := newTestDev(t, &bus, SEN66)

			gotMajor, gotMinor, err := d.GetVersion()

			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tc.wantErr {
				if gotMajor != tc.wantMajor {
					t.Errorf("got major version %d, want %d", gotMajor, tc.wantMajor)
				}

				if gotMinor != tc.wantMinor {
					t.Errorf("got minor version %d, want %d", gotMinor, tc.wantMinor)
				}
			}
		})
	}
}

func TestDevDeviceReset(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx:    []byte{0xd3, 0x04},
		},
	}

	runWriteTests(t, cases, (*Dev).DeviceReset)
}

func TestDevStartFanCleaning(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx:    []byte{0x56, 0x07},
		},
	}

	runWriteTests(t, cases, (*Dev).StartFanCleaning)
}
