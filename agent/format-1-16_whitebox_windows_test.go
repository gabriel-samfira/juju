// Copyright 2013 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

// The format tests are white box tests, meaning that the tests are in the
// same package as the code, as all the format details are internal to the
// package.

package agent

import (
	"path/filepath"

	"github.com/juju/utils"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/juju/paths"
	"github.com/juju/juju/version"
)

func (s *format_1_16Suite) TestMissingAttributesWindows(c *gc.C) {
	logDir, err := paths.LogDir("win8")
	c.Assert(err, gc.IsNil)
	realDataDir, err := paths.DataDir("win8")
	c.Assert(err, gc.IsNil)

	realDataDir = filepath.FromSlash(realDataDir)
	logPath := filepath.Join(logDir, "juju")
	logPath = filepath.FromSlash(logPath)

	dataDir := c.MkDir()
	formatPath := filepath.Join(dataDir, legacyFormatFilename)
	err = utils.AtomicWriteFile(formatPath, []byte(legacyFormatFileContents), 0600)
	c.Assert(err, gc.IsNil)
	configPath := filepath.Join(dataDir, agentConfigFilename)

	err = utils.AtomicWriteFile(configPath, []byte(configDataWithoutNewAttributes), 0600)
	c.Assert(err, gc.IsNil)
	readConfig, err := ReadConfig(configPath)
	c.Assert(err, gc.IsNil)
	c.Assert(readConfig.UpgradedToVersion(), gc.Equals, version.MustParse("1.16.0"))
	configLogDir := filepath.FromSlash(readConfig.LogDir())
	configDataDir := filepath.FromSlash(readConfig.DataDir())

	c.Assert(configLogDir, gc.Equals, logPath)
	c.Assert(configDataDir, gc.Equals, realDataDir)
	// Test data doesn't include a StateServerKey so StateServingInfo
	// should *not* be available
	_, available := readConfig.StateServingInfo()
	c.Assert(available, gc.Equals, false)
}
