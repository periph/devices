package firmatareg

import (
	"errors"
	"strconv"
	"strings"
	"sync"

	"periph.io/x/devices/v3/firmata"
)

// Opener opens a handle to a firmata device.
//
// It is provided by the actual device driver.
type Opener func() (firmata.ClientI, error)

// Ref references a Firmata device.
//
// It is returned by All() to enumerate all registered devices.
type Ref struct {
	// Name of the device.
	//
	// It must be unique across the host.
	Name string
	// Aliases are the alternative names that can be used to reference this bus.
	Aliases []string
	// Open is the factory to open a handle to this Firmata device.
	Open Opener
}

// Open opens a firmata device by its name, an alias or its file path and returns
// a handle to it.
//
// Specify the empty string "" to get the first available device. This is the
// recommended default value unless an application knows the exact device to use.
//
// Each device can register multiple aliases, each leading to the same device handle.
//
// "file path" is a generic concept that is highly dependent on the platform
// and OS. Depending on the OS, a serial port can be `/dev/tty*` or `COM*`.
func Open(name string) (firmata.ClientI, error) {
	var r *Ref
	var err error
	func() {
		mu.Lock()
		defer mu.Unlock()
		if len(byName) == 0 {
			err = errors.New("firmatareg: no device found; did you forget to call Init()")
			return
		}
		if len(name) == 0 {
			r = getDefault()
			return
		}
		// Try by name, by alias, by number.
		if r = byName[name]; r == nil {
			r = byAlias[name]
		}
	}()
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("firmatareg: can't open unknown device: " + strconv.Quote(name))
	}
	return r.Open()
}

// All returns a copy of all the registered references to all know firmata devices
// available on this host.
//
// The list is sorted by the bus name.
func All() []*Ref {
	mu.Lock()
	defer mu.Unlock()
	out := make([]*Ref, 0, len(byName))
	for _, v := range byName {
		r := &Ref{Name: v.Name, Aliases: make([]string, len(v.Aliases)), Open: v.Open}
		copy(r.Aliases, v.Aliases)
		out = insertRef(out, r)
	}
	return out
}

// Register registers a firmata device.
//
// Registering the same device name twice is an error, e.g. o.Name().
func Register(name string, aliases []string, o Opener) error {
	if len(name) == 0 {
		return errors.New("firmatareg: can't register a bus with no name")
	}
	if o == nil {
		return errors.New("firmatareg: can't register bus " + strconv.Quote(name) + " with nil Opener")
	}
	if _, err := strconv.Atoi(name); err == nil {
		return errors.New("firmatareg: can't register bus " + strconv.Quote(name) + " with name being only a number")
	}
	if strings.Contains(name, ":") {
		return errors.New("firmatareg: can't register bus " + strconv.Quote(name) + " with name containing ':'")
	}
	for _, alias := range aliases {
		if len(alias) == 0 {
			return errors.New("firmatareg: can't register bus " + strconv.Quote(name) + " with an empty alias")
		}
		if name == alias {
			return errors.New("firmatareg: can't register bus " + strconv.Quote(name) + " with an alias the same as the bus name")
		}
		if _, err := strconv.Atoi(alias); err == nil {
			return errors.New("firmatareg: can't register bus " + strconv.Quote(name) + " with an alias that is a number: " + strconv.Quote(alias))
		}
		if strings.Contains(alias, ":") {
			return errors.New("firmatareg: can't register bus " + strconv.Quote(name) + " with an alias containing ':': " + strconv.Quote(alias))
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if _, ok := byName[name]; ok {
		return errors.New("firmatareg: can't register bus " + strconv.Quote(name) + " twice")
	}
	if _, ok := byAlias[name]; ok {
		return errors.New("firmatareg: can't register bus " + strconv.Quote(name) + " twice; it is already an alias")
	}
	for _, alias := range aliases {
		if _, ok := byName[alias]; ok {
			return errors.New("firmatareg: can't register bus " + strconv.Quote(name) + " twice; alias " + strconv.Quote(alias) + " is already a bus")
		}
		if _, ok := byAlias[alias]; ok {
			return errors.New("firmatareg: can't register bus " + strconv.Quote(name) + " twice; alias " + strconv.Quote(alias) + " is already an alias")
		}
	}

	r := &Ref{Name: name, Aliases: make([]string, len(aliases)), Open: o}
	copy(r.Aliases, aliases)
	byName[name] = r
	for _, alias := range aliases {
		byAlias[alias] = r
	}
	return nil
}

// Unregister removes a previously registered firmata device.
//
// This can happen when a firmata device is exposed via a USB device and the device
// is unplugged.
func Unregister(name string) error {
	mu.Lock()
	defer mu.Unlock()
	r := byName[name]
	if r == nil {
		return errors.New("firmatareg: can't unregister unknown bus name " + strconv.Quote(name))
	}
	delete(byName, name)
	for _, alias := range r.Aliases {
		delete(byAlias, alias)
	}
	return nil
}

//

var (
	mu     sync.Mutex
	byName = map[string]*Ref{}
	// Caches
	byNumber = map[int]*Ref{}
	byAlias  = map[string]*Ref{}
)

// getDefault returns the Ref that should be used as the default bus.
func getDefault() *Ref {
	var o *Ref
	if len(byNumber) == 0 {
		// Fallback to use byName using a lexical sort.
		name := ""
		for n, o2 := range byName {
			if len(name) == 0 || n < name {
				o = o2
				name = n
			}
		}
		return o
	}
	number := int((^uint(0)) >> 1)
	for n, o2 := range byNumber {
		if number > n {
			number = n
			o = o2
		}
	}
	return o
}

func insertRef(l []*Ref, r *Ref) []*Ref {
	n := r.Name
	i := search(len(l), func(i int) bool { return l[i].Name > n })
	l = append(l, nil)
	copy(l[i+1:], l[i:])
	l[i] = r
	return l
}

// search implements the same algorithm as sort.Search().
//
// It was extracted to not depend on sort, which depends on reflect.
func search(n int, f func(int) bool) int {
	lo := 0
	for hi := n; lo < hi; {
		if i := int(uint(lo+hi) >> 1); !f(i) {
			lo = i + 1
		} else {
			hi = i
		}
	}
	return lo
}
