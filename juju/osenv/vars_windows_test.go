// Copyright 2013 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package osenv_test

import (
	"path/filepath"

	gc "launchpad.net/gocheck"

	"github.com/juju/juju/juju/osenv"
)

func (s *varsSuite) TestJujuHomeWin(c *gc.C) {
	path := `P:\FooBar\AppData`
	s.PatchEnvironment("APPDATA", path)
	c.Assert(osenv.JujuHomeWin(), gc.Equals, filepath.Join(path, "Juju"))
}
