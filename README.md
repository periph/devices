# periph - Peripherals I/O in Go

Documentation is at https://periph.io

Join us for a chat on
[gophers.slack.com/messages/periph](https://gophers.slack.com/messages/periph),
get an [invite here](https://invite.slack.golangbridge.org/).

[![mascot](https://raw.githubusercontent.com/periph/website/master/site/static/img/periph-mascot-280.png)](https://periph.io/)

[![PkgGoDev](https://pkg.go.dev/badge/periph.io/x/devices/v3)](https://pkg.go.dev/periph.io/x/devices/v3)
[![codecov](https://codecov.io/gh/periph/devices/branch/main/graph/badge.svg?token=UA4NGFM2YJ)](https://codecov.io/gh/periph/devices)


## Example

Blink a LED:

~~~go
package main

import (
    "time"
    "periph.io/x/conn/v3/gpio"
    "periph.io/x/host/v3"
    "periph.io/x/host/v3/rpi"
)

func main() {
    host.Init()
    t := time.NewTicker(500 * time.Millisecond)
    for l := gpio.Low; ; l = !l {
        rpi.P1_33.Out(l)
        <-t.C
    }
}
~~~

Curious? Look at [supported devices](https://periph.io/device/) for more
examples!


## Supported Devices

| Package | Description |
|---------|-------------|
| [ads1x15](ads1x15) | ADS1015/ADS1115 analog-to-digital converters via I²C |
| [adxl345](adxl345) | ADXL345 3-axis accelerometer over SPI |
| [aht20](aht20) | AHT20 temperature/humidity sensor over I²C |
| [aip31068](aip31068) | AIP31068 HD44780-compatible I²C LCD driver |
| [am2320](am2320) | AOSONG AM2320 temperature/humidity sensor |
| [apa102](apa102) | APA102 LED strip over SPI |
| [as7262](as7262) | AMS AS7262 6-channel visible spectral sensor via I²C |
| [bh1750](bh1750) | ROHM BH1750 ambient light sensor over I²C |
| [bitbang](bitbang) | Software bit-banging protocols (I²C, SPI, UART) via GPIO pins |
| [bmxx80](bmxx80) | Bosch BMP180/BME280/BMP280 temperature/pressure/humidity sensor over I²C or SPI |
| [cap1xxx](cap1xxx) | Microchip CAP1xxx capacitive touch sensors over I²C |
| [ccs811](ccs811) | CCS811 volatile organic compound (VOC) sensor via I²C |
| [common](common) | Shared utility functions used across multiple packages |
| [dht22](dht22) | DHT22/AM2302 temperature/humidity sensor |
| [ds18b20](ds18b20) | DS18B20/DS18S20/MAX31820 1-wire temperature sensors |
| [ds248x](ds248x) | Maxim DS2483/DS2482-100 1-wire interface chip over I²C |
| [ep0099](ep0099) | EP-0099 Raspberry Pi HAT with 4 relays via I²C |
| [epd](epd) | Waveshare e-paper display series |
| [gc9a01](gc9a01) | GC9A01 240x240 round RGB LCD display over SPI |
| [hd44780](hd44780) | Hitachi HD44780 LCD display chipset |
| [hdc302x](hdc302x) | Texas Instruments HDC3021/3022 temperature/humidity sensor over I²C |
| [ht16k33](ht16k33) | Holtek HT16K33 16×8 LED driver |
| [hx711](hx711) | HX711 24-bit analog-to-digital converter (load cells) |
| [ina219](ina219) | Texas Instruments INA219 current/voltage/power monitor over I²C |
| [inky](inky) | Pimoroni Inky pHAT/wHAT e-ink displays |
| [lepton](lepton) | FLIR Lepton infrared (IR) camera |
| [lirc](lirc) | Infrared receiver support via Linux LIRC |
| [matrixorbital](matrixorbital) | MatrixOrbital character LCD displays |
| [max7219](max7219) | MAX7219 7-segment and LED matrix displays |
| [mcp23xxx](mcp23xxx) | Microchip MCP23xxx GPIO expanders |
| [mcp472x](mcp472x) | Microchip MCP472x digital-to-analog converters |
| [mcp9808](mcp9808) | Microchip MCP9808 temperature sensor |
| [mfrc522](mfrc522) | MFRC522 Mifare RFID card reader |
| [mpu9250](mpu9250) | MPU-9250 9-axis IMU (gyroscope, accelerometer, magnetometer) |
| [nrzled](nrzled) | WS2811/WS2812/WS2812B and compatible NRZ-encoded LEDs (SK6812, UCS1903) |
| [nxp74hc595](nxp74hc595) | 74HC595 serial-in parallel-out shift register |
| [pca9548](pca9548) | PCA9548 8-port I²C multiplexer |
| [pca9633](pca9633) | PCA9633 4-channel LED PWM controller |
| [pca9685](pca9685) | PCA9685 16-channel PWM controller (servos, LEDs) |
| [pcf857x](pcf857x) | TI/NXP PCF857x I²C I/O expander |
| [piblaster](piblaster) | PWM via the pi-blaster daemon on Raspberry Pi |
| [rainbowhat](rainbowhat) | Pimoroni Rainbow HAT |
| [scd4x](scd4x) | Sensirion SCD4x CO₂ sensors |
| [screen1d](screen1d) | 1D display output to terminal via ANSI color codes |
| [serlcd](serlcd) | SparkFun SerLCD intelligent LCD display |
| [sgp30](sgp30) | Sensirion SGP30 multi-gas sensor (TVOC and CO₂eq) |
| [sht4x](sht4x) | Sensirion SHT-40/SHT-41/SHT-45 temperature/humidity sensors |
| [sn3218](sn3218) | SN3218 18-channel LED driver over I²C |
| [ssd1306](ssd1306) | SSD1306/SH1106/SH1107 monochrome OLED displays |
| [st7567](st7567) | ST7567 single-chip dot matrix LCD |
| [tca95xx](tca95xx) | Texas Instruments TCA95xx 8-bit I²C GPIO expanders |
| [tic](tic) | Tic stepper motor controllers via I²C |
| [tlv493d](tlv493d) | Infineon TLV493D 3D magnetic (Hall effect) sensor |
| [tm1637](tm1637) | TM1637 LED display driver over GPIO |
| [tmp102](tmp102) | Texas Instruments TMP102 temperature sensor over I²C |
| [unicornhd](unicornhd) | Pimoroni Unicorn HD HAT (16×16 RGB LED matrix) |
| [videosink](videosink) | Display driver serving frames over HTTP |
| [waveshare1602](waveshare1602) | Waveshare 1602 LCD display |
| [waveshare2in13v2](waveshare2in13v2) | Waveshare 2.13" v2 e-paper display |
| [waveshare2in13v3](waveshare2in13v3) | Waveshare 2.13" v3 e-paper display |
| [waveshare2in13v4](waveshare2in13v4) | Waveshare 2.13" v4 e-paper display |


## Authors

`periph` was initiated with ❤️️ and passion by [Marc-Antoine
Ruel](https://github.com/maruel). The full list of contributors is in
[AUTHORS](https://github.com/periph/devices/blob/main/AUTHORS) and
[CONTRIBUTORS](https://github.com/periph/devices/blob/main/CONTRIBUTORS).
