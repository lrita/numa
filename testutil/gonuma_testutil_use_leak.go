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
)

// LeakVerifyNone -> enabled (wrapper function)
func LeakVerifyNone(
	t *testing.T,
) error {
	err := leakVerifyNone(
		t,
		true,
	)
	if err != nil {
		return fmt.Errorf(
			fmt.Sprintf(
				"	%v",
				err,
			),
		)
	}
	return nil
}
