// Copyright 2023 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package inky

//go:generate go install golang.org/x/tools/cmd/stringer@latest
//go:generate stringer -type=Model,Color,ImpressionColor -output types_string.go

import (
	"fmt"
)

// Model lists the supported e-ink display models.
type Model int

// Supported Model.
const (
	PHAT Model = iota
	WHAT
	PHAT2
	IMPRESSION4
	IMPRESSION57
	IMPRESSION73
)

// Set sets the Model to a value represented by the string s. Set implements the flag.Value interface.
func (m *Model) Set(s string) error {
	switch s {
	case "PHAT":
		*m = PHAT
	case "PHAT2":
		*m = PHAT2
	case "WHAT":
		*m = WHAT
	case "IMPRESSION4":
		*m = IMPRESSION4
	case "IMPRESSION57":
		*m = IMPRESSION57
	case "IMPRESSION73":
		*m = IMPRESSION73
	default:
		return fmt.Errorf("unknown model %q: expected PHAT, PHAT2, WHAT, IMPRESSION4 or IMPRESSION57 or IMPRESSION73", s)
	}
	return nil
}

// Color is used to define which model of inky is being used, and also for
// setting the border color.
type Color int

// Valid Color.
const (
	Black Color = iota
	Red
	Yellow
	White
	Multi
)

// Set sets the Color to a value represented by the string s. Set implements the flag.Value interface.
func (c *Color) Set(s string) error {
	switch s {
	case "black":
		*c = Black
	case "red":
		*c = Red
	case "yellow":
		*c = Yellow
	case "white":
		*c = White
	default:
		return fmt.Errorf("unknown color %q: expected either black, red, yellow or white", s)
	}
	return nil
}

// ImpressionColor is used to define colors used by Inky Impression models.
type ImpressionColor uint8

const (
	BlackImpression ImpressionColor = iota
	WhiteImpression
	GreenImpression
	BlueImpression
	RedImpression
	YellowImpression
	OrangeImpression
	CleanImpression
)

// Set sets the ImpressionColor to a value represented by the string s. Set implements the flag.Value interface.
func (c *ImpressionColor) Set(s string) error {
	switch s {
	case "black":
		*c = BlackImpression
	case "white":
		*c = WhiteImpression
	case "green":
		*c = GreenImpression
	case "blue":
		*c = BlueImpression
	case "red":
		*c = RedImpression
	case "yellow":
		*c = YellowImpression
	case "orange":
		*c = OrangeImpression
	case "clean":
		*c = CleanImpression
	default:
		return fmt.Errorf("unknown color %q: expected either black, white. green, blue, red, yellow, orange or clean", s)
	}
	return nil
}
