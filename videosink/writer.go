// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package videosink

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/textproto"
	"strconv"
)

// randomBoundary generates a MIME multipart boundary compatible with RFC 2046
// (section 5.1.1).
func randomBoundary() string {
	var buf [34]byte
	if _, err := io.ReadFull(rand.Reader, buf[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}

type partWriter struct {
	u        io.Writer
	boundary string
	started  bool
}

func makePartWriter(u io.Writer) partWriter {
	return partWriter{
		u:        u,
		boundary: randomBoundary(),
	}
}

// writeFrame sends a single part of a MIME multipart entity, ensuring it's
// fully written by the time the function returns.
//
// The caller-owned headers are modified to set a Content-Length header.
//
// Go has a writer for MIME multipart messages in "mime/multipart".Writer. As
// of Go 1.17 it's not suitable for writing a neverending stream of parts where
// each must be flushed to the client with the part-ending boundary line.
func (w *partWriter) writeFrame(header textproto.MIMEHeader, body []byte) error {
	header.Set("Content-Length", strconv.FormatInt(int64(len(body)), 10))

	var buf bytes.Buffer

	if !w.started {
		fmt.Fprintf(&buf, "--%s\r\n", w.boundary)
		w.started = true
	}

	for name := range header {
		for _, value := range header[name] {
			fmt.Fprintf(&buf, "%s: %s\r\n", name, value)
		}
	}

	buf.WriteString("\r\n")

	_, err := buf.WriteTo(w.u)
	if err == nil {
		_, err = io.Copy(w.u, bytes.NewReader(body))
		if err == nil {
			_, err = fmt.Fprintf(w.u, "\r\n--%s\r\n", w.boundary)
		}
	}

	return err
}
