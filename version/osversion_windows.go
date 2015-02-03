// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package version

import (
	"strings"

	"github.com/juju/errors"

	"github.com/juju/juju/utils/winreg"
)

func osVersion() (string, error) {
	ver, err := winreg.ReadRegistryString(
		"HKLM:\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion", "ProductName")
	if err != nil {
		return "unknown", err
	}
	if val, ok := windowsVersions[ver]; ok {
		return val, nil
	}
	for key, value := range windowsVersions {
		if strings.HasPrefix(ver, key) {
			return value, nil
		}
	}
	return "unknown", errors.Errorf("unknown series %q", ver)
}
