// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package version

func IsWindowsNano() bool {
	return false
}

func osVersion() (string, error) {
	return readSeries()
}
