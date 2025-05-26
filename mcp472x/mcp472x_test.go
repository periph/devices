// Copyright 2025 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package mcp472x

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/conn/v3/physic"
)

var liveDevice bool

func getDev(testName string, variant Variant, vRef physic.ElectricPotential) (*Dev, error) {
	addr := DefaultAddress
	if variant == MCP4725 {
		addr = 0x62
	}
	d, err := New(&i2ctest.Playback{Ops: recordingData[testName], DontPanic: true}, addr, variant, vRef)
	return d, err
}

func TestBasic(t *testing.T) {
	d, err := New(nil, 0, MCP4728, MCP4728InternalRef)
	if err != nil {
		t.Fatal(err)
	}

	s := d.String()
	if len(s) == 0 {
		t.Error("expected string received \"\"")
	}
}

func TestPotentialToCount(t *testing.T) {

	d, err := New(nil, 0, MCP4728, MCP4728InternalRef)
	if err != nil {
		t.Fatal(err)
	}

	count, boost, err := d.PotentialToCount(0)
	if err != nil {
		t.Error(err)
	}
	if count != 0 {
		t.Errorf("v=0, count=%d", count)
	}
	if boost {
		t.Errorf("v=0 boost=%t", boost)
	}

	count, boost, err = d.PotentialToCount(MCP4728InternalRef)
	if err != nil {
		t.Error(err)
	}
	if count != maxCount {
		t.Errorf("v=%s, count=%d", MCP4728InternalRef, count)
	}
	if boost {
		t.Errorf("count=%d boost=%t", count, boost)
	}
	count, boost, _ = d.PotentialToCount(MCP4728InternalRef >> 1)
	if count != (stepCount >> 1) {
		t.Errorf("invalid count expected %d received %d", (stepCount>>1)-1, count)
	}
	if boost {
		t.Errorf("count=%d boost=%t", count, boost)
	}

	_, _, err = d.PotentialToCount(physic.ElectricPotential(-1))
	if err == nil {
		t.Error("expected error on negative voltage")
	}
	_, _, err = d.PotentialToCount(3 * MCP4728InternalRef)
	if err == nil {
		t.Error("expected error on out of range voltage")
	}
	_, boost, _ = d.PotentialToCount(4 * physic.Volt)
	if !boost {
		t.Error("expected boost for 4v output")
	}
	d, _ = New(nil, 0, MCP4725, 3300*physic.MilliVolt)
	_, _, err = d.PotentialToCount(5 * physic.Volt)
	if err == nil {
		t.Error("expected error on out of range voltage")
	}
}

func TestOutputParams4725(t *testing.T) {
	d, err := New(nil, 0, MCP4725, 3_300*physic.MilliVolt)
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		outputParam   SetOutputParam
		expectedBytes []byte
	}{
		{SetOutputParam{DAC: 0,
			V:      0,
			PDMode: PDMode1K,
		},
			[]byte{0x42, 0x0, 0x0},
		},
		{SetOutputParam{DAC: 1,
			V:      MCP4728InternalRef >> 1,
			PDMode: PDMode500K,
		},
			[]byte{0x46, 0x4f, 0x70},
		},
		{SetOutputParam{DAC: 2,
			V:      MCP4728InternalRef >> 2,
			PDMode: PDModeNormal,
		},
			[]byte{0x40, 0x27, 0xb0},
		},
		{SetOutputParam{DAC: 3,
			V:      MCP4728InternalRef >> 9,
			PDMode: PDMode100K,
		},
			[]byte{0x44, 0x0, 0x50},
		},
	}
	for _, tc := range testCases {
		bytes := d.paramToBytes(&tc.outputParam)
		if len(bytes) == len(tc.expectedBytes) {
			for ix := range bytes {
				if bytes[ix] != tc.expectedBytes[ix] {
					t.Errorf("for OutputParam %s, byte %d got 0x%x %.8b expected 0x%x %.8b", tc.outputParam.String(), ix, bytes[ix], bytes[ix], tc.expectedBytes[ix], tc.expectedBytes[ix])
				}
			}
		} else {
			t.Errorf("testcase %s expected %d bytes, received %d", tc.outputParam.String(), len(tc.expectedBytes), len(bytes))
		}
	}

	// Test Equality
	d1 := SetOutputParam{DAC: 1, V: physic.Volt, UseInternalRef: true, BoostGain: false}
	d2 := SetOutputParam{DAC: 1, V: physic.Volt, UseInternalRef: true, BoostGain: false}
	if !d1.Equal(d2) {
		t.Errorf("expected d1==d2")
	}
	// Test Inequality
	d1.V = physic.MilliVolt
	if d1.Equal(d2) {
		t.Errorf("expected d1!=d2")
	}

	// Test the String method works and we can unmarshal the JSON into
	// a new SetOutputParam
	s2 := d2.String()
	d3 := SetOutputParam{}
	err = json.Unmarshal([]byte(s2), &d3)
	if err != nil {
		t.Error(err)
	}
	if !d3.Equal(d2) {
		t.Errorf("Expected unmarshal from string to equal original\ns2=%s, d3=%s", s2, d3.String())
	}
}

func TestOutputParams4728(t *testing.T) {
	d, err := New(nil, 0, MCP4728, MCP4728InternalRef)
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		outputParam   SetOutputParam
		expectedBytes []byte
	}{
		{SetOutputParam{DAC: 0,
			V:              0,
			BoostGain:      false,
			UseInternalRef: true,
			PDMode:         PDMode1K,
		},
			[]byte{0x0, 0xa0, 0x0},
		},
		{SetOutputParam{DAC: 1,
			V:              MCP4728InternalRef >> 1,
			BoostGain:      false,
			UseInternalRef: true,
			PDMode:         PDMode500K,
		},
			[]byte{0x02, 0xe8, 0x00},
		},
		{SetOutputParam{DAC: 2,
			V:      MCP4728InternalRef >> 2,
			PDMode: PDModeNormal,
		},
			[]byte{0x04, 0x04, 0x00},
		},
		{SetOutputParam{DAC: 3,
			V:      MCP4728InternalRef >> 6,
			PDMode: PDMode100K,
		},
			[]byte{0x06, 0x40, 0x40},
		},
	}
	for _, tc := range testCases {
		bytes := d.paramToBytes(&tc.outputParam)
		if len(bytes) == len(tc.expectedBytes) {
			for ix := range bytes {
				if bytes[ix] != tc.expectedBytes[ix] {
					t.Errorf("for OutputParam %s, byte %d got 0x%x expected 0x%x", tc.outputParam.String(), ix, bytes[ix], tc.expectedBytes[ix])
				}
			}
		} else {
			t.Errorf("testcase %s expected %d bytes, received %d", tc.outputParam.String(), len(tc.expectedBytes), len(bytes))
		}

	}
}

func testSetGetOutput(d *Dev, t *testing.T) {
	outputs := make([]SetOutputParam, d.maxChannels)
	for i := range physic.ElectricPotential(d.maxChannels) {
		outputs[i] = SetOutputParam{DAC: byte(i), V: (i + 1) * 256 * physic.MilliVolt, UseInternalRef: true, PDMode: PDModeNormal}
		t.Logf("outputs[%d]=%s", i, outputs[i].String())
	}
	err := d.SetOutput(outputs...)
	if err != nil {
		t.Fatal(err)
	}

	cur, eeprom, err := d.GetOutput()
	if err != nil {
		t.Error(err)
	}
	t.Logf("cur=%#v cur[0].V=%s", cur, cur[0].V.String())
	if len(cur) != d.maxChannels {
		t.Errorf("expected %d channels of current output values", d.maxChannels)
	}

	for ix, op := range outputs {
		curChannel := cur[ix]
		diffMilliVolt := math.Abs(float64((curChannel.V - op.V) / physic.MilliVolt))
		if diffMilliVolt > 2.0 {
			t.Errorf("Channel %d Read after program. Expected < 2mV, got %f", op.DAC, diffMilliVolt)
		}
	}

	t.Logf("eeprom=%#v", eeprom)
	if len(eeprom) != d.maxChannels {
		t.Errorf("expected %d channels of eeprom output values", d.maxChannels)
	}
	if liveDevice {
		time.Sleep(30 * time.Second)
	}
}

func testSetEEPROM(d *Dev, dac byte, t *testing.T) {
	op := SetOutputParam{DAC: dac, V: 512 * physic.MilliVolt, UseInternalRef: true, PDMode: PDModeNormal}
	for range 2 {
		err := d.SetOutputWithSave(op)
		if err != nil {
			t.Fatal(err)
		}

		cur, eeprom, err := d.GetOutput()
		if err != nil {
			t.Error(err)
		}
		curChannel := cur[op.DAC]
		diffMilliVolt := math.Abs(float64((curChannel.V - op.V) / physic.MilliVolt))
		if diffMilliVolt > 2.0 {
			t.Errorf("Read after program. Expected difference <2mV, got %f", diffMilliVolt)
		}

		curEeprom := eeprom[op.DAC]
		diffMilliVolt = math.Abs(float64((curEeprom.V - op.V) / physic.MilliVolt))
		if diffMilliVolt > 2.0 {
			t.Errorf("Read EEPROM after program. Expected <2mV, got %f", diffMilliVolt)
		}
		if curEeprom.PDMode != op.PDMode {
			t.Errorf("Read EEPROM after program. Expected PDMode %d, got %d", op.PDMode, curEeprom.PDMode)
		}
		op.V *= 2
		op.PDMode += 1
	}
}

func testFastWrite(d *Dev, t *testing.T) {
	t.Logf("testFastWrite(variant=%s)", d.variant)

	vals := make([]uint16, d.maxChannels)
	t.Log("Writing 0V to all channels")
	err := d.FastWrite(vals...)
	if err != nil {
		t.Fatal(err)
	}
	cur, _, err := d.GetOutput()
	if err != nil {
		t.Error(err)
	}
	for ix, val := range cur {
		if val.V != 0 {
			t.Errorf("Channel %d expected 0V, got %s", ix, val.V)
		}
	}
	if liveDevice {
		time.Sleep(10 * time.Second)
	}

	// Now, write 768mV to all channels.
	vTest := 768 * physic.MilliVolt
	count, _, _ := d.PotentialToCount(vTest)
	for ix := range len(vals) {
		vals[ix] = count
	}
	t.Logf("Writing %s to all channels", vTest)
	err = d.FastWrite(vals...)
	if err != nil {
		t.Fatal(err)
	}
	cur, _, err = d.GetOutput()
	if err != nil {
		t.Error(err)
	}
	for ix := range len(cur) {
		diffMillivolts := math.Abs(float64((cur[ix].V - vTest) / physic.MilliVolt))
		if diffMillivolts > 2 {
			t.Errorf("Channel %d expected %s, received %s", ix, vTest, cur[ix].V)
		}
	}
	if liveDevice {
		time.Sleep(10 * time.Second)
	}
}

func TestSetGetOutput4728(t *testing.T) {
	d, err := getDev("TestSetGetOutput4728", MCP4728, MCP4728InternalRef)
	if err != nil {
		t.Fatal(err)
	}
	testSetGetOutput(d, t)

	err = d.SetOutput()
	if err == nil || err != errInvalidInputCount {
		t.Error("SetOutput() failed to error on zero outputs")
	}
	err = d.SetOutput(SetOutputParam{}, SetOutputParam{}, SetOutputParam{}, SetOutputParam{}, SetOutputParam{})
	if err == nil || err != errInvalidInputCount {
		t.Error("SetOutput() failed to error on too many outputs")
	}

}

func TestSetEEPROM4728(t *testing.T) {
	d, err := getDev("TestSetEEPROM4728", MCP4728, MCP4728InternalRef)
	if err != nil {
		t.Fatal(err)
	}
	testSetEEPROM(d, 2, t)

	err = d.SetOutputWithSave()
	if err == nil || err != errInvalidInputCount {
		t.Error("SetOutputWithSave() failed to error on zero outputs")
	}
	err = d.SetOutputWithSave(SetOutputParam{}, SetOutputParam{}, SetOutputParam{}, SetOutputParam{}, SetOutputParam{})
	if err == nil || err != errInvalidInputCount {
		t.Error("SetOutputWithSave() failed to error on too many outputs")
	}

}

func TestFastWrite4728(t *testing.T) {
	d, err := getDev("TestFastWrite4728", MCP4728, MCP4728InternalRef)
	if err != nil {
		t.Fatal(err)
	}
	testFastWrite(d, t)
}

func TestSetGetOutput4725(t *testing.T) {
	d, err := getDev("TestSetGetOutput4725", MCP4725, 3_300*physic.MilliVolt)
	if err != nil {
		t.Fatal(err)
	}
	testSetGetOutput(d, t)
}

func TestSetEEPROM4725(t *testing.T) {
	d, err := getDev("TestSetEEPROM4725", MCP4725, 3_300*physic.MilliVolt)
	if err != nil {
		t.Fatal(err)
	}
	testSetEEPROM(d, 0, t)
}

func TestFastWrite4725(t *testing.T) {
	d, err := getDev("TestFastWrite4725", MCP4725, 3_300*physic.MilliVolt)
	if err != nil {
		t.Fatal(err)
	}
	testFastWrite(d, t)
}
