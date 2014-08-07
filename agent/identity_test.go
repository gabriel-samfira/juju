// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package agent

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/juju/names"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/state/api/params"
	"github.com/juju/juju/testing"
	"github.com/juju/juju/version"
)

type identitySuite struct {
	testing.BaseSuite
	mongodConfigPath string
	mongodPath       string
}

var _ = gc.Suite(&identitySuite{})

var attributeParams = AgentConfigParams{
	Tag:               names.NewMachineTag("1"),
	UpgradedToVersion: version.Current.Number,
	Password:          "sekrit",
	CACert:            "ca cert",
	StateAddresses:    []string{"localhost:1234"},
	APIAddresses:      []string{"localhost:1235"},
	Nonce:             "a nonce",
}

var servingInfo = params.StateServingInfo{
	Cert:           "old cert",
	PrivateKey:     "old key",
	StatePort:      69,
	APIPort:        47,
	SharedSecret:   "shared",
	SystemIdentity: "identity",
}

func (s *identitySuite) TestWriteSystemIdentityFile(c *gc.C) {
	params := attributeParams
	params.DataDir = c.MkDir()
	conf, err := NewStateMachineConfig(params, servingInfo)
	c.Assert(err, gc.IsNil)
	err = WriteSystemIdentityFile(conf)
	c.Assert(err, gc.IsNil)

	contents, err := ioutil.ReadFile(conf.SystemIdentityPath())
	c.Assert(err, gc.IsNil)
	c.Check(string(contents), gc.Equals, servingInfo.SystemIdentity)

	fi, err := os.Stat(conf.SystemIdentityPath())
	c.Assert(err, gc.IsNil)
	if runtime.GOOS == "linux" {
		c.Check(fi.Mode().Perm(), gc.Equals, os.FileMode(0600))
	}
	// ensure that file is deleted when SystemIdentity is empty
	info := servingInfo
	info.SystemIdentity = ""
	conf, err = NewStateMachineConfig(params, info)
	c.Assert(err, gc.IsNil)
	err = WriteSystemIdentityFile(conf)
	c.Assert(err, gc.IsNil)
	fmt.Println(conf)
	fi, err = os.Stat(conf.SystemIdentityPath())
	if runtime.GOOS == "linux" {
		c.Assert(err, gc.ErrorMatches, `stat .*: no such file or directory`)
	} else if runtime.GOOS == "windows" {
		c.Assert(err, gc.ErrorMatches, `GetFileAttributesEx .*: The system cannot find the file specified.`)
	}

}
