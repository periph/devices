// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v4

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

func (r *fakeController) sendByte(data byte) {
	cur := &(*r)[len(*r)-1]
	cur.data = append(cur.data, data)
}

func (*fakeController) readBusy() {
}

func TestInitDisplay(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts Opts
		want []record
	}{
		{
			name: "epd2in13v4",
			opts: EPD2in13v4,
			want: []record{
				{cmd: swReset},
				{
					cmd:  driverOutputControl,
					data: []byte{250 - 1, 0, 0},
				},
				{cmd: dataEntryModeSetting, data: []byte{0x03}},
				{cmd: setRAMXAddressStartEndPosition, data: []uint8{0x00, 0x0f}},
				{cmd: setRAMYAddressStartEndPosition, data: []uint8{0x00, 0x00, 0xf9, 0x00}},
				{cmd: setRAMXAddressCounter, data: []uint8{0x00}},
				{cmd: setRAMYAddressCounter, data: []uint8{0x00, 0x00}},
				{cmd: borderWaveformControl, data: []uint8{0x05}},
				{cmd: displayUpdateControl1, data: []uint8{0x80, 0x80}},
				{cmd: tempSensorSelect, data: []uint8{0x80}},
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

func TestInitDisplayFast(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts Opts
		want []record
	}{
		{
			name: "epd2in13v4",
			opts: EPD2in13v4,
			want: []record{
				{cmd: swReset},
				{cmd: tempSensorSelect, data: []uint8{0x80}},
				{cmd: dataEntryModeSetting, data: []byte{0x03}},
				{cmd: setRAMXAddressStartEndPosition, data: []uint8{0x00, 0x0f}},
				{cmd: setRAMYAddressStartEndPosition, data: []uint8{0x00, 0x00, 0xf9, 0x00}},
				{cmd: setRAMXAddressCounter, data: []uint8{0x00}},
				{cmd: setRAMYAddressCounter, data: []uint8{0x00, 0x00}},
				{cmd: displayUpdateControl2, data: []uint8{0x81}},
				{cmd: masterActivation},
				{cmd: tempSensorRegWrite, data: []uint8{0x64, 0x00}},
				{cmd: displayUpdateControl2, data: []uint8{0x91}},
				{cmd: masterActivation},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			initDisplayFast(&got, &tc.opts)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("initDisplay() difference (-got +want):\n%s", diff)
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

func TestTurnOnDisplayFast(t *testing.T) {
	for _, tc := range []struct {
		name string
		want []record
	}{
		{
			name: "Fast",
			want: []record{
				{cmd: displayUpdateControl2, data: []byte{0xC7}},
				{cmd: masterActivation},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			turnOnDisplayFast(&got)

			if diff := cmp.Diff([]record(got), tc.want, cmpopts.EquateEmpty(), cmp.AllowUnexported(record{})); diff != "" {
				t.Errorf("updateDisplay() difference (-got +want):\n%s", diff)
			}
		})
	}
}

func TestTurnOnDisplayPart(t *testing.T) {
	for _, tc := range []struct {
		name string
		want []record
	}{
		{
			name: "Part",
			want: []record{
				{cmd: displayUpdateControl2, data: []byte{0xFF}},
				{cmd: masterActivation},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fakeController

			turnOnDisplayPart(&got)

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
			opts: EPD2in13v4,
			want: []record{
				{cmd: writeRAMBW, data: buff},
				{cmd: writeRAMRed, data: buff},
				{cmd: displayUpdateControl2, data: []byte{0xf7}},
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
