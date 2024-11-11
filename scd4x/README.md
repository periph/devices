# Sensirion SCD4x CO<sub>2</sub> Sensors

## Overview

This package provides a driver for the Sensirion SCD4x CO<sub>2</sub> sensors. This is a 
compact sensor that provides temperature, humidity, and CO<sub>2</sub> concentration 
readings. The datasheet for this device is available at:

https://sensirion.com/media/documents/48C4B7FB/66E05452/CD_DS_SCD4x_Datasheet_D1.pdf

## Testing

The unit tests can function with either a live sensor, or in playback mode. If the 
environment variable SCD4X is set, then the self test code will use a live 
sensor on the default I<sup>2</sup>C bus. For example:

```bash
$> SCD4X=1 go test -v
```
If the environment variable is not present, then unit tests will be conducted using
playback values.

## Notes

### Acquisition Time

The minimum acquisition time for the sensor is 5 seconds. If you call Sense() more
frequently, it will block until a reading is ready.

### Forced Calibration and Self-Test

These functions are not implemented. From examining the datasheet, and 
experimenting, it appears that these two calls require the i2c communication 
driver to wait a specified period before initiating the read. The periph.io 
I<sup>2</sup>C library doesn't support this functionality. This means that attempts 
to call these functions will always fail so they're not implemented.

### Acquisition Mode

Only certain commands can be issued while the device is running in acquisition 
mode. If you're working on the low-level code, be aware that attempts to send
a non-allowed command while in acquisition mode will return an i2c remote 
io-error.

### Automatic Self Calibration

When Automatic Self Calibration is enabled, and the sensor has run for the 
required period, it will adjust itself so that the LOWEST recorded reading 
during the period yields the value set for ASC Target. The factory default 
target is 400PPM, but the current PPM is ~425PPM. To get a more accurate 
value for CO2 concentration in Earth's atmosphere, refer to:

https://www.co2.earth/daily-co2

For more details, refer to the datasheet.

