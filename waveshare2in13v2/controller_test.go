// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type record struct {
	cmd  byte
	data []byte
}

type fakeController []record

func (r *fakeController) sendCommand(cmd byte) {
	*r = append(*r, record{
		cmd: cmd,
	})
}

func (r *fakeController) sendData(data []byte) {
	cur := &(*r)[len(*r)-1]
	cur.data = append(cur.data, data...)
}

func (*fakeController) waitUntilIdle() {
}

func TestInitDisplay(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts Opts
		want []record
	}{
		{
			name: "epd2in13v2",
			opts: EPD2in13v2,
			want: []record{
				{cmd: swReset},
				{cmd: setAnalogBlockControl, data: []byte{0x54}},
				{cmd: setDigitalBlockControl, data: []byte{0x3b}},
				{
					cmd:  driverOutputControl,
					data: []byte{250 - 1, 0, 0},
				},
				{cmd: gateDrivingVoltageControl, data: []byte{gateDrivingVoltage19V}},
				{
					cmd: sourceDrivingVoltageControl,
					data: []byte{
						sourceDrivingVoltageVSH1_15V,
						sourceDrivingVoltageVSH2_5V,
						sourceDrivingVoltageVSL_neg15V,
					},
				},
				{cmd: setDummyLinePeriod, data: []byte{0x30}},
				{cmd: setGateTime, data: []byte{0x0a}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			initDisplay(&got, &tc.opts)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("initDisplay() difference (-got +want):\n%s", diff)
			}
		})
	}
}

func TestConfigDisplayMode(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode PartialUpdate
		lut  LUT
		want []record
	}{
		{
			name: "full",
			mode: Full,
			lut:  bytes.Repeat([]byte{'F'}, 100),
			want: []record{
				{cmd: writeVcomRegister, data: []byte{0x55}},
				{cmd: borderWaveformControl, data: []byte{0x03}},
				{cmd: writeLutRegister, data: bytes.Repeat([]byte{'F'}, 70)},
			},
		},
		{
			name: "partial",
			mode: Partial,
			lut:  bytes.Repeat([]byte{'P'}, 70),
			want: []record{
				{cmd: writeVcomRegister, data: []byte{0x24}},
				{cmd: borderWaveformControl, data: []byte{0x01}},
				{cmd: writeLutRegister, data: bytes.Repeat([]byte{'P'}, 70)},
				{cmd: 0x37, data: []byte{0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00}},
				{cmd: displayUpdateControl2, data: []byte{0xc0}},
				{cmd: masterActivation},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			configDisplayMode(&got, tc.mode, tc.lut)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("configDisplayMode() difference (-got +want):\n%s", diff)
			}
		})
	}
}

func TestUpdateDisplay(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode PartialUpdate
		want []record
	}{
		{
			name: "full",
			mode: Full,
			want: []record{
				{cmd: displayUpdateControl1, data: []byte{0}},
				{cmd: displayUpdateControl2, data: []byte{0xc7}},
				{cmd: masterActivation},
			},
		},
		{
			name: "partial",
			mode: Partial,
			want: []record{
				{cmd: displayUpdateControl1, data: []byte{0x80}},
				{cmd: displayUpdateControl2, data: []byte{0xc7}},
				{cmd: masterActivation},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			updateDisplay(&got, tc.mode)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("updateDisplay() difference (-got +want):\n%s", diff)
			}
		})
	}
}
