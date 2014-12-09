// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package kvm_test

import (
	"runtime"
	"testing"

	gc "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("No point in testing for KVM on windows")
	}
	gc.TestingT(t)
}
