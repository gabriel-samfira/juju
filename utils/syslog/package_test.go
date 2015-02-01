// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package syslog_test

import (
	"runtime"
	"testing"

	gc "gopkg.in/check.v1"
)

func TestPackage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping rsyslog tests on windows")
	}
	gc.TestingT(t)
}
