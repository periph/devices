// Copyright 2024 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.
//
// tmp102 provides a package for interfacing a Texas Instruments TMP102 I2C
// temperature sensor. This driver is also compatible with the TMP112 and
// TMP75 sensors.
//
// Range: -40°C - 125°C
//
// Accuracy: +/- 0.5°C
//
// Resolution: 0.0625°C
//
// For detailed information, refer to the [datasheet].
//
// A [command line example] is available in periph.io/x/devices/cmd/tmp102
//
// [datasheet]: https://www.ti.com/lit/ds/symlink/tmp102.pdf
// [command line example]: https://github.com/periph/cmd/tree/main/tmp102/
package tmp102
