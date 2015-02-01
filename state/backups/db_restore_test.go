// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package backups_test

import (
	"path/filepath"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/state/backups"
	"github.com/juju/juju/testing"
	"github.com/juju/juju/version"
)

var _ = gc.Suite(&mongoRestoreSuite{})

type mongoRestoreSuite struct {
	testing.BaseSuite
}

func (s *mongoRestoreSuite) TestMongoRestoreArgsForVersion(c *gc.C) {
	dir := filepath.Join(agent.DefaultDataDir, "db")
	versionNumber := version.Number{}
	versionNumber.Major = 1
	versionNumber.Minor = 21
	args, err := backups.MongoRestoreArgsForVersion(versionNumber, "/some/fake/path")
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(args[0], gc.Equals, "--drop")
	c.Assert(args[1], gc.Equals, "--dbpath")
	c.Assert(args[2], jc.SamePath, dir)
	c.Assert(args[3], gc.Equals, "/some/fake/path")

	versionNumber.Major = 1
	versionNumber.Minor = 22
	args, err = backups.MongoRestoreArgsForVersion(versionNumber, "/some/fake/path")
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(args[0], gc.Equals, "--drop")
	c.Assert(args[1], gc.Equals, "--oplogReplay")
	c.Assert(args[2], gc.Equals, "--dbpath")
	c.Assert(args[3], jc.SamePath, dir)
	c.Assert(args[4], gc.Equals, "/some/fake/path")

	versionNumber.Major = 0
	versionNumber.Minor = 0
	args, err = backups.MongoRestoreArgsForVersion(versionNumber, "/some/fake/path")
	c.Assert(err, gc.ErrorMatches, "this backup file is incompatible with the current version of juju")
}

func (s *mongoRestoreSuite) TestPlaceNewMongo(c *gc.C) {
	var argsVersion version.Number
	var newMongoDumpPath string
	ranArgs := make([][]string, 0, 3)
	ranCommands := []string{}

	restorePathCalled := false

	runCommand := func(command string, mongoRestoreArgs ...string) error {
		mgoArgs := make([]string, len(mongoRestoreArgs), len(mongoRestoreArgs))
		for i, v := range mongoRestoreArgs {
			mgoArgs[i] = v
		}
		ranArgs = append(ranArgs, mgoArgs)
		ranCommands = append(ranCommands, command)
		return nil
	}
	s.PatchValue(backups.RunCommand, runCommand)

	restorePath := func() (string, error) {
		restorePathCalled = true
		return "/fake/mongo/restore/path", nil
	}
	s.PatchValue(backups.RestorePath, restorePath)

	ver := version.Number{Major: 1, Minor: 22}
	args := []string{"a", "set", "of", "args"}
	restoreArgsForVersion := func(versionNumber version.Number, mongoDumpPath string) ([]string, error) {
		newMongoDumpPath = mongoDumpPath
		argsVersion = versionNumber
		return args, nil
	}
	s.PatchValue(backups.RestoreArgsForVersion, restoreArgsForVersion)

	err := backups.PlaceNewMongo("fakemongopath", ver)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(argsVersion, gc.DeepEquals, ver)
	c.Assert(newMongoDumpPath, gc.Equals, "fakemongopath")
	expectedCommands := []string{"initctl", "/fake/mongo/restore/path", "initctl"}
	c.Assert(ranCommands, gc.DeepEquals, expectedCommands)
	c.Assert(len(ranArgs), gc.Equals, 3)
	expectedArgs := [][]string{[]string{"stop", "juju-db"}, []string{"a", "set", "of", "args"}, []string{"start", "juju-db"}}
	c.Assert(ranArgs, gc.DeepEquals, expectedArgs)
}
