package aht20

import (
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2ctest"
	"periph.io/x/conn/v3/physic"
	"testing"
)

func TestDev_Sense(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Trigger measurement
			{Addr: deviceAddress, W: argsMeasure},
			// Read measurement
			{Addr: deviceAddress, R: []byte{0x18, 0x75, 0x52, 0x05, 0x8E, 0x40, 0x7F}},
		},
	}
	dev := Dev{d: &i2c.Dev{Bus: &bus, Addr: deviceAddress}, opts: DefaultOpts}
	e := physic.Env{}
	if err := dev.Sense(&e); err != nil {
		t.Fatal(err)
	}
	if expected := 19445800781*physic.NanoKelvin + physic.ZeroCelsius; e.Temperature != expected {
		t.Fatalf("temperature %s(%d) != %s(%d)", expected, expected, e.Temperature, e.Temperature)
	}
	if expected := 4582824 * physic.TenthMicroRH; e.Humidity != expected {
		t.Fatalf("humidity %s(%d) != %s(%d)", expected, expected, e.Humidity, e.Humidity)
	}
	if expected := 0 * physic.Pascal; e.Pressure != expected {
		t.Fatalf("pressure %s(%d) != %s(%d)", expected, expected, e.Pressure, e.Pressure)
	}
	if err := bus.Close(); err != nil {
		t.Fatal(err)
	}
}
