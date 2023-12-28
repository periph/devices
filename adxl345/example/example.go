package example

import (
	"fmt"
	"log"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/adxl345"
	"periph.io/x/host/v3"
	"time"
)

// Example reads the acceleration values every 30ms for 3 seconds.
func Example() {

	// Initialize the host
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Use spireg SPI port registry to find the first available SPI bus.
	p, err := spireg.Open("")
	if err != nil {
		log.Fatal(err)
	}

	defer p.Close()

	d, err := adxl345.New(p, &adxl345.DefaultOpts)
	if err != nil {
		panic(err)
	}

	fmt.Println(d.String())

	// use a ticker to read the acceleration values every 200ms
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	// stop after 3 seconds
	stop := time.After(3 * time.Second)

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			a := d.Update()
			fmt.Println(a)
		}
	}
}
