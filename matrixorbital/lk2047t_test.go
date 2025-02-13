// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package matrixorbital

import (
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/display/displaytest"
)

type mockReadWriterCloser struct {
	closed       bool
	bytesWritten int
	bytesRead    int
	readChars    string
	hash         hash.Hash32
	// TODO: Add CRC32 verification of written bytes...
}

func (mr *mockReadWriterCloser) Read(p []byte) (n int, err error) {
	if mr.closed {
		err = io.EOF
	}
	cPos := rand.Intn(len(mr.readChars))
	p[0] = byte(mr.readChars[cPos])
	n = 1
	mr.bytesRead += 1
	return
}

func (mr *mockReadWriterCloser) Write(p []byte) (n int, err error) {
	if mr.closed {
		err = io.EOF
		return
	}
	n = len(p)
	mr.hash.Write(p)
	mr.bytesWritten += n
	return
}

func (mr *mockReadWriterCloser) Close() error {
	mr.closed = true
	return nil
}

// Shutdown accepts an expected hash. If expectedHash doesn't match the hash of
// the stream written to the mock, it writes an error to t. If the test changes,
// or the implementation of the device changes, you'll have to update the
// expectedHash value in the Shutdown() call.
func (mr *mockReadWriterCloser) Shutdown(t *testing.T, expectedHash uint32) {
	finalHash := mr.hash.Sum32()
	if expectedHash == 0 {
		t.Logf("Mock Reader: Final Hash=0x%x", finalHash)
	} else if finalHash != expectedHash {
		t.Errorf("Incorrect hash. Expected: 0x%x, received 0x%x", expectedHash, finalHash)
	}
}

// getDisplay returns an LCD display constructed with a mock read/writer for
// testing.
func getDisplay() (*LK2047T, *mockReadWriterCloser) {

	wr := &mockReadWriterCloser{readChars: "ABCDEGH", hash: crc32.NewIEEE()}

	return NewWriterLK2047T(wr, 4, 20), wr
}

func TestTextDisplay(t *testing.T) {
	fmt.Println("beginning tests")
	dev, mock := getDisplay()
	defer mock.Shutdown(t, 0x4b8e39ef)
	wrTest := "abcdef"
	nWritten, err := dev.WriteString(wrTest)
	if err != nil {
		t.Error(err)
	}
	if nWritten != len(wrTest) || mock.bytesWritten != len(wrTest) {
		t.Errorf("write string error wrote %d bytes expected %d", mock.bytesWritten, len(wrTest))
	}
}

func TestInterface(t *testing.T) {
	dev, mock := getDisplay()
	defer mock.Shutdown(t, 0x1d42cd75)
	errors := displaytest.TestTextDisplay(dev, false)
	for _, err := range errors {
		if err != display.ErrNotImplemented {
			t.Error(err)
		}
	}
	if mock.bytesWritten == 27 {
		t.Error("27")
	}
}

func TestLEDs(t *testing.T) {
	dev, mock := getDisplay()
	defer mock.Shutdown(t, 0x3d9030ea)
	for ix := range 3 {
		for color := Off; color <= Yellow; color++ {
			err := dev.LED(ix, color)
			if err != nil {
				t.Error(err)
			}
		}
	}
}

// Peform a basic test on optional interface methods.
func TestContrastBacklight(t *testing.T) {
	dev, mock := getDisplay()
	defer mock.Shutdown(t, 0x93097842)
	if err := dev.Contrast(0); err != nil {
		t.Error(err)
	}
	if err := dev.Contrast(100); err != nil {
		t.Error(err)
	}
	if err := dev.Backlight(0); err != nil {
		t.Error(err)
	}
	if err := dev.Backlight(50); err != nil {
		t.Error(err)
	}
	if err := dev.KeypadBacklight(true); err != nil {
		t.Error(err)
	}
	if err := dev.KeypadBacklight(false); err != nil {
		t.Error(err)
	}
}

func TestPins(t *testing.T) {
	dev, _ := getDisplay()
	for ix, pin := range dev.Pins {
		if len(pin.String()) == 0 {
			t.Errorf("pin %d return empty string", ix)
		}
		if pin.Number() != ix+1 {
			t.Errorf("Pin %d, unexpected pin # %d", ix, pin.Number())
		}
		if len(pin.Name()) == 0 {
			t.Errorf("pin %d returned empty name!", ix)
		}
	}
}

func TestKeypad(t *testing.T) {
	dev, _ := getDisplay()
	ch, err := dev.ReadKeypad()
	if err != nil {
		t.Fatal(err)
	}
	if ch == nil {
		t.Fatal("ReadKeypad() returned nil channel!")
	}
	received := atomic.Int32{}
	expected := int32(10)
	go func() {
		for {
			if received.Load() >= expected {

				if err := dev.Halt(); err != nil {
					t.Error(err)
				}
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
	for c := range ch {
		t.Logf("received %s", string(c))
		received.Add(1)
		time.Sleep(5 * time.Millisecond)
	}
}

var _ io.Reader = &mockReadWriterCloser{}
var _ io.Writer = &mockReadWriterCloser{}
var _ io.Closer = &mockReadWriterCloser{}
