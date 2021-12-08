// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package videosink

import (
	"image/jpeg"
	"image/png"
	"sync"
)

var jpegOptions = jpeg.Options{
	Quality: 95,
}

type pngEncoderBufferPool sync.Pool

func (p *pngEncoderBufferPool) Get() *png.EncoderBuffer {
	buf, _ := (*sync.Pool)(p).Get().(*png.EncoderBuffer)
	return buf
}

func (p *pngEncoderBufferPool) Put(buf *png.EncoderBuffer) {
	(*sync.Pool)(p).Put(buf)
}

type pngEncoderManager struct {
	once sync.Once
	pool pngEncoderBufferPool
	enc  *png.Encoder
}

var pngEncoder pngEncoderManager

// get returns a PNG encoder with a globally shared buffer pool.
func (m *pngEncoderManager) get() *png.Encoder {
	m.once.Do(func() {
		m.enc = &png.Encoder{
			CompressionLevel: png.BestSpeed,
			BufferPool:       &m.pool,
		}
	})

	return m.enc
}
