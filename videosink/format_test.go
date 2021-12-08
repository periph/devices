// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package videosink

import (
	"fmt"
	"testing"
)

func TestImageFormat(t *testing.T) {
	for _, tc := range []struct {
		format       ImageFormat
		wantString   string
		wantMimeType string
	}{
		{
			format:       ImageFormat(-1),
			wantString:   "-1",
			wantMimeType: "application/octet-stream",
		},
		{
			wantString:   "PNG",
			wantMimeType: "image/png",
		},
		{
			format:       DefaultFormat,
			wantString:   "PNG",
			wantMimeType: "image/png",
		},
		{
			format:       PNG,
			wantString:   "PNG",
			wantMimeType: "image/png",
		},
		{
			format:       JPEG,
			wantString:   "JPEG",
			wantMimeType: "image/jpeg",
		},
	} {
		t.Run(fmt.Sprint(tc), func(t *testing.T) {
			if got := tc.format.String(); got != tc.wantString {
				t.Errorf("String() returned %q, want %q", got, tc.wantString)
			}

			if got := tc.format.mimeType(); got != tc.wantMimeType {
				t.Errorf("mimeType() returned %q, want %q", got, tc.wantMimeType)
			}
		})
	}
}
