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


## Authors

`periph` was initiated with ❤️️ and passion by [Marc-Antoine
Ruel](https://github.com/maruel). The full list of contributors is in
[AUTHORS](https://github.com/periph/devices/blob/main/AUTHORS) and
[CONTRIBUTORS](https://github.com/periph/devices/blob/main/CONTRIBUTORS).
