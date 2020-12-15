// Copyright 2020 Jeffrey H. Johnson.
// Copyright 2020 Gridfinity, LLC.
// Copyright 2019 The Go Authors.
// All rights reserved.
// Use of this source code is governed by the BSD-style
// license that can be found in the LICENSE file.

// +build leaktest

package testutil

import (
	"fmt"
	"testing"

	goleak "go.uber.org/goleak"
)

func leakVerifyNone(
	t *testing.T,
	r bool,
) error {
	if r != true {
		return fmt.Errorf(
			"testutil.leakVerifyNone: r != true",
		)
	}
	goleak.VerifyNone(
		t,
	)
	return nil
}
