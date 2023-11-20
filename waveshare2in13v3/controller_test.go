// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v3

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
			name: "epd2in13v3",
			opts: EPD2in13v3,
			want: []record{
				{cmd: swReset},
				{
					cmd:  driverOutputControl,
					data: []byte{250 - 1, 0, 0},
				},
				{cmd: dataEntryModeSetting, data: []uint8{0x03}},
				{cmd: setRAMXAddressStartEndPosition, data: []uint8{0x00, 0x0f}},
				{cmd: setRAMYAddressStartEndPosition, data: []uint8{0x00, 0x00, 0xf9, 0x00}},
				{cmd: setRAMXAddressCounter, data: []uint8{0x00}},
				{cmd: setRAMYAddressCounter, data: []uint8{0x00, 0x00}},
				{cmd: borderWaveformControl, data: []uint8{0x05}},
				{cmd: displayUpdateControl1, data: []uint8{0x00, 0x80}},
				{cmd: tempSensorSelect, data: []uint8{0x80}},
				{cmd: writeLutRegister, data: EPD2in13v3.FullUpdate[:153]},
				{cmd: endOptionEOPT, data: []uint8{EPD2in13v3.FullUpdate[153]}},
				{cmd: gateDrivingVoltageControl, data: []uint8{EPD2in13v3.FullUpdate[154]}},
				{cmd: sourceDrivingVoltageControl, data: EPD2in13v3.FullUpdate[155:157]},
				{cmd: writeVcomRegister, data: []uint8{EPD2in13v3.FullUpdate[158]}},
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
				{cmd: 0x37, data: []byte{0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x00}},
				{cmd: displayUpdateControl2, data: []byte{0xc0}},
				{cmd: masterActivation},
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

func TestClear(t *testing.T) {
	var buff []byte
	const linewidth = int(122/8) + 1
	for j := 0; j < 250; j++ {
		for i := 0; i < linewidth; i++ {
			buff = append(buff, 0x00)
		}
	}
	for _, tc := range []struct {
		name  string
		opts  Opts
		color byte
		want  []record
	}{
		{
			name: "clear",
			opts: EPD2in13v3,
			want: []record{
				{cmd: writeRAMBW, data: buff},
				{cmd: displayUpdateControl2, data: []byte{0xC7}},
				{cmd: masterActivation},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			clear(&got, tc.color, &tc.opts)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("updateDisplay() difference (-got +want):\n%s", diff)
			}
		})
	}
}
