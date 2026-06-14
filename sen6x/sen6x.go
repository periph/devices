// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
)

const i2cAddr = 0x6b

// deviceSamplingInterval is the sensor's native sampling interval. The
// datasheet specifies a sampling interval of 1 ± 0.03 seconds so we take
// the maximum value to be safe.
var deviceSamplingInterval = 1030 * time.Millisecond

// SensorValues contains the measurements returned from the device.
// Any values not relevant to the sensor model being used will be nil.
// Any values equal to the device's "unknown" sentinel value (0x7fff or 0xfff
// depending on data type) will also be nil.
type SensorValues struct {
	// Particulate matter measurements in μg/m³.
	PM1, PM25, PM4, PM10 *float32

	// Relative humidity.
	RH *float32

	// Temp in °C.
	Temp *float32

	// VOC level in terms of Sensirion's VOC index.
	VOC *float32

	// NOx level in terms of Sensirion's NOx index.
	NOx *float32

	// CO2 concentration in ppm.
	CO2 *int16

	// Formaldehyde (HCHO) concentration in ppb.
	HCHO *float32
}

// RawSensorValues contains the raw measurements returned from the device.
// Any values not relevant to the sensor model being used will be nil.
// Any values equal to the device's "unknown" sentinel value (0x7fff or 0xfff
// depending on data type) will also be nil.
type RawSensorValues struct {
	// Raw measured relative humidity.
	RH *float32

	// Raw measured temp in °C.
	Temp *float32

	// Raw measured VOC ticks without scale factor.
	VOC *uint16

	// Raw measured NOx ticks without scale factor.
	NOx *uint16

	// Non-interpolated CO2 concentration in ppm updated every five seconds.
	//
	// NOTE: This is only applicable to SEN66. While SEN63C and SEN69C also have
	// CO2 sensors, only SEN66 returns raw CO2 measurements.
	CO2 *uint16
}

// Dev represents a SEN6x sensor.
type Dev struct {
	dev   *i2c.Dev
	model Model

	// Sleep function that can be redefined for tests.
	// Defaults to time.Sleep.
	sleep func(time.Duration)

	mu   sync.Mutex
	stop chan struct{}
	wg   sync.WaitGroup
}

// New creates a new SEN6x device.
func New(bus i2c.Bus, model Model) *Dev {
	return &Dev{
		dev:   &i2c.Dev{Bus: bus, Addr: i2cAddr},
		model: model,
		sleep: time.Sleep,
	}
}

func (d *Dev) String() string {
	return d.model.String()
}

// SenseContinuous puts the sensor in measurement mode and sends measurements over
// the returned channel at the given interval. Call [Dev.Halt] to stop measurement
// and ensure resources are cleaned up (e.g. that the channel is closed and goroutines
// are stopped).
//
// After starting the measurement it takes some time (~1.1 s) until the first
// measurement results are available.
//
// It's the responsibility of the caller to retrieve the values from the
// channel as fast as possible, otherwise the interval may not be respected.
//
// Note on the interval: The sensor's internal measurement interval is 1 ± 0.03
// seconds, so an interval value less than that duration will return values at
// the device's native interval. Higher interval values will work as expected.
//
// Note for SEN63C and SEN69C only: SEN63C and SEN69C condition the CO2 sensor
// during the first 24 seconds after starting a measurement. As this process
// cannot be interrupted, the following limitations apply during this period:
//   - You may stop the measurement if needed, but do not start it again until
//     at least 24 seconds have passed to avoid a CO2 sensor error.
//   - Do not stop the sensor and call [Dev.PerformForcedCO2Recalibration],
//     [Dev.SetCO2SensorAutomaticSelfCalibration], or [Dev.PerformCO2SensorFactoryReset].
func (d *Dev) SenseContinuous(interval time.Duration) (<-chan *SensorValues, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stop != nil {
		return nil, errors.New("sen6x: already sensing continuously")
	}

	results := make(chan *SensorValues)
	d.stop = make(chan struct{})
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer close(results)
		d.doSenseContinuous(interval, results, d.stop)
	}()

	return results, nil
}

func (d *Dev) doSenseContinuous(interval time.Duration, results chan<- *SensorValues, stop <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()

	if err := d.StartContinuousMeasurement(); err != nil {
		log.Printf("sen6x: failed to start continuous measurement: %v", err)
		return
	}

	for {
		d.mu.Lock()

		if interval < deviceSamplingInterval {
			if err := d.waitOnDataReady(stop); err != nil {
				d.mu.Unlock()
				log.Printf("sen6x: failed to check if data is ready: %v", err)
				return
			}
		}

		sv, err := d.doReadMeasuredValues()
		d.mu.Unlock()

		if err != nil {
			log.Printf("sen6x: failed to read measured values: %v", err)
			return
		}

		select {
		case results <- sv:
		case <-stop:
			return
		}

		select {
		case <-stop:
			return
		case <-t.C:
		}
	}
}

func (d *Dev) waitOnDataReady(stop <-chan struct{}) error {
	t := time.NewTicker(300 * time.Millisecond)
	defer t.Stop()

	for {
		ready, err := d.doGetDataReady()
		if err != nil {
			return err
		}
		if ready {
			return nil
		}

		select {
		case <-stop:
			return nil
		case <-t.C:
		}
	}
}

// Halt halts continuous sensing, cleans up resources, and puts the sensor in idle mode.
func (d *Dev) Halt() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stop == nil {
		return nil
	}

	close(d.stop)
	d.stop = nil
	d.wg.Wait()

	return d.writeAndWait(cmdStopMeasurement, nil)
}

// StartContinuousMeasurement starts a continuous measurement. After starting the
// measurement, it takes some time (~1.1 s) until the first measurement results are
// available.
//
// You may poll [Dev.GetDataReady] to check if results are ready to be read.
//
// Note for SEN63C and SEN69C only: SEN63C and SEN69C condition the CO2 sensor
// during the first 24 seconds after starting a measurement. As this process
// cannot be interrupted, the following limitations apply during this period:
//   - You may stop the measurement if needed, but do not start it again until
//     at least 24 seconds have passed to avoid a CO2 sensor error.
//   - Do not stop the sensor and call [Dev.PerformForcedCO2Recalibration],
//     [Dev.SetCO2SensorAutomaticSelfCalibration], or [Dev.PerformCO2SensorFactoryReset].
func (d *Dev) StartContinuousMeasurement() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdStartContinuousMeasurement, nil)
}

// StopMeasurement stops the measurement and returns the sensor to idle mode.
// After sending this command, wait at least 1400 ms before starting a new measurement.
func (d *Dev) StopMeasurement() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdStopMeasurement, nil)
}

// GetDataReady checks if new measurement results are ready to read. The data ready
// flag is automatically reset after reading the measurement values.
func (d *Dev) GetDataReady() (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.doGetDataReady()
}

func (d *Dev) doGetDataReady() (bool, error) {
	data, err := d.writeAndRead(cmdGetDataReady, nil)
	if err != nil {
		return false, err
	}

	return data[1] == 1, nil
}

// ReadMeasuredValues reads the current measured values. Measurement must have
// already been started by [Dev.StartContinuousMeasurement]. After starting the
// measurement, it takes some time (~1.1 s) until the first measurement results
// are available.
//
// [Dev.GetDataReady] may be polled to check if new data is available since the
// last read operation. If no new data is available, the previous values will
// be returned. If no data is available at all for a particular measurement (e.g.
// measurement not running for at least one second), it will be nil.
//
// Any values that aren't applicable to the Dev's sensor model will be nil.
func (d *Dev) ReadMeasuredValues() (*SensorValues, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.doReadMeasuredValues()
}

func (d *Dev) doReadMeasuredValues() (*SensorValues, error) {
	var cmd command
	switch d.model {
	case SEN62:
		cmd = cmdReadMeasuredValuesSEN62
	case SEN63C:
		cmd = cmdReadMeasuredValuesSEN63C
	case SEN65:
		cmd = cmdReadMeasuredValuesSEN65
	case SEN66:
		cmd = cmdReadMeasuredValuesSEN66
	case SEN68:
		cmd = cmdReadMeasuredValuesSEN68
	case SEN69C:
		cmd = cmdReadMeasuredValuesSEN69C
	default:
		return nil, fmt.Errorf("sen6x: unknown model: %v", d.model)
	}

	data, err := d.writeAndRead(cmd, nil)
	if err != nil {
		return nil, err
	}

	sv := &SensorValues{}

	if rawPM1 := binary.BigEndian.Uint16(data[0:2]); rawPM1 != 0xffff {
		sv.PM1 = ptr(float32(rawPM1) / 10.0)
	}
	if rawPM25 := binary.BigEndian.Uint16(data[2:4]); rawPM25 != 0xffff {
		sv.PM25 = ptr(float32(rawPM25) / 10.0)
	}
	if rawPM4 := binary.BigEndian.Uint16(data[4:6]); rawPM4 != 0xffff {
		sv.PM4 = ptr(float32(rawPM4) / 10.0)
	}
	if rawPM10 := binary.BigEndian.Uint16(data[6:8]); rawPM10 != 0xffff {
		sv.PM10 = ptr(float32(rawPM10) / 10.0)
	}

	if rawRH := int16(binary.BigEndian.Uint16(data[8:10])); rawRH != 0x7fff {
		sv.RH = ptr(float32(rawRH) / 100.0)
	}
	if rawTemp := int16(binary.BigEndian.Uint16(data[10:12])); rawTemp != 0x7fff {
		sv.Temp = ptr(float32(rawTemp) / 200.0)
	}

	i := 12
	if d.model.hasVOCNOx() {
		if rawVOC := int16(binary.BigEndian.Uint16(data[i : i+2])); rawVOC != 0x7fff {
			sv.VOC = ptr(float32(rawVOC) / 10.0)
		}
		i += 2

		if rawNOx := int16(binary.BigEndian.Uint16(data[i : i+2])); rawNOx != 0x7fff {
			sv.NOx = ptr(float32(rawNOx) / 10.0)
		}
		i += 2
	}

	if d.model.hasHCHO() {
		if rawHCHO := binary.BigEndian.Uint16(data[i : i+2]); rawHCHO != 0xffff {
			sv.HCHO = ptr(float32(rawHCHO) / 10.0)
		}
		i += 2
	}

	if d.model.hasCO2() {
		if d.model == SEN66 {
			// SEN66 encodes CO2 concentration as a uint16.
			if rawCO2 := binary.BigEndian.Uint16(data[i : i+2]); rawCO2 != 0xffff {
				sv.CO2 = ptr(int16(rawCO2))
			}
		} else {
			// SEN63C and SEN69C encode CO2 concentration as an int16.
			if rawCO2 := int16(binary.BigEndian.Uint16(data[i : i+2])); rawCO2 != 0x7fff {
				sv.CO2 = ptr(rawCO2)
			}
		}
	}

	return sv, nil
}

// ReadMeasuredRawValues reads the current raw measured values.
//
// Any values that aren't applicable to Dev's sensor model will be nil.
//
// [Dev.GetDataReady] may be used to check if new data is available since the
// last read operation. If no new data is available, the previous values will
// be returned. If no data is available at all (e.g. measurement not running
// for at least one second), all values will be nil.
func (d *Dev) ReadMeasuredRawValues() (*RawSensorValues, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var cmd command
	switch d.model {
	case SEN62, SEN63C:
		cmd = cmdReadMeasuredRawValuesSEN62SEN63C
	case SEN65, SEN68, SEN69C:
		cmd = cmdReadMeasuredRawValuesSEN65SEN68SEN69C
	case SEN66:
		cmd = cmdReadMeasuredRawValuesSEN66
	default:
		return nil, fmt.Errorf("sen6x: unknown model: %v", d.model)
	}

	data, err := d.writeAndRead(cmd, nil)
	if err != nil {
		return nil, err
	}

	rv := &RawSensorValues{}

	if rawRH := int16(binary.BigEndian.Uint16(data[0:2])); rawRH != 0x7fff {
		rv.RH = ptr(float32(rawRH) / 100)
	}
	if rawTemp := int16(binary.BigEndian.Uint16(data[2:4])); rawTemp != 0x7fff {
		rv.Temp = ptr(float32(rawTemp) / 200)
	}

	i := 4
	if d.model.hasVOCNOx() {
		if rawVOC := binary.BigEndian.Uint16(data[i : i+2]); rawVOC != 0xffff {
			rv.VOC = ptr(rawVOC)
		}
		i += 2

		if rawNOx := binary.BigEndian.Uint16(data[i : i+2]); rawNOx != 0xffff {
			rv.NOx = ptr(rawNOx)
		}
		i += 2
	}

	// We check specifically for SEN66 here instead of using d.model.hasCO2()
	// because while SEN63C and SEN69C also have CO2 sensors, only SEN66 returns
	// raw CO2 measurements.
	if d.model == SEN66 {
		if rawCO2 := binary.BigEndian.Uint16(data[i : i+2]); rawCO2 != 0xffff {
			rv.CO2 = ptr(rawCO2)
		}
	}

	return rv, nil
}

// GetProductName gets the product name from the device.
func (d *Dev) GetProductName() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdGetProductName, nil)
	if err != nil {
		return "", err
	}

	return string(data[:clen(data)]), nil
}

// GetSerialNumber gets the serial number from the device.
func (d *Dev) GetSerialNumber() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdGetSerialNumber, nil)
	if err != nil {
		return "", err
	}

	return string(data[:clen(data)]), nil
}

// GetVersion gets the firmware version, returning the major and minor version numbers.
func (d *Dev) GetVersion() (uint8, uint8, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdGetVersion, nil)
	if err != nil {
		return 0, 0, err
	}

	return data[0], data[1], nil
}

// DeviceReset executes a reset on the device. This has the same effect as a power cycle.
func (d *Dev) DeviceReset() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdDeviceReset, nil)
}

// StartFanCleaning triggers fan cleaning. The fan is set to the maximum speed for
// 10 seconds and then automatically stopped. Wait at least 10s after this command
// before starting a measurement.
func (d *Dev) StartFanCleaning() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdStartFanCleaning, nil)
}

// writeAndWait writes a command followed by optional txData and
// waits for its execution time.
func (d *Dev) writeAndWait(cmd command, txData []byte) error {
	buf := make([]byte, 2+len(txData))
	buf[0] = byte(cmd.id >> 8)
	buf[1] = byte(cmd.id)
	copy(buf[2:], txData)

	if err := d.dev.Tx(buf[:], nil); err != nil {
		return err
	}

	d.sleep(cmd.execTime)

	return nil
}

// writeAndRead writes a command followed by optional txData, waits for
// the command's execution time, and then reads the response.
func (d *Dev) writeAndRead(cmd command, txData []byte) ([]byte, error) {
	if err := d.writeAndWait(cmd, txData); err != nil {
		return nil, err
	}

	read := make([]byte, cmd.rxDataLen)
	if err := d.dev.Tx(nil, read); err != nil {
		return nil, err
	}

	rxData, err := validateAndStripCRC(read)
	if err != nil {
		return nil, err
	}

	return rxData, nil
}

var _ conn.Resource = &Dev{}
