// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package version_test

import (
	"sort"

	"github.com/juju/juju/testing"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/version"
)

type supportedSeriesWindowsSuite struct {
	testing.BaseSuite
}

var _ = gc.Suite(&supportedSeriesWindowsSuite{})

func (s *supportedSeriesWindowsSuite) TestSeriesVersion(c *gc.C) {
	vers, err := version.SeriesVersion("win8")
	if err != nil && err.Error() == `invalid series "win8"` {
		c.Fatalf(`Unable to lookup series "win8"`)
	}
	c.Assert(err, gc.IsNil)
	c.Assert(vers, gc.Equals, "win8")
}

func (s *supportedSeriesWindowsSuite) TestSupportedSeries(c *gc.C) {
	series := version.SupportedSeries()
	sort.Strings(series)
	c.Assert(series, gc.DeepEquals, []string{"precise", "quantal", "raring", "saucy", "trusty", "utopic", "win2012", "win2012hv", "win2012hvr2", "win2012r2", "win7", "win8", "win81"})
}
