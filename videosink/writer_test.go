// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Tests MIME multipart boundary generation.

package videosink

import (
	"regexp"
	"testing"
)

var boundaryRe = regexp.MustCompile(`^[a-f0-9]{60,70}$`)

func TestRandomBoundary(t *testing.T) {
	t.Parallel()
	for range 100 {
		if got := randomBoundary(); !boundaryRe.MatchString(got) {
			t.Errorf("Boundary must match the expression %q: %s", boundaryRe.String(), got)
		}
	}
}
