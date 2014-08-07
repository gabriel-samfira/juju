// Copyright 2013 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package osenv_test

import (
	"path/filepath"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/juju/osenv"
)

func (s *varsSuite) TestJujuHomeLinux(c *gc.C) {
	path := `/foo/bar/baz/`
	s.PatchEnvironment("HOME", path)
	c.Assert(osenv.JujuHomeLinux(), gc.Equals, filepath.Join(path, ".juju"))
}
