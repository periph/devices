# Max7219 7-Segment/Matrix LED Driver

## Introduction

The Maxim 7219 LED driver is an SPI chip that simplifies driving 7-Segment LED 
displays. It's also commonly used to create 8x8 LED matrixes. The datasheet 
is available at:

https://www.analog.com/media/en/technical-documentation/data-sheets/MAX7219-MAX7221.pdf

You can find 8 digit, 7-segment LED boards on Ebay for $1-2. You can find 
inexpensive matrixes in various sizes from 1 - 8 segments as well. 

For matrixes, the driver provides a basic CP437 font that can be used to
display characters on 8x8 matrixes. If desired, you can supply your own glyph
set, or special characters for your specific application.

## Driver Functions

This driver provides simplified handling for using either 7-segment numeric 
displays, or 8x8 matrixes.

It provides methods for scrolling data across either one or multiple displays.
For example, you can pass an IP address of "10.100.10.11" to the scroll function
and it will automatically scroll the display a desired number of times. The 
scroll feature also works with matrixes, and scrolls characters one LED column
at a time for a smooth, even display.

## Notes About Daisy-Chaining

The Max7219 is specifically designed to handle larger displays by daisy chaining 
units together. Say you want to write to digit 8 on the 3rd unit chained 
together. You would make one SPI write. The first write would be the register 
number (8), and the data value. Next, you would write a NOOP. The first record 
would then be shifted to the second device. Finally, you write another NOOP. The 
first record is shifted from the second device to the 3rd device, the first NOOP is 
shifted to the second device. When the ChipSelect line goes low, each unit applies 
the last data it received.
