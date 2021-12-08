// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package videosink

import "fmt"

type ImageFormat int

const (
	PNG ImageFormat = iota
	JPEG

	// DefaultFormat is the format used when not set explicitly in options or
	// as a URL parameter.
	DefaultFormat = PNG
)

func (f ImageFormat) String() string {
	switch f {
	case PNG:
		return "PNG"
	case JPEG:
		return "JPEG"
	default:
		return fmt.Sprint(int(f))
	}
}

func (f ImageFormat) mimeType() string {
	switch f {
	case PNG:
		return "image/png"
	case JPEG:
		return "image/jpeg"
	}

	return "application/octet-stream"
}

// ImageFormatFromString returns the ImageFormat value for the given format
// abbreviation.
func ImageFormatFromString(value string) (ImageFormat, error) {
	switch value {
	case "png":
		return PNG, nil
	case "jpg", "jpeg":
		return JPEG, nil
	}

	return DefaultFormat, fmt.Errorf("unrecognized image format %q", value)
}
