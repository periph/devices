// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package bme68x_test

import (
	"fmt"
	"time"

	"periph.io/x/conn/v3/i2c/i2creg"
	bme680 "periph.io/x/devices/v3/bme68x"
	"periph.io/x/host/v3"
)

const (
	i2cBus  = "/dev/i2c-1"
	i2cAddr = 0x77
)

func main() {
	if _, err := host.Init(); err != nil {
		fmt.Println("Error: Failed to Host init()")
		return
	}

	b, err := i2creg.Open(i2cBus)
	if err != nil {
		fmt.Printf("failed to open I2C bus: %v", err)
		return
	}
	defer b.Close()

	// Get the Device handler
	d, err := bme680.NewI2C(b, i2cAddr)
	if err != nil {
		fmt.Printf("Error: failed to initialize BME680 sensor: %v\n", err)
	}

	// user configuration
	userCfg := &bme680.SensorConfig{
		TempOversampling: bme680.OS2x, PressureOversampling: bme680.OS16x,
		HumidityOversampling: bme680.OS1x, IIRFilter: bme680.NoFilter,
		GasProfiles: [10]bme680.GasProfile{
			0: {TargetTempC: 300, HeatingDurationMs: 250},
			7: {TargetTempC: 150, HeatingDurationMs: 100},
		},
		GasEnabled:    true,
		OperatingMode: bme680.ForcedMode,
	}

	if err := d.SetupSensor(userCfg); err != nil {
		fmt.Printf("Error: Failed Setup Sensor %v\n", err)
	}
	if err := d.SetGasProfile(0); err != nil {
		fmt.Printf("Error: Failed to select gas profile %v\n", err)
	}

	// Create a ticker to trigger measurements every 15 seconds
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Infinite loop to continuously read measurements at the specified interval
	for range ticker.C {
		env, gasResistance, valid, err := d.Sense()
		if err != nil {
			fmt.Printf("Failed to read sensor: %v", err)
			continue
		}
		fmt.Printf("[%s] Temp: %.3f C, Humidity: %5s, Pressure: %9s, Gas: %s\n",
			time.Now().Format("15:04:05"), env.Temperature.Celsius(), env.Humidity, env.Pressure,
			func() string {
				if valid {
					return fmt.Sprintf("%d Ohm", gasResistance)
				}
				return "INVALID"
			}())
	}
}
