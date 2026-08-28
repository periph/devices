// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package bme68x

import (
	"errors"
	"testing"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/conn/v3/physic"
)

// TestDev_validateDeviceID tests the validateDeviceID method of the Dev struct.
// It ensures that the device correctly validates both the chip ID and variant ID
// over I2C and returns the expected errors for invalid cases.
func TestDev_validateDeviceID(t *testing.T) {
	for _, test := range []struct {
		name      string
		ops       []i2ctest.IO
		expectErr error
	}{
		{
			name: "success",
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{byte(regID)}, R: []byte{ChipDeviceID}},
				{Addr: I2CAddr, W: []byte{byte(regVariantID)}, R: []byte{0x0}},
			},
			expectErr: nil,
		},
		{
			name:      "chipIdFailure",
			ops:       []i2ctest.IO{{Addr: I2CAddr, W: []byte{byte(regID)}, R: []byte{0x62}}},
			expectErr: ErrInvalidChipId,
		},
		{
			name: "variantIdFailure",
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{byte(regID)}, R: []byte{ChipDeviceID}},
				{Addr: I2CAddr, W: []byte{byte(regVariantID)}, R: []byte{0x4}},
			},
			expectErr: ErrInvalidVariantId,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			dev := Device{}
			dev.d = i2c.Dev{Bus: &b, Addr: I2CAddr}
			err := dev.validateDeviceID()
			if !errors.Is(err, test.expectErr) {
				t.Fatalf("Expected error: %v, got: %v", test.expectErr, err)
			}
			if err := b.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestNewI2cAddrErr tests that creating a new Dev with an invalid I2C address
// returns the expected ErrI2cAddress error.
func TestNewI2cAddrErr(t *testing.T) {
	var invalidI2cAddr uint16 = 0x60
	b := i2ctest.Playback{
		Ops:       []i2ctest.IO{},
		DontPanic: true,
	}
	_, err := NewI2C(&b, invalidI2cAddr)
	if !errors.Is(err, ErrI2cAddress) {
		t.Fatalf("Expected error: %v, got: %v", ErrI2cAddress, err)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestDev_IsNewMeasurementReady tests the IsNewMeasurementReady method of Dev.
// It verifies that the method correctly interprets the sensor status register.
func TestDev_IsNewMeasurementReady(t *testing.T) {
	for _, test := range []struct {
		name      string
		ops       []i2ctest.IO
		want      bool
		expectErr error
	}{
		{
			name:      "measurementReady",
			ops:       []i2ctest.IO{{Addr: I2CAddr, W: []byte{byte(regEASStatus0)}, R: []byte{0x80}}},
			expectErr: nil,
			want:      true,
		},
		{
			name:      "measurementNotReady",
			ops:       []i2ctest.IO{{Addr: I2CAddr, W: []byte{byte(regEASStatus0)}, R: []byte{0x70}}},
			expectErr: nil,
			want:      false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			dev := Device{d: i2c.Dev{Bus: &b, Addr: I2CAddr}, variant: VariantNameBME680}
			dev.ops = dev.newBME680()
			status, err := dev.IsNewMeasurementReady()
			if !errors.Is(err, test.expectErr) {
				t.Fatalf("Expected error: %v, got: %v", test.expectErr, err)
			}
			if status != test.want {
				t.Fatalf("Expected status: %v, got: %v", test.want, status)
			}
			if err := b.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestDev_Sense tests the Sense() method of the BME680 driver.
// It simulates I2C transactions and verifies that temperature, pressure,
// humidity, and gas resistance readings are correctly returned and compensated.
func TestDev_Sense(t *testing.T) {
	for _, test := range []struct {
		name         string
		ops          []i2ctest.IO
		testCfg      SensorConfig
		testGasIndex int8
		want         physic.Env
		gasRes       GasResistance
		gasValid     bool
		expectErr    error
	}{
		{
			name: "success",
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{byte(regCtrlMeas), 0x55}, R: []byte{}},
				{Addr: I2CAddr, W: []byte{byte(regPressMSB)}, R: []byte{71, 58, 176, 118, 196, 16, 96, 8}},
				{Addr: I2CAddr, W: []byte{byte(regGasRMsb)}, R: []byte{98, 186}},
			},
			testCfg: SensorConfig{TempOversampling: OS2x, PressureOversampling: OS16x, HumidityOversampling: OS1x,
				IIRFilter: NoFilter, GasEnabled: true, OperatingMode: ForcedMode, GasProfiles: defaultGasProfiles(),
			},
			testGasIndex: 0,
			expectErr:    nil,
			want:         physic.Env{Temperature: 2260*10*physic.MilliCelsius + physic.ZeroCelsius, Pressure: 101860.0 * physic.Pascal, Humidity: 67 * physic.PercentRH},
			gasRes:       8514,
			gasValid:     true,
		},
		{
			name: "gas sensor disabled",
			testCfg: SensorConfig{
				GasEnabled:    false,
				OperatingMode: ForcedMode,
				GasProfiles:   defaultGasProfiles(),
			},
			testGasIndex: 0,
			expectErr:    ErrNoGasProfileSelected,
			want:         physic.Env{},
			gasRes:       0,
			gasValid:     false,
		},
		{
			name:         "no active gas profile",
			testCfg:      SensorConfig{GasEnabled: true, OperatingMode: ForcedMode},
			testGasIndex: -1,
			expectErr:    ErrNoGasProfileSelected,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			dev := Device{d: i2c.Dev{Bus: &b, Addr: I2CAddr},
				c:                     mockCalibration(),
				cfg:                   test.testCfg,
				activeGasProfileIndex: test.testGasIndex,
				variant:               VariantNameBME680,
			}
			// Assign sensorOps implementation
			dev.ops = dev.newBME680()
			eExp, gExp, gValid, err := dev.Sense()
			if !errors.Is(err, test.expectErr) {
				t.Fatalf("Expected error: %v, got: %v", test.expectErr, err)
			}
			// Compare Env safely with delta tolerance for Humidity
			assertEnvEqual(t, eExp, test.want, 0.1, 1.0, 0.01)
			// Compare gas sensor
			if gExp != test.gasRes {
				t.Fatalf("Gas resistance mismatch: got %v, want %v", gExp, test.gasRes)
			}
			if gValid != test.gasValid {
				t.Fatalf("Gas valid mismatch: got %v, want %v", gValid, test.gasValid)
			}
		})
	}
}

// TestDev_InitCalibration tests the initialization of sensor calibration data.
// It uses i2c test Playback to simulate I2C register reads and verifies
// that the Dev.InitCalibration method correctly populates the calibration struct.
func TestDev_InitCalibration(t *testing.T) {
	for _, test := range []struct {
		name      string
		ops       []i2ctest.IO
		want      SensorCalibration
		expectErr error
	}{
		{
			name: "success",
			ops: []i2ctest.IO{
				{Addr: I2CAddr, W: []byte{byte(regParT1)}, R: []byte{254, 100}},
				{Addr: I2CAddr, W: []byte{byte(regParT2)}, R: []byte{181, 101, 3, 240, 209, 142, 117, 215, 88, 0, 159, 38, 8, 255, 38, 30, 0, 0, 106, 247, 36, 245, 30}},
				{Addr: I2CAddr, W: []byte{byte(regParH2)}, R: []byte{63, 91, 47, 0, 45, 20, 120, 156}},
				{Addr: I2CAddr, W: []byte{byte(regParG2)}, R: []byte{81, 211, 186, 18}},
				{Addr: I2CAddr, W: []byte{byte(regResHeatVal)}, R: []byte{54, 170, 22, 73, 19}},
			},
			expectErr: nil,
			want:      mockCalibration(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := i2ctest.Playback{
				Ops:       test.ops,
				DontPanic: true,
			}
			dev := Device{d: i2c.Dev{Bus: &b, Addr: I2CAddr}, variant: VariantNameBME680}
			err := dev.InitCalibration()
			if !errors.Is(err, test.expectErr) {
				t.Fatalf("Expected error: %v, got: %v", test.expectErr, err)
			}
			if dev.c != test.want {
				t.Fatalf("Expected calibration: %v, got: %v", test.want, dev.c)
			}
			if err := b.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestCompensations verifies that the sensor compensation functions produce correct results
// given known raw ADC readings and a mocked calibration. This ensures the Dev methods
// for temperature, pressure, humidity, and gas sensor calculations are accurate.
func TestCompensations(t *testing.T) {
	d := Device{c: mockCalibration(), variant: VariantNameBME680}
	// Raw Sensor ADC Readings
	var tRaw uint32 = 480355
	var pRaw uint32 = 291843
	var hRaw uint32 = 24615
	var gRaw uint32 = 386
	var gasResRange gasRangeR = 10
	// Expected compensated results (from datasheet formulas / reference implementation)
	var tExp int32 = 2070   // Temperature in hundredths of °C (20.70°C)
	var pExp int32 = 101513 // Pressure in Pa
	var hExp int32 = 67561  // Humidity in thousandths of %RH (67.561%)
	var gExp int32 = 8566   // Gas resistance in Ohms

	tComp, pComp, hComp, gComp := expectedCompensated(&d, tRaw, pRaw, hRaw, gRaw, gasResRange)
	if tComp != tExp { // °C
		t.Fatalf("temp compensation does not match expected value : %v, got: %v", tExp, tComp)
	}
	if pComp != pExp { //Pa
		t.Fatalf("pressure compensation does not match expected = %v, got: %v", pExp, pComp)
	}
	if hComp != hExp {
		t.Fatalf("Humidity compensation does not match expected = %v, got: %v", hExp, hComp)
	}
	if gComp != gExp { //Ohm
		t.Fatalf("Gas compensation does not match expected = %v, got: %v", gExp, gComp)
	}
}

// mockCalibration returns a SensorCalibration struct populated with fixed calibration constants.
// These values simulate a real sensor's calibration data and are used in unit tests to
// produce deterministic compensation outputs.
func mockCalibration() SensorCalibration {
	return SensorCalibration{
		t1: 25854, t2: 26037, t3: 3,
		p1: 36561, p2: -10379, p3: 88, p4: 9887,
		p5: -248, p6: 30, p7: 38, p8: -2198, p9: -2780, p10: 30,
		h1: 763, h2: 1013, h3: 0, h4: 45, h5: 20, h6: 120, h7: -100,
		g1: -70, g2: -11439, g3: 18,
		resHeatVal: 54, resHeatRange: 2, switchingErr: 19,
		tFine: 0, tempComp: 0, pressureComp: 0, humidityComp: 0,
	}
}

// expectedCompensated computes the fully compensated sensor readings for temperature, pressure,
// humidity, and gas resistance based on raw ADC values. This helper is used in tests to
// compare actual sensor compensation outputs against expected results.
func expectedCompensated(dev *Device, tRaw, pRaw, hRaw, gRaw uint32, gasRange gasRangeR) (int32, int32, int32, int32) {
	return dev.compensatedTemperature(tRaw),
		dev.compensatedPressure(pRaw),
		dev.compensatedHumidity(hRaw),
		dev.compensatedGasSensor(gRaw, gasRange)
}

// assertEnvEqual compares two physic.Env values (got vs want) with specified tolerances.
// deltaTemp, deltaPress, and deltaRH are the allowed differences for temperature, pressure, and humidity respectively.
// This is useful because floating-point conversions or sensor rounding can produce small variations.
func assertEnvEqual(t *testing.T, got, want physic.Env, deltaTemp, deltaPress, deltaRH float64) {
	t.Helper()
	if diff := float64(got.Temperature - want.Temperature); diff < -deltaTemp || diff > deltaTemp {
		t.Fatalf("Temperature mismatch: got %v, want %v", got.Temperature, want.Temperature)
	}
	if diff := float64(got.Pressure - want.Pressure); diff < -deltaPress || diff > deltaPress {
		t.Fatalf("Pressure mismatch: got %v, want %v", got.Pressure, want.Pressure)
	}
	if diff := float64(got.Humidity - want.Humidity); diff < -deltaRH || diff > deltaRH {
		t.Fatalf("Humidity mismatch: got %v, want %v", got.Humidity, want.Humidity)
	}
}

// defaultGasProfiles returns a complete default set of GasProfile entries.
// This ensures that tests have a consistent, fully populated array and prevents
// accidental gaps if additional profiles are added later.
func defaultGasProfiles() [10]GasProfile {
	var profiles [10]GasProfile
	for i := range profiles {
		// Default values (can be zero or some safe value)
		profiles[i] = GasProfile{TargetTempC: 0, HeatingDurationMs: 0}
	}
	// Set specific profiles used in tests
	profiles[0] = GasProfile{TargetTempC: 300, HeatingDurationMs: 250}
	profiles[7] = GasProfile{TargetTempC: 150, HeatingDurationMs: 100}
	return profiles
}
