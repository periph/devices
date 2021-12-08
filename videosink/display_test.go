// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package videosink

import "testing"

func TestNewHalt(t *testing.T) {
	d := New(&Options{Width: 100, Height: 100})

	if err := d.Halt(); err != nil {
		t.Errorf("Halt() failed: %v", err)
	}
}
