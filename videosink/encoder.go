// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package videosink

import (
	"image/png"
	"sync"
)

type pngEncoderBufferPool sync.Pool

func (p *pngEncoderBufferPool) Get() *png.EncoderBuffer {
	buf, _ := (*sync.Pool)(p).Get().(*png.EncoderBuffer)
	return buf
}

func (p *pngEncoderBufferPool) Put(buf *png.EncoderBuffer) {
	(*sync.Pool)(p).Put(buf)
}

type pngEncoderManager struct {
	mu   sync.Mutex
	pool pngEncoderBufferPool
	enc  map[png.CompressionLevel]*png.Encoder
}

var pngEncoder pngEncoderManager

// get returns a PNG encoder with a globally shared buffer pool.
func (m *pngEncoderManager) get(level png.CompressionLevel) *png.Encoder {
	m.mu.Lock()
	defer m.mu.Unlock()

	enc := m.enc[level]
	if enc == nil {
		if m.enc == nil {
			// The vast majority of use cases will involve exactly one
			// compression level.
			m.enc = make(map[png.CompressionLevel]*png.Encoder, 1)
		}

		enc = &png.Encoder{
			CompressionLevel: level,
			BufferPool:       &m.pool,
		}

		m.enc[level] = enc
	}

	return enc
}
