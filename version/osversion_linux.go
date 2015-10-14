// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package version

func isWindowsNano() bool {
	return false
}

func osVersion() (string, error) {
	return readSeries()
}
