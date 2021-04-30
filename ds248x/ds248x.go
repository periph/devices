// Copyright 2016 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package ds248x

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"bytes"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/onewire"
)

// PupOhm controls the strength of the passive pull-up resistor
// on the 1-wire data line. The default value is 1000Ω.
type PupOhm uint8

const (
	// R500Ω passive pull-up resistor.
	R500Ω = 4
	// R1000Ω passive pull-up resistor.
	R1000Ω = 6
)

// Opts contains options to pass to the constructor.
type Opts struct {
	PassivePullup bool // false:use active pull-up, true: disable active pullup

	// The following options are only available on the ds2483 (not ds2482-100).
	// The actual value used is the closest possible value (rounded up or down).
	ResetLow       time.Duration // reset low time, range 440μs..740μs
	PresenceDetect time.Duration // presence detect sample time, range 58μs..76μs
	Write0Low      time.Duration // write zero low time, range 52μs..70μs
	Write0Recovery time.Duration // write zero recovery time, range 2750ns..25250ns
	PullupRes      PupOhm        // passive pull-up resistance, true: 500Ω, false: 1kΩ
}

// DefaultOpts is the recommended default options.
var DefaultOpts = Opts{
	PassivePullup:  false,
	ResetLow:       560 * time.Microsecond,
	PresenceDetect: 68 * time.Microsecond,
	Write0Low:      64 * time.Microsecond,
	Write0Recovery: 5250 * time.Nanosecond,
	PullupRes:      R1000Ω,
}

// New returns a device object that communicates over I²C to the DS2482/DS2483
// controller.
//
// This device object implements onewire.Bus and can be used to
// access devices on the bus.
//
// Valid I²C addresses are 0x18, 0x19, 0x20 and 0x21.
func New(i i2c.Bus, addr uint16, opts *Opts) (*Dev, error) {
	switch addr {
	case 0x18, 0x19, 0x20, 0x21:
	default:
		return nil, errors.New("ds248x: given address not supported by device")
	}
	d := &Dev{i2c: &i2c.Dev{Bus: i, Addr: addr}}
	if err := d.makeDev(opts); err != nil {
		return nil, err
	}
	return d, nil
}

// Dev is a handle to a ds248x device and it implements the onewire.Bus
// interface.
//
// Dev implements a persistent error model: if a fatal error is encountered it
// places itself into an error state and immediately returns the last error on
// all subsequent calls. A fresh Dev, which reinitializes the hardware, must be
// created to proceed.
//
// A persistent error is only set when there is a problem with the ds248x
// device itself (or the I²C bus used to access it). Errors on the 1-wire bus
// do not cause persistent errors and implement the onewire.BusError interface
// to indicate this fact.
type Dev struct {
	sync.Mutex               // lock for the bus while a transaction is in progress
	i2c        conn.Conn     // i2c device handle for the ds248x
	isDS248x   int           // 0: ds2482-100 1: ds2482-800 2: ds2483,
	confReg    byte          // value written to configuration register
	tReset     time.Duration // time to perform a 1-wire reset
	tSlot      time.Duration // time to perform a 1-bit 1-wire read/write
	err        error         // persistent error, device will no longer operate
}

func (d *Dev) String() string {
	switch d.isDS248x {
	case isDS2482x100:
		return fmt.Sprintf("DS2482-100{%s}", d.i2c)
	case isDS2482x800:
		return fmt.Sprintf("DS2482-800{%s}", d.i2c)
	case isDS2483:
		return fmt.Sprintf("DS2483{%s}", d.i2c)
	default:
		return fmt.Sprintf("Undefined{%s}", d.i2c)
	}
}

// Halt implements conn.Resource.
func (d *Dev) Halt() error {
	return nil
}

// Tx performs a bus transaction, sending and receiving bytes, and ending by
// pulling the bus high either weakly or strongly depending on the value of
// power.
//
// A strong pull-up is typically required to power temperature conversion or
// EEPROM writes.
func (d *Dev) Tx(w, r []byte, power onewire.Pullup) error {
	d.Lock()
	defer d.Unlock()

	// Issue 1-wire bus reset.
	if present, err := d.reset(); err != nil {
		return err
	} else if !present {
		return busError("ds248x: no device present")
	}

	// Send bytes onto 1-wire bus.
	for i, b := range w {
		if power == onewire.StrongPullup && i == len(w)-1 && len(r) == 0 {
			// This is the last byte, need to activate strong pull-up.
			d.i2cTx([]byte{cmdWriteConfig, d.confReg&0xbf | 0x4}, nil)
		}
		d.i2cTx([]byte{cmd1WWrite, b}, nil)
		d.waitIdle(7 * d.tSlot)
	}

	// Read bytes from one-wire bus.
	for i := range r {
		if power == onewire.StrongPullup && i == len(r)-1 {
			// This is the last byte, need to activate strong-pull-up
			d.i2cTx([]byte{cmdWriteConfig, d.confReg&0xbf | 0x4}, nil)
		}
		d.i2cTx([]byte{cmd1WRead}, r[i:i+1])
		d.waitIdle(7 * d.tSlot)
		d.i2cTx([]byte{cmdSetReadPtr, regRDR}, r[i:i+1])
	}

	return d.err
}

// Search performs a "search" cycle on the 1-wire bus and returns the addresses
// of all devices on the bus if alarmOnly is false and of all devices in alarm
// state if alarmOnly is true.
//
// If an error occurs during the search the already-discovered devices are
// returned with the error.
func (d *Dev) Search(alarmOnly bool) ([]onewire.Address, error) {
	return onewire.Search(d, alarmOnly)
}

// ChannelSelect function is for selecting one of eight 1-w channels on DS2482-800. 
// On other chips it does nothing. Function silently limits channel selection between
// 0 and 7. It is expected that application keeps track of
// with 1-w device is connected to with channel.
//
// Communication error is returned if present.
//
func (d *Dev) ChannelSelect(ch int) (err error) {
	switch d.isDS248x {
	case isDS2482x100:

	case isDS2482x800:
		if ch < 0 {
			ch = 0
		}
		if ch > 7 {
			ch = 7
		}
		csc := []byte{cscIO0w, cscIO1w, cscIO2w, cscIO3w, cscIO4w, cscIO5w, cscIO6w, cscIO7w}
		buf := []byte{cmdChannelSelect, csc[ch]}
		if err = d.i2c.Tx(buf, nil); err != nil {
			return fmt.Errorf("ds2482-800: error while selecting channel: %s", err)
		}
	case isDS2483:

	default:

	}
	return
}

// SelectedChannel function is to read with 1-w channel selected on DS2482-800. 
// On other chips it always returns 0. It is expected that application keeps track of
// with 1-w device is connected to with channel.
//
// On error returns 255.
//
func (d *Dev) SelectedChannel() (ch int) {
	ch = 0
	switch d.isDS248x {
	case isDS2482x100:

	case isDS2482x800:
		var sch [1]byte
		if err := d.i2c.Tx([]byte{cmdSetReadPtr, regCSR}, sch[:]); err != nil {
			ch = 255
			return 
		}
		csc := []byte{cscIO0r, cscIO1r, cscIO2r, cscIO3r, cscIO4r, cscIO5r, cscIO6r, cscIO7r}
		ch = bytes.Index(csc, []byte{sch[0]})
		if ch < 0 {
			ch = 255
		}
		if ch > 7 {
			ch = 255
		}
	case isDS2483:

	default:

	}
	return
}

// SearchTriplet performs a single bit search triplet command on the bus, waits
// for it to complete and returs the outcome.
//
// SearchTriplet should not be used directly, use Search instead.
func (d *Dev) SearchTriplet(direction byte) (onewire.TripletResult, error) {
	// Send one-wire triplet command.
	var dir byte
	if direction != 0 {
		dir = 0x80
	}
	d.i2cTx([]byte{cmd1WTriplet, dir}, nil)
	// Wait and read status register, concoct result from there.
	status := d.waitIdle(0 * d.tSlot) // in theory 3*tSlot but it's actually overlapped
	tr := onewire.TripletResult{
		GotZero: status&0x20 == 0,
		GotOne:  status&0x40 == 0,
		Taken:   status >> 7,
	}
	return tr, d.err
}

//

// reset issues a reset signal on the 1-wire bus and returns true if any device
// responded with a presence pulse.
func (d *Dev) reset() (bool, error) {
	// Issue reset.
	d.i2cTx([]byte{cmd1WReset}, nil)

	// Wait for reset to complete.
	status := d.waitIdle(d.tReset)
	if d.err != nil {
		return false, d.err
	}
	// Detect bus short and turn into 1-wire error
	if (status & 4) != 0 {
		return false, shortedBusError("onewire/ds248x: bus has a short")
	}
	return (status & 2) != 0, nil
}

// i2cTx is a helper function to call i2c.Tx and handle the error by persisting
// it.
func (d *Dev) i2cTx(w, r []byte) {
	if d.err != nil {
		return
	}
	d.err = d.i2c.Tx(w, r)
}

// waitIdle waits for the one wire bus to be idle.
//
// It initially sleeps for the delay and then polls the status register and
// sleeps for a tenth of the delay each time the status register indicates that
// the bus is still busy. The last read status byte is returned.
//
// An overall timeout of 3ms is applied to the whole procedure. waitIdle uses
// the persistent error model and returns 0 if there is an error.
func (d *Dev) waitIdle(delay time.Duration) byte {
	if d.err != nil {
		return 0
	}
	// Overall timeout.
	tOut := time.Now().Add(3 * time.Millisecond)
	sleep(delay)
	for {
		// Read status register.
		var status [1]byte
		d.i2cTx(nil, status[:])
		// If bus idle complete, return status. This also returns if d.err!=nil
		// because in that case status[0]==0.
		if (status[0] & 1) == 0 {
			return status[0]
		}
		// If we're timing out return error. This is an error with the ds248x, not with
		// devices on the 1-wire bus, hence it is persistent.
		if time.Now().After(tOut) {
			d.err = fmt.Errorf("ds248x: timeout waiting for bus cycle to finish")
			return 0
		}
		// Try not to hog the kernel thread.
		sleep(delay / 10)
	}
}

func (d *Dev) makeDev(opts *Opts) error {
	d.tReset = 2 * opts.ResetLow
	d.tSlot = opts.Write0Low + opts.Write0Recovery

	// Issue a reset command.
	if err := d.i2c.Tx([]byte{cmdReset}, nil); err != nil {
		return fmt.Errorf("ds248x: error while resetting: %s", err)
	}

	// Read the status register to confirm that we have a responding ds248x
	var stat [1]byte
	if err := d.i2c.Tx([]byte{cmdSetReadPtr, regStatus}, stat[:]); err != nil {
		return fmt.Errorf("ds248x: error while reading status register: %s", err)
	}
	if stat[0] != 0x18 {
		return fmt.Errorf("ds248x: invalid status register value: %#x, expected 0x18", stat[0])
	}

	// Write the device configuration register to get the chip out of reset state, immediately
	// read it back to get confirmation.
	d.confReg = 0xe1 // standard-speed, no strong pullup, no powerdown, active pull-up
	if opts.PassivePullup {
		d.confReg ^= 0x11
	}
	var dcr [1]byte
	if err := d.i2c.Tx([]byte{cmdWriteConfig, d.confReg}, dcr[:]); err != nil {
		return fmt.Errorf("ds248x: error while writing device config register: %s", err)
	}
	// When reading back we only get the bottom nibble
	if dcr[0] != d.confReg&0x0f {
		return fmt.Errorf("ds248x: failure to write device config register, wrote %#x got %#x back",
			d.confReg, dcr[0])
	}

	// Set the read ptr to the port configuration register to determine whether we have a
	// ds2483 vs ds2482-100. This will fail on devices that do not have a port config
	// register, such as the ds2482-100.
	if d.i2c.Tx([]byte{cmdSetReadPtr, regPCR}, nil) == nil {
		d.isDS248x = isDS2483
		buf := []byte{cmdAdjPort,
			byte(0x00 + ((opts.ResetLow/time.Microsecond - 430) / 20 & 0x0f)),
			byte(0x20 + ((opts.PresenceDetect/time.Microsecond - 55) / 2 & 0x0f)),
			byte(0x40 + ((opts.Write0Low/time.Microsecond - 51) / 2 & 0x0f)),
			byte(0x60 + (((opts.Write0Recovery-1250)/2500 + 5) & 0x0f)),
			byte(0x80 + (opts.PullupRes & 0x0f)),
		}
		if err := d.i2c.Tx(buf, nil); err != nil {
			return fmt.Errorf("ds248x: error while setting port config values: %s", err)
		}

	} else {
		if d.i2c.Tx([]byte{cmdSetReadPtr, regCSR}, nil) == nil {
			d.isDS248x = isDS2482x800
			buf := []byte{cmdChannelSelect, cscIO0w}
			if err := d.i2c.Tx(buf, nil); err != nil {
				return fmt.Errorf("ds2482-800: error while selecting channel: %s", err)
			}
		} else {
			d.isDS248x = isDS2482x100
		}
	}
	return nil
}

// shortedBusError implements error and onewire.ShortedBusError.
type shortedBusError string

func (e shortedBusError) Error() string   { return string(e) }
func (e shortedBusError) IsShorted() bool { return true }
func (e shortedBusError) BusError() bool  { return true }

// busError implements error and onewire.BusError.
type busError string

func (e busError) Error() string  { return string(e) }
func (e busError) BusError() bool { return true }

var sleep = time.Sleep

var _ conn.Resource = &Dev{}

const (
	cmdReset         = 0xf0 // reset ds248x
	cmdSetReadPtr    = 0xe1 // set the read pointer
	cmdWriteConfig   = 0xd2 // write the device configuration
	cmdAdjPort       = 0xc3 // adjust 1-wire port (ds2483)
	cmdChannelSelect = 0xc3 // channel select (ds2482-800)
	cmd1WReset       = 0xb4 // reset the 1-wire bus
	cmd1WBit         = 0x87 // perform a single-bit transaction on the 1-wire bus
	cmd1WWrite       = 0xa5 // perform a byte write on the 1-wire bus
	cmd1WRead        = 0x96 // perform a byte read on the 1-wire bus
	cmd1WTriplet     = 0x78 // perform a triplet operation (2 bit reads, a bit write)

	regDCR    = 0xc3 // read ptr for device configuration register
	regStatus = 0xf0 // read ptr for status register
	regRDR    = 0xe1 // read ptr for read-data register
	regPCR    = 0xb4 // read ptr for port configuration register
	regCSR    = 0xd2 // read ptr for channel selection register

	// ds2482-800 channel selection codes to be written and read back
	cscIO0w = 0xF0 // channel 0 writing
	cscIO0r = 0xB8 // channel 0 reading
	cscIO1w = 0xE1 // channel 1 writing
	cscIO1r = 0xB1 // channel 1 reading
	cscIO2w = 0xD2 // channel 2 writing
	cscIO2r = 0xAA // channel 2 reading
	cscIO3w = 0xC3 // channel 3 writing
	cscIO3r = 0xA3 // channel 3 reading
	cscIO4w = 0xB4 // channel 4 writing
	cscIO4r = 0x9C // channel 4 reading
	cscIO5w = 0xA5 // channel 5 writing
	cscIO5r = 0x95 // channel 5 reading
	cscIO6w = 0x96 // channel 6 writing
	cscIO6r = 0x8E // channel 6 reading
	cscIO7w = 0x87 // channel 7 writing
	cscIO7r = 0x87 // channel 7 reading

	isDS2482x100 = 0 // DS2482-100 selected
	isDS2482x800 = 1 // DS2482-800 selected
	isDS2483     = 2 // DS2483 selected
)


