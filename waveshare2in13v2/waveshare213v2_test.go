// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package waveshare2in13v2

import (
	"fmt"
	"testing"
)

func TestDataDimensions(t *testing.T) {
	for _, tc := range []struct {
		opts       *Opts
		wantHeight int
		wantWidth  int
	}{
		{opts: &Opts{Width: 0, Height: 0}},
		{
			opts:       &Opts{Height: 48, Width: 16},
			wantHeight: 48,
			wantWidth:  2,
		},
		{
			opts:       &Opts{Height: 250, Width: 122},
			wantHeight: 250,
			wantWidth:  16,
		},
	} {
		t.Run(fmt.Sprintf("%+v", *tc.opts), func(t *testing.T) {
			gotHeight, gotWidth := dataDimensions(tc.opts)

			if !(gotHeight == tc.wantHeight && gotWidth == tc.wantWidth) {
				t.Errorf("dataDimensions(%#v) returned %d, %d; want %d, %d", tc.opts, gotHeight, gotWidth, tc.wantHeight, tc.wantWidth)
			}
		})
	}
}
