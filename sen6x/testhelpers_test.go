// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2ctest"
)

// newTestDev creates a Dev wired to the given playback bus, for use in tests.
func newTestDev(t *testing.T, bus *i2ctest.Playback, model Model) *Dev {
	t.Helper()
	return &Dev{
		dev:   &i2c.Dev{Bus: bus, Addr: i2cAddr},
		model: model,
		sleep: func(time.Duration) {},
	}
}

type writeTestCase struct {
	name    string
	model   Model
	tx      []byte
	wantErr bool
}

// runWriteTests runs tests that write to the I2C bus and expect no data in response.
func runWriteTests(t *testing.T, cases []writeTestCase, f func(d *Dev) error) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ops := []i2ctest.IO{}
			if tc.tx != nil {
				ops = append(ops, i2ctest.IO{Addr: i2cAddr, W: tc.tx})
			}

			bus := i2ctest.Playback{
				Ops: ops,
			}
			defer func() {
				if err := bus.Close(); err != nil {
					t.Error(err)
				}
			}()
			d := newTestDev(t, &bus, tc.model)

			err := f(d)

			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

type writeAndReadTestCase[ReturnType any] struct {
	name      string
	model     Model
	tx        []byte
	rx        []byte
	want      ReturnType
	wantErr   bool
	dontPanic bool
}

// runWriteAndReadTests runs tests that write to the I2C bus and expect to read data back.
func runWriteAndReadTests[ReturnType any](t *testing.T, cases []writeAndReadTestCase[ReturnType], f func(d *Dev) (ReturnType, error)) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ops := []i2ctest.IO{}
			if tc.tx != nil {
				ops = append(ops, i2ctest.IO{Addr: i2cAddr, W: tc.tx})
			}
			if tc.rx != nil {
				ops = append(ops, i2ctest.IO{Addr: i2cAddr, R: tc.rx})
			}

			bus := i2ctest.Playback{
				Ops:       ops,
				DontPanic: tc.dontPanic,
			}
			defer func() {
				if err := bus.Close(); err != nil {
					t.Error(err)
				}
			}()
			d := newTestDev(t, &bus, tc.model)

			got, err := f(d)

			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tc.wantErr {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

type senseContinuousTestCase struct {
	name                     string
	model                    Model
	interval                 time.Duration
	expectedMeasurementCount int
	ops                      []i2ctest.IO
	dontPanic                bool
}

// runSenseContinuousTests runs tests that exercise [Dev.SenseContinuous].
func runSenseContinuousTests(t *testing.T, cases []senseContinuousTestCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bus := i2ctest.Playback{
				Ops:       tc.ops,
				DontPanic: tc.dontPanic,
			}
			defer func() {
				if err := bus.Close(); err != nil {
					t.Error(err)
				}
			}()

			d := newTestDev(t, &bus, tc.model)

			ch, err := d.SenseContinuous(tc.interval)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := d.Halt(); err != nil {
					t.Fatal(err)
				}
			}()

			// Verify double start returns error.
			if _, err := d.SenseContinuous(tc.interval); err == nil {
				t.Error("expected error on second SenseContinuous call")
			}

			received := 0
		loop:
			for {
				select {
				case sv, ok := <-ch:
					if !ok {
						break loop
					}
					received++

					if sv == nil {
						t.Error("received nil SensorValues")
					}

					if received == tc.expectedMeasurementCount {
						break loop
					}

				case <-time.After(5 * time.Second):
					t.Fatal("timed out waiting for measurements")
				}
			}

			if received != tc.expectedMeasurementCount {
				t.Errorf("received %d measurements, want %d", received, tc.expectedMeasurementCount)
			}
		})
	}
}
