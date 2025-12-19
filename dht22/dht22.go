// Package dht22 reads temperature and humidity from DHT22/AM2302 sensors.
//
// Datasheet: https://www.adafruit.com/datasheets/DHT22.pdf
package dht22

import (
	"errors"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/pin"
)

// Dev represents a DHT22/AM2302 sensor.
type Dev struct {
    pin gpio.PinIO
    mu  sync.Mutex
}

// New opens a connection to a DHT22 sensor on the given GPIO pin.
func New(p gpio.PinIO) (*Dev, error) {
    if err := p.In(gpio.PullUp, gpio.FallingEdge); err != nil {
        return nil, err
    }
    return &Dev{pin: p}, nil
}

// SenseContinuous returns a channel that will receive measurements at the
// specified interval. The channel is closed when stop is closed.
func (d *Dev) SenseContinuous(interval time.Duration, stop <-chan struct{}) (<-chan physic.Env, error) {
    ch := make(chan physic.Env)
    go func() {
        defer close(ch)
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for {
            select {
            case <-stop:
                return
            case <-ticker.C:
                if env, err := d.Read(); err == nil {
                    select {
                    case ch <- env:
                    case <-stop:
                        return
                    }
                }
            }
        }
    }()
    return ch, nil
}

// Read reads temperature and humidity from the sensor.
// Reading more frequently than once per 2 seconds may cause errors.
func (d *Dev) Read() (physic.Env, error) {
    d.mu.Lock()
    defer d.mu.Unlock()

    // Prepare pin for output
    if err := d.pin.Out(gpio.Low); err != nil {
        return physic.Env{}, err
    }

    // Send start signal: pull low for 1-10ms, then pull high for 20-40µs
    d.pin.Out(gpio.Low)
    time.Sleep(2 * time.Millisecond)
    d.pin.Out(gpio.High)
    time.Sleep(40 * time.Microsecond)

    // Switch to input with pull-up
    if err := d.pin.In(gpio.PullUp, gpio.NoEdge); err != nil {
        return physic.Env{}, err
    }

    // Wait for sensor response
    if !waitForState(d.pin, false, 80*time.Microsecond) {
        return physic.Env{}, errors.New("dht22: no response from sensor")
    }
    if !waitForState(d.pin, true, 80*time.Microsecond) {
        return physic.Env{}, errors.New("dht22: response timeout")
    }

    // Read 40 bits of data (humidity, temperature, checksum)
    var data [5]byte
    for i := 0; i < 40; i++ {
        if !waitForState(d.pin, false, 50*time.Microsecond) {
            return physic.Env{}, errors.New("dht22: data sync error")
        }
        
        // High pulse length determines bit value
        start := time.Now()
        waitForState(d.pin, true, 70*time.Microsecond)
        duration := time.Since(start)
        
        bit := byte(i / 8)
        shift := 7 - (i % 8)
        if duration > 50*time.Microsecond {
            data[bit] |= 1 << shift
        }
    }

    // Verify checksum
    if len(data) >= 5{
        checksum := data[0] + data[1] + data[2] + data[3]
        if data[4] != checksum {
            return physic.Env{}, errors.New("dht22: checksum error")
        }
    }

    // Parse data (big-endian)
    humidityInt := uint16(data[0])<<8 | uint16(data[1])
    tempRaw := int16(uint16(data[2])<<8 | uint16(data[3]))
    
    // Convert to physic units
    // humidity is 0.1% RH units, so divide by 10 to get percent
    humidity := physic.RelativeHumidity(float64(humidityInt)/10.0) * physic.PercentRH
    
    // temperature is 0.1°C units, with bit 15 for sign
    // Convert to nano Kelvin (base unit for physic.Temperature)
    // 1°C = 1K in delta, but physic uses nanoKelvin
    temperature := physic.Temperature(float64(tempRaw)/10.0) * physic.Celsius

    env := physic.Env{
        Humidity:    humidity,
        Temperature: temperature,
    }
    
    return env, nil
}

// String implements conn.Resource.
func (d *Dev) String() string {
    return "DHT22{" + d.pin.String() + "}"
}

// Halt implements conn.Resource.
func (d *Dev) Halt() error {
    return d.pin.Halt()
}

// Pin returns the GPIO pin used by the sensor.
func (d *Dev) Pin() pin.Pin {
    return d.pin
}

// waitForState waits for pin to reach desired state within timeout.
func waitForState(p gpio.PinIO, state bool, timeout time.Duration) bool {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if p.Read() == gpio.Level(state) {
            return true
        }
        time.Sleep(1 * time.Microsecond)
    }
    return false
}

// Ensure Dev implements conn.Resource.
var _ conn.Resource = &Dev{}