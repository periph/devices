// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

import (
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

func TestInitDisplayFull(t *testing.T) {
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
				{cmd: borderWaveformControl, data: []byte{0x03}},
				{cmd: writeVcomRegister, data: []byte{0x55}},
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
				{cmd: writeLutRegister, data: EPD2in13v2.FullUpdate},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			initDisplayFull(&got, &tc.opts)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("initDisplayFull() difference (-got +want):\n%s", diff)
			}
		})
	}
}

func TestInitDisplayPartial(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts Opts
		want []record
	}{
		{
			name: "epd2in13v2",
			opts: EPD2in13v2,
			want: []record{
				{cmd: writeVcomRegister, data: []byte{0x26}},
				{cmd: writeLutRegister, data: EPD2in13v2.PartialUpdate},
				{cmd: 0x37, data: []byte{0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00}},
				{cmd: displayUpdateControl2, data: []byte{0xc0}},
				{cmd: masterActivation},
				{cmd: borderWaveformControl, data: []byte{0x01}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			initDisplayPartial(&got, &tc.opts)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("initDisplayPartial() difference (-got +want):\n%s", diff)
			}
		})
	}
}
