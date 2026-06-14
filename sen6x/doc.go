// Copyright 2026 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package sen6x controls the Sensirion SEN6x family of environmental sensors over I²C.
//
// # Details
//
// These sensors measure the following, in different combinations depending on the model:
//   - Particulate matter (PM1.0, PM2.5, PM4, and PM10, with the addition of PM0.5
//     from the Read Number Concentration Values command)
//   - Relative humidity
//   - Temperature
//   - VOC
//   - NOx
//   - Formaldehyde (HCHO)
//   - CO2
//
// Sensor model capabilities:
//   - [SEN62]: PM, RH, T
//   - [SEN63C]: PM, RH, T, CO2
//   - [SEN65]: PM, RH, T, VOC, NOx
//   - [SEN66]: PM, RH, T, VOC, NOx, CO2
//   - [SEN68]: PM, RH, T, VOC, NOx, HCHO
//   - [SEN69C]: PM, RH, T, VOC, NOx, HCHO, CO2
//
// All SEN6x sensors use a JST GH 1.25mm-pitch 6 pin connector (model number
// [GHR-06V-S], which uses connector model number [SSHL-002T-P0.2]).
//
// # Datasheet
//
// [Datasheet] for all SEN6x sensors. Also see ["What is Sensirion's VOC Index?"]
// and ["What is Sensirion's NOx Index?"].
//
// # Other resources
//
// Adafruit makes a nifty [breakout board] that bridges the SEN6x's JST GH connector
// with the standard STEMMA QT / Qwiic connector and includes a 3.3V regulator and
// level shifter so that the sensors work with either 3.3V or 5V power and logic.
// They also make a [cable] that works with the SEN6x, but you can of course make your
// own with the parts mentioned above.
//
// [SEN62]: https://sensirion.com/products/catalog/SEN62
// [SEN63C]: https://sensirion.com/products/catalog/SEN63C
// [SEN65]: https://sensirion.com/products/catalog/SEN65
// [SEN66]: https://sensirion.com/products/catalog/SEN66
// [SEN68]: https://sensirion.com/products/catalog/SEN68
// [SEN69C]: https://sensirion.com/products/catalog/SEN69C
// [GHR-06V-S]: https://www.digikey.com/en/products/detail/jst-sales-america-inc/GHR-06V-S/807818
// [SSHL-002T-P0.2]: https://www.digikey.com/en/products/detail/jst-sales-america-inc/SSHL-002T-P0-2/27687535
// [Datasheet]: https://sensirion.com/media/documents/FAFC548D/693FBB15/PS_DS_SEN6x.pdf
// ["What is Sensirion's VOC Index?"]: https://sensirion.com/media/documents/02232963/6294E043/Info_Note_VOC_Index.pdf
// ["What is Sensirion's NOx Index?"]: https://sensirion.com/media/documents/9F289B95/6294DFFC/Info_Note_NOx_Index.pdf
// [breakout board]: https://www.adafruit.com/product/6331
// [cable]: https://www.adafruit.com/product/5754
package sen6x
