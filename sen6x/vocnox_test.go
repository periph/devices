// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sen6x

import (
	"testing"
)

func TestDevGetVOCAlgorithmTuningParameters(t *testing.T) {
	cmd := []byte{0x60, 0xd0}

	cases := []writeAndReadTestCase[*VOCNOxAlgorithmTuningParameters]{
		{
			name:  "all values set",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x00, 0x64, 0xfe, // Index offset
				0x00, 0x0c, 0xfc, // Learning time offset
				0x00, 0x0c, 0xfc, // Learning time gain
				0x00, 0xb4, 0xfa, // Gating max duration
				0x00, 0x32, 0x26, // Initial std dev estimate
				0x00, 0xe6, 0xe6, // Gain factor
			},
			want: &VOCNOxAlgorithmTuningParameters{
				IndexOffset:              int16(100),
				LearningTimeOffsetHours:  int16(12),
				LearningTimeGainHours:    int16(12),
				GatingMaxDurationMinutes: int16(180),
				InitialStdDevEstimate:    int16(50),
				GainFactor:               int16(230),
			},
		},
		{
			name:  "bad crc",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x00, 0x64, 0xfe, // Index offset
				0x00, 0x0c, 0xfc, // Learning time offset
				0x00, 0x0c, 0xfc, // Learning time gain
				0x00, 0xb4, 0xfa, // Gating max duration
				0x00, 0x32, 0x26, // Initial std dev estimate
				0x00, 0xe6, 0xff, // Gain factor with incorrect CRC (should be 0xe6)
			},
			wantErr: true,
		},
		{
			// This fails before sending any data, so no tx or rx set.
			name:    "model without VOC/NOx capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).GetVOCAlgorithmTuningParameters)
}

func TestDevSetVOCAlgorithmTuningParameters(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx: []byte{
				0x60, 0xd0, // Command
				0x00, 0x64, 0xfe, // Index offset
				0x00, 0x0c, 0xfc, // Learning time offset
				0x00, 0x0c, 0xfc, // Learning time gain
				0x00, 0xb4, 0xfa, // Gating max duration
				0x00, 0x32, 0x26, // Initial std dev estimate
				0x00, 0xe6, 0xe6, // Gain factor
			},
		},
		{
			name:    "model without VOC/NOx capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	params := VOCNOxAlgorithmTuningParameters{
		IndexOffset:              int16(100),
		LearningTimeOffsetHours:  int16(12),
		LearningTimeGainHours:    int16(12),
		GatingMaxDurationMinutes: int16(180),
		InitialStdDevEstimate:    int16(50),
		GainFactor:               int16(230),
	}

	runWriteTests(t, cases, func(d *Dev) error {
		return d.SetVOCAlgorithmTuningParameters(params)
	})
}

func TestDevGetVOCAlgorithmState(t *testing.T) {
	cmd := []byte{0x61, 0x81}

	cases := []writeAndReadTestCase[[8]byte]{
		{
			name:  "success",
			model: SEN66,
			tx:    cmd,
			rx:    []byte{0x08, 0x43, 0xd8, 0x8b, 0x2b, 0xd4, 0x0f, 0x34, 0x19, 0xa7, 0x72, 0x4a},
			want:  [8]byte{0x08, 0x43, 0x8b, 0x2b, 0x0f, 0x34, 0xa7, 0x72},
		},
		{
			// writeAndRead will fail because no response is set.
			name:      "read error",
			model:     SEN66,
			tx:        cmd,
			dontPanic: true,
			wantErr:   true,
		},
		{
			name:    "model without VOC/NOx capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).GetVOCAlgorithmState)
}

func TestDevSetVOCAlgorithmState(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx: []byte{
				0x61, 0x81, // Command
				0x08, 0x43, 0xd8, 0x8b, 0x2b, 0xd4, 0x0f, 0x34, 0x19, 0xa7, 0x72, 0x4a, // VOC alg state
			},
		},
		{
			name:    "model without VOC/NOx capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteTests(t, cases, func(d *Dev) error {
		return d.SetVOCAlgorithmState([8]byte{0x08, 0x43, 0x8b, 0x2b, 0x0f, 0x34, 0xa7, 0x72})
	})
}

func TestDevGetNOxAlgorithmTuningParameters(t *testing.T) {
	cmd := []byte{0x60, 0xe1}

	cases := []writeAndReadTestCase[*VOCNOxAlgorithmTuningParameters]{
		{
			name:  "all values set",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x00, 0x01, 0xb0, // Index offset
				0x00, 0x0c, 0xfc, // Learning time offset
				0x00, 0x0c, 0xfc, // Learning time gain
				0x02, 0xd0, 0x5c, // Gating max duration
				0x00, 0x32, 0x26, // Initial std dev estimate
				0x00, 0xe6, 0xe6, // Gain factor
			},
			want: &VOCNOxAlgorithmTuningParameters{
				IndexOffset:              int16(1),
				LearningTimeOffsetHours:  int16(12),
				LearningTimeGainHours:    int16(12),
				GatingMaxDurationMinutes: int16(720),
				InitialStdDevEstimate:    int16(50),
				GainFactor:               int16(230),
			},
		},
		{
			name:  "bad crc",
			model: SEN66,
			tx:    cmd,
			rx: []byte{
				0x00, 0x01, 0xb0, // Index offset
				0x00, 0x0c, 0xfc, // Learning time offset
				0x00, 0x0c, 0xfc, // Learning time gain
				0x02, 0xd0, 0x5c, // Gating max duration
				0x00, 0x32, 0x26, // Initial std dev estimate
				0x00, 0xe6, 0xff, // Gain factor with incorrect CRC (should be 0xe6)
			},
			wantErr: true,
		},
		{
			// This fails before sending a command, so no tx or rx set.
			name:    "model without VOC/NOx capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	runWriteAndReadTests(t, cases, (*Dev).GetNOxAlgorithmTuningParameters)
}

func TestDevSetNOxAlgorithmTuningParameters(t *testing.T) {
	cases := []writeTestCase{
		{
			name:  "success",
			model: SEN66,
			tx: []byte{
				0x60, 0xe1, // Command
				0x00, 0x01, 0xb0, // Index offset
				0x00, 0x0c, 0xfc, // Learning time offset
				0x00, 0x0c, 0xfc, // Learning time gain
				0x02, 0xd0, 0x5c, // Gating max duration
				0x00, 0x32, 0x26, // Initial std dev estimate
				0x00, 0xe6, 0xe6, // Gain factor
			},
		},
		{
			name:    "model without VOC/NOx capability",
			model:   SEN62,
			wantErr: true,
		},
	}

	params := VOCNOxAlgorithmTuningParameters{
		IndexOffset:              int16(1),
		LearningTimeOffsetHours:  int16(12),
		LearningTimeGainHours:    int16(12),
		GatingMaxDurationMinutes: int16(720),
		InitialStdDevEstimate:    int16(50),
		GainFactor:               int16(230),
	}

	runWriteTests(t, cases, func(d *Dev) error {
		return d.SetNOxAlgorithmTuningParameters(params)
	})
}
