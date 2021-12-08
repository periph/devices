// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package videosink

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"log"
	"mime"
	"net/http"
	"net/textproto"
	"net/url"
	"sync"
	"time"
)

// bufferPool stores reusable []byte instances.
var bufferPool = sync.Pool{
	New: func() interface{} {
		return []byte(nil)
	},
}

// stopAndDrain ensures that the given timer is stopped and has no pending
// event.
func stopAndDrain(timer *time.Timer) {
	if timer != nil && !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

type imageConfig struct {
	format ImageFormat
}

func (d *Display) configFromQuery(values url.Values) (imageConfig, error) {
	cfg := imageConfig{
		format: d.defaultFormat,
	}

	if value := values.Get("format"); value != "" {
		if format, err := ImageFormatFromString(value); err != nil {
			return imageConfig{}, err
		} else {
			cfg.format = format
		}
	}

	return cfg, nil
}

type client struct {
	disp *Display

	refresh   chan struct{}
	terminate chan struct{}

	mostRecent time.Time
}

func newClient(d *Display) *client {
	return &client{
		disp:      d,
		refresh:   make(chan struct{}, 1),
		terminate: make(chan struct{}, 1),
	}
}

// Wait until the next frame should be sent, either because the keep-alive
// interval has passed or a change has been made to the buffer and the rate
// limit isn't violated.
func (c *client) waitNext(ctx context.Context) bool {
	earliestFrameAt := c.mostRecent.Add(c.disp.minFrameInterval)
	latestFrameAt := c.mostRecent.Add(c.disp.keepAliveInterval)

	var rateLimit <-chan time.Time
	keepAliveTimer := time.NewTimer(time.Until(latestFrameAt))

	defer stopAndDrain(keepAliveTimer)

	for {
		select {
		case <-rateLimit:
			return true

		case <-c.refresh:
			if remaining := time.Until(earliestFrameAt); remaining <= 0 {
				return true
			} else if rateLimit == nil {
				rateLimitTimer := time.NewTimer(remaining)
				defer stopAndDrain(rateLimitTimer)
				rateLimit = rateLimitTimer.C
			}

		case <-keepAliveTimer.C:
			return true

		case <-c.terminate:
			return false

		case <-ctx.Done():
			return false
		}
	}
}

func (d *Display) bufferChangedLocked() {
	for cfg, buffer := range d.snapshot {
		if buffer != nil {
			//lint:ignore SA6002 buffer is []byte and thus pointer-like
			bufferPool.Put(buffer)
		}

		delete(d.snapshot, cfg)
	}

	for c := range d.clients {
		select {
		case c.refresh <- struct{}{}:
		default:
		}
	}
}

func (d *Display) terminateClientsLocked() {
	for c := range d.clients {
		select {
		case c.terminate <- struct{}{}:
		default:
		}
	}
}

func (d *Display) encodeBufferLocked(format ImageFormat) ([]byte, error) {
	buf := bytes.NewBuffer(bufferPool.Get().([]byte)[:0])

	switch format {
	case PNG:
		if err := pngEncoder.get(d.pngCompressionLevel).Encode(buf, d.buffer); err != nil {
			return nil, err
		}

	case JPEG:
		if err := jpeg.Encode(buf, d.buffer, &d.jpegOptions); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unhandled image format %s", format)
	}

	return buf.Bytes(), nil
}

func (d *Display) grabSnapshot(cfg imageConfig) []byte {
	d.mu.Lock()
	defer d.mu.Unlock()

	encoded, ok := d.snapshot[cfg]
	if !ok {
		var err error

		encoded, err = d.encodeBufferLocked(cfg.format)
		if err != nil {
			panic(fmt.Sprintf("encoding image failed: %v", err))
		}
		d.snapshot[cfg] = encoded
	}

	return append(bufferPool.Get().([]byte)[:0], encoded...)
}

// ServeHTTP handles HTTP GET requests and sends a stream of images
// representing the display buffer in response. The display options control the
// default format and clients can explicitly request PNG or JPEG images using
// the "format" parameter ("?format=png", "?format=jpeg").
func (d *Display) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.Body.Close(); err != nil {
		log.Printf("Closing request body failed: %v", err)
	}

	if r.Method != http.MethodGet {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := d.configFromQuery(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pw := makePartWriter(w)

	w.Header().Set("Content-Type",
		mime.FormatMediaType("multipart/x-mixed-replace", map[string]string{
			"boundary": pw.boundary,
		}))

	c := newClient(d)

	d.mu.Lock()
	d.clients[c] = struct{}{}
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		delete(d.clients, c)
		d.mu.Unlock()
	}()

	partHeaders := make(textproto.MIMEHeader)
	partHeaders.Set("Content-Type", mime.FormatMediaType(cfg.format.mimeType(), nil))
	partHeaders.Set("Content-Transfer-Encoding", "binary")

	for {
		payload := d.grabSnapshot(cfg)
		err := pw.writeFrame(partHeaders, payload)

		if payload != nil {
			//lint:ignore SA6002 buffer is []byte and thus pointer-like
			bufferPool.Put(payload)
		}

		if err != nil {
			// Errors cause the request to be silently terminated. There's no
			// good way to deliver an error message to the client within an
			// image stream.
			return
		}

		c.mostRecent = time.Now()

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		if !c.waitNext(r.Context()) {
			return
		}
	}
}
