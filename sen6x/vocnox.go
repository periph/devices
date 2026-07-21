// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// VOCNOxAlgorithmTuningParameters represents parameters
// used to customize the VOC and NOx algorithm.
type VOCNOxAlgorithmTuningParameters struct {
	// IndexOffset is the VOC or NOx index representing typical (average) conditions.
	// Allowed values are in range 1..250. Device's default: 100 for VOC, 1 for NOx
	IndexOffset int16

	// LearningTimeOffsetHours is the time constant used to estimate the VOC/NOx
	// algorithm offset from measurement history, in hours. Past events will be
	// forgotten after about twice the learning time.
	// Allowed values are in range 1..1000. Device's default: 12 hours
	LearningTimeOffsetHours int16

	// LearningTimeGainHours is the time constant used to estimate the VOC
	// algorithm gain from measurement history, in hours. Past events will be
	// forgotten after about twice the learning time.
	// Allowed values are in range 1..1000. Device's default: 12 hours
	//
	// NOTE: This is only applicable to VOC. It has no impact for NOx. The datasheet
	// says that this is included in the NOx parameters to keep it consistent with
	// the VOC parameters. For NOx, it must always be set to 12.
	LearningTimeGainHours int16

	// GatingMaxDurationMinutes is the maximum duration of gating (freeze of
	// estimator during high VOC/NOx index signal), in minutes. Zero disables
	// gating. Allowed values are in range 0..3000. Device's default: 180 minutes
	// for VOC, 720 minutes for NOx
	GatingMaxDurationMinutes int16

	// InitialStdDevEstimate is the initial VOC standard deviation estimate.
	// A lower value boosts events during the initial learning period but may
	// result in larger device-to-device variation. Allowed values are in range
	// 10..5000. Device's default: 50
	//
	// NOTE: This is only applicable to VOC. It has no impact for NOx. The datasheet
	// says that this is included in the NOx parameters to keep it consistent with
	// the VOC parameters. For NOx, it must always be set to 50.
	InitialStdDevEstimate int16

	// GainFactor amplifies or attenuates the VOC/NOx index output.
	// Allowed values are in range 1..1000. Device's default: 230
	GainFactor int16
}

func (params VOCNOxAlgorithmTuningParameters) pack() []byte {
	return packWordsWithCRC([]uint16{
		uint16(params.IndexOffset),
		uint16(params.LearningTimeOffsetHours),
		uint16(params.LearningTimeGainHours),
		uint16(params.GatingMaxDurationMinutes),
		uint16(params.InitialStdDevEstimate),
		uint16(params.GainFactor),
	})
}

// GetVOCAlgorithmTuningParameters gets the parameters used to customize the
// VOC algorithm.
//
// For more information on the VOC index, see ["What is Sensirion's VOC Index?"].
// You may also consult "Sensirion's VOC and NOx Indices for Indoor Air Applications",
// available from Sensirion by request.
//
// ["What is Sensirion's VOC Index?"]: https://sensirion.com/media/documents/02232963/6294E043/Info_Note_VOC_Index.pdf
func (d *Dev) GetVOCAlgorithmTuningParameters() (*VOCNOxAlgorithmTuningParameters, error) {
	if !d.model.hasVOCNOx() {
		return nil, errors.New("sen6x: GetVOCAlgorithmTuningParameters requires a VOC-equipped model")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdGetVOCAlgorithmTuningParamsSEN65SEN66SEN68SEN69C, nil)
	if err != nil {
		return nil, err
	}

	return &VOCNOxAlgorithmTuningParameters{
		IndexOffset:              int16(binary.BigEndian.Uint16(data[0:2])),
		LearningTimeOffsetHours:  int16(binary.BigEndian.Uint16(data[2:4])),
		LearningTimeGainHours:    int16(binary.BigEndian.Uint16(data[4:6])),
		GatingMaxDurationMinutes: int16(binary.BigEndian.Uint16(data[6:8])),
		InitialStdDevEstimate:    int16(binary.BigEndian.Uint16(data[8:10])),
		GainFactor:               int16(binary.BigEndian.Uint16(data[10:12])),
	}, nil
}

// SetVOCAlgorithmTuningParameters sets the parameters used to customize the
// VOC algorithm.
//
// It has no effect if at least one parameter is outside the specified range.
//
// Note: This configuration is volatile, i.e. the parameters will be reverted
// to their default values after a device reset.
//
// For more information on the VOC index, see ["What is Sensirion's VOC Index?"].
// You may also consult "Sensirion's VOC and NOx Indices for Indoor Air Applications",
// available from Sensirion by request.
//
// ["What is Sensirion's VOC Index?"]: https://sensirion.com/media/documents/02232963/6294E043/Info_Note_VOC_Index.pdf
func (d *Dev) SetVOCAlgorithmTuningParameters(params VOCNOxAlgorithmTuningParameters) error {
	if !d.model.hasVOCNOx() {
		return errors.New("sen6x: SetVOCAlgorithmTuningParameters requires a VOC-equipped model")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdSetVOCAlgorithmTuningParamsSEN65SEN66SEN68SEN69C, params.pack())
}

// GetVOCAlgorithmState gets the current VOC algorithm state. This data can be
// used to restore the state with [Dev.SetVOCAlgorithmState] after a power cycle
// or device reset.
//
// This can be used either in measurement mode or in idle mode (which will then
// return the state at the time when the measurement was stopped). In measurement
// mode, the state can be read each measure interval to always have the latest
// state available.
func (d *Dev) GetVOCAlgorithmState() ([8]byte, error) {
	if !d.model.hasVOCNOx() {
		return [8]byte{}, errors.New("sen6x: GetVOCAlgorithmState requires a VOC-equipped model")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdGetVOCAlgorithmStateSEN65SEN66SEN68SEN69C, nil)
	if err != nil {
		return [8]byte{}, err
	}

	if len(data) != 8 {
		return [8]byte{}, fmt.Errorf("sen6x: expected VOC algorithm state to be 8 bytes, received %d", len(data))
	}

	return [8]byte(data), nil
}

// SetVOCAlgorithmState sets the VOC algorithm state previously received from
// [Dev.GetVOCAlgorithmState]. This command is only available in idle mode and
// the state will be applied when starting the next measurement. In measurement
// mode this command has no effect.
//
// Note: This configuration is volatile, i.e. the parameters will be reverted
// to their default values after a device reset.
func (d *Dev) SetVOCAlgorithmState(state [8]byte) error {
	if !d.model.hasVOCNOx() {
		return errors.New("sen6x: SetVOCAlgorithmState requires a VOC-equipped model")
	}

	packed, err := packBytesWithCRC(state[:])
	if err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdSetVOCAlgorithmStateSEN65SEN66SEN68SEN69C, packed)
}

// GetNOxAlgorithmTuningParameters gets the parameters used to customize the
// NOx algorithm.
//
// For more information on the NOx index, see ["What is Sensirion's NOx Index?"].
// You may also consult "Sensirion's VOC and NOx Indices for Indoor Air Applications",
// available from Sensirion by request.
//
// ["What is Sensirion's NOx Index?"]: https://sensirion.com/media/documents/9F289B95/6294DFFC/Info_Note_NOx_Index.pdf
func (d *Dev) GetNOxAlgorithmTuningParameters() (*VOCNOxAlgorithmTuningParameters, error) {
	if !d.model.hasVOCNOx() {
		return nil, errors.New("sen6x: GetNOxAlgorithmTuningParameters requires a NOx-equipped model")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.writeAndRead(cmdGetNOxAlgorithmTuningParamsSEN65SEN66SEN68SEN69C, nil)
	if err != nil {
		return nil, err
	}

	return &VOCNOxAlgorithmTuningParameters{
		IndexOffset:              int16(binary.BigEndian.Uint16(data[0:2])),
		LearningTimeOffsetHours:  int16(binary.BigEndian.Uint16(data[2:4])),
		LearningTimeGainHours:    int16(binary.BigEndian.Uint16(data[4:6])),
		GatingMaxDurationMinutes: int16(binary.BigEndian.Uint16(data[6:8])),
		InitialStdDevEstimate:    int16(binary.BigEndian.Uint16(data[8:10])),
		GainFactor:               int16(binary.BigEndian.Uint16(data[10:12])),
	}, nil
}

// SetNOxAlgorithmTuningParameters sets the parameters used to customize the
// NOx algorithm.
//
// It has no effect if at least one parameter is outside the specified range.
//
// Note: This configuration is volatile, i.e. the parameters will be reverted
// to their default values after a device reset.
//
// For more information on the NOx index, see ["What is Sensirion's NOx Index?"].
// You may also consult "Sensirion's VOC and NOx Indices for Indoor Air Applications",
// available from Sensirion by request.
//
// ["What is Sensirion's NOx Index?"]: https://sensirion.com/media/documents/9F289B95/6294DFFC/Info_Note_NOx_Index.pdf
func (d *Dev) SetNOxAlgorithmTuningParameters(params VOCNOxAlgorithmTuningParameters) error {
	if !d.model.hasVOCNOx() {
		return errors.New("sen6x: SetNOxAlgorithmTuningParameters requires a NOx-equipped model")
	}

	// These two parameters only apply to VOC but are included in the NOx parameters
	// for consistency. The datasheet specifies that these parameters must always
	// have the values set here.
	params.LearningTimeGainHours = 12
	params.InitialStdDevEstimate = 50

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.writeAndWait(cmdSetNOxAlgorithmTuningParamsSEN65SEN66SEN68SEN69C, params.pack())
}
