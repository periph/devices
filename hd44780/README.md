# Hitachi HD44780 Package

## Overview

The Hitachi HD44780 is an LCD driver chip. It's used in a variety of text LCD 
displays. The datasheet is available here:

https://www.sparkfun.com/datasheets/LCD/HD44780.pdf

Generally, there are three kinds of displays that use this chip.

1. A raw display. This is a board with a row of pin connectors at the top. It's
made for interfacing directly to GPIO pins. This is the most complicated method
of using the LCD because you have to wire a minimum of 6 GPIO pins (4 data, 1 
reset, 1 enable) and 2 power pins to make it work.

There are also complex initialization routines that have to be performed. This
is further complicated by having specific initialization calls depending upon
whether the device is connected to 4 data lines, or 8 data lines.

2. A backpack display. With this type of LCD display, an I2C, serial, or SPI
interface is provided. This type of display is easier because there are fewer
pins, but the complex intialization remains. Examples of backpack interfaces
include The Adafruit I2C/SPI backpack (MCP23008/74HC595D), the generic 
LCDXXXX/PCF8547T backpack, etc.

3. The third and final variety is the "Intelligent" display. These displays have
a micro-controller that is connected to the LCD chip. Typically these intelligent
displays support multiple I/O methods and the micro-controller handles the
LCD initialization/communication.

Examples of this kind of display would be the SparkFun SerLCD display, the
MatrixOrbital LK2047T, the AdaFruit USB+Serial Backpack etc.

## Interfacing Notes

The driver package is designed to use the gpio.Group interface. This allows
the LCD driver to be agnostic about the physical connection between the display 
and the host device. Any host/expander that supports the 
periph.io/x/conn/v3/gpio.Group interface can be used to easily drive the LCD 
display.

## Hardware Notes

DO NOT attempt to source VCC for the unit backlight, or sink VCC to ground.
The backlight draws ~250ma of current which exceeds the current capability
of GPIO pins. It will permanently damage your device. If you would like to 
control the backlight, connect a GPIO pin through a 1K Ohm resistor to a 
transistor (2N2222 or equivalent).

## Troubleshooting

If nothing displays at all, check the contrast. Adjust the contrast control
until the 5x7 dot grid on the display is visible.

If the first row contains blocks of dots, and the other rows are blank, then
the initialization of the device failed. Check IO pins are connected properly.

If the text is garbled, verify the gpio.Group is configured and the IO Pins
are connected to the right pins of the LCD display.
