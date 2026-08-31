package ina260_test

import (
	"fmt"
	"log"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/ina260"
	"periph.io/x/host/v3"
)

func main() {
	if _, err := host.Init(); err != nil {
		fmt.Println(err)
	}

	busNumber := 0
	bus, err := i2creg.Open(fmt.Sprintf("/dev/i2c-%d", busNumber))
	if err != nil {
		log.Fatal(err)
	}
	defer bus.Close()

	// Fuel gauge
	fuelGauge := ina260.New(bus)
	if err != nil {
		log.Fatal(err)
	}
	for {
		f, err := fuelGauge.Read()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%f V    %f A    %f W", f.Voltage, f.Current, f.Power)
	}

}
