// Copyright 2016 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package lirc

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/ir"
)

// New returns a IR receiver / emitter handle.
func New() (*Conn, error) {
	w, err := net.Dial("unix", "/var/run/lirc/lircd")
	if err != nil {
		return nil, err
	}
	c := &Conn{w: w, c: make(chan ir.Message), list: map[string][]string{}}
	// Unconditionally retrieve the list of all known keys at start.
	if _, err := w.Write([]byte("LIST\n")); err != nil {
		_ = w.Close()
		return nil, err
	}
	go c.loop(bufio.NewReader(w))
	return c, nil
}

// Conn is an open port to lirc.
type Conn struct {
	w net.Conn
	c chan ir.Message

	mu          sync.Mutex
	list        map[string][]string // list of remotes and associated keys
	pendingList map[string][]string // list of remotes and associated keys being created.
}

// String implements conn.Resource.
func (c *Conn) String() string {
	return "lirc"
}

// Halt implements conn.Resource.
//
// It has no effect.
func (c *Conn) Halt() error {
	return nil
}

// Close closes the socket to lirc. It is not a requirement to close before
// process termination.
func (c *Conn) Close() error {
	return c.w.Close()
}

// Emit implements ir.IR.
func (c *Conn) Emit(remote string, key ir.Key) error {
	// http://www.lirc.org/html/lircd.html#lbAH
	_, err := fmt.Fprintf(c.w, "SEND_ONCE %s %s\n", remote, key)
	return err
}

// Channel implements ir.IR.
func (c *Conn) Channel() <-chan ir.Message {
	return c.c
}

// Codes returns all the known codes.
//
// Empty if the list was not retrieved yet.
func (c *Conn) Codes() map[string][]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.list
}

//

func (c *Conn) loop(r *bufio.Reader) {
	defer func() {
		close(c.c)
		c.c = nil
	}()
	for {
		line, err := read(r)
		if line == "BEGIN" {
			err = c.readData(r)
		} else if len(line) != 0 {
			// Format is: <code> <repeat count> <button name> <remote control name>
			// http://www.lirc.org/html/lircd.html#lbAG
			if parts := strings.SplitN(line, " ", 5); len(parts) != 4 {
				log.Printf("ir: corrupted line: %v", line)
			} else {
				if i, err2 := strconv.Atoi(parts[1]); err2 != nil {
					log.Printf("ir: corrupted line: %v", line)
				} else if len(parts[2]) != 0 && len(parts[3]) != 0 {
					c.c <- ir.Message{Key: ir.Key(parts[2]), RemoteType: parts[3], Repeat: i != 0}
				}
			}
		}
		if err != nil {
			break
		}
	}
}

func (c *Conn) readData(r *bufio.Reader) error {
	// Format is:
	// BEGIN
	// <original command>
	// SUCCESS
	// DATA
	// <number of entries 1 based>
	// <entries>
	// ...
	// END
	cmd, err := read(r)
	if err != nil {
		return err
	}
	switch cmd {
	case "SIGHUP":
		_, err = c.w.Write([]byte("LIST\n"))
		if err != nil {
			return err
		}
	default:
		// In case of any error, ignore the rest.
		line := ""
		if line, err = read(r); err != nil {
			return err
		}
		if line != "SUCCESS" {
			log.Printf("ir: unexpected line: %v, expected SUCCESS", line)
			return nil
		}
		if line, err = read(r); err != nil {
			return err
		}
		if line != "DATA" {
			log.Printf("ir: unexpected line: %v, expected DATA", line)
			return nil
		}
		if line, err = read(r); err != nil {
			return err
		}
		nbLines := 0
		if nbLines, err = strconv.Atoi(line); err != nil {
			return err
		}
		list := make([]string, nbLines)
		for i := 0; i < nbLines; i++ {
			if list[i], err = read(r); err != nil {
				return err
			}
		}
		switch {
		case cmd == "LIST":
			// Request the codes for each remote.
			c.pendingList = map[string][]string{}
			for _, l := range list {
				if _, ok := c.pendingList[l]; ok {
					log.Printf("ir: unexpected %s", cmd)
				} else {
					c.pendingList[l] = []string{}
					if _, err = fmt.Fprintf(c.w, "LIST %s\n", l); err != nil {
						return err
					}
				}
			}
		case strings.HasPrefix(line, "LIST "):
			if c.pendingList == nil {
				log.Printf("ir: unexpected %s", cmd)
			} else {
				remote := cmd[5:]
				c.pendingList[remote] = list
				all := true
				for _, v := range c.pendingList {
					if len(v) == 0 {
						all = false
						break
					}
				}
				if all {
					c.mu.Lock()
					c.list = c.pendingList
					c.pendingList = nil
					c.mu.Unlock()
				}
			}
		default:
		}
	}
	line, err := read(r)
	if err != nil {
		return err
	}
	if line != "END" {
		log.Printf("ir: unexpected line: %v, expected END", line)
	}
	return nil
}

func read(r *bufio.Reader) (string, error) {
	raw, err := r.ReadBytes('\n')
	if err != nil {
		return "", err
	}
	if len(raw) != 0 {
		raw = raw[:len(raw)-1]
	}
	return string(raw), nil
}

var _ ir.Conn = &Conn{}
var _ conn.Resource = &Conn{}
