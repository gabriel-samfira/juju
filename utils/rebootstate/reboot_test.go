package rebootstate_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	jujutesting "github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/utils/rebootstate"
)

func TestPackage(t *testing.T) {
	gc.TestingT(t)
}

type rebootstateSuite struct {
	jujutesting.IsolationSuite
}

var _ = gc.Suite(&rebootstateSuite{})

func (s *rebootstateSuite) SetUpTest(c *gc.C) {
	dataDir := c.MkDir()
	s.PatchValue(&rebootstate.RebootStateFile, filepath.Join(dataDir, "reboot-state.txt"))
	s.IsolationSuite.SetUpTest(c)
}

func (s *rebootstateSuite) TestNewState(c *gc.C) {
	s.PatchValue(&rebootstate.UptimeFunc, func() (int64, error) { return int64(2), nil })
	err := rebootstate.New()
	c.Assert(err, gc.IsNil)
	contents, err := ioutil.ReadFile(rebootstate.RebootStateFile)
	c.Assert(err, gc.IsNil)
	c.Assert(string(contents), gc.Equals, "2")
}

func (s *rebootstateSuite) TestMultipleNewState(c *gc.C) {
	s.PatchValue(&rebootstate.UptimeFunc, func() (int64, error) { return int64(2), nil })
	err := rebootstate.New()
	c.Assert(err, gc.IsNil)
	err = rebootstate.New()
	c.Assert(err, gc.ErrorMatches, "state file (.*) already exists")
}

func (s *rebootstateSuite) TestReadState(c *gc.C) {
	s.PatchValue(&rebootstate.UptimeFunc, func() (int64, error) { return int64(2), nil })
	expectUtime, _ := rebootstate.UptimeFunc()
	err := rebootstate.New()
	c.Assert(err, gc.IsNil)
	uptime, err := rebootstate.Read()
	c.Assert(err, gc.IsNil)
	c.Assert(uptime, gc.Equals, expectUtime)
}

func (s *rebootstateSuite) fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func (s *rebootstateSuite) TestRemoveState(c *gc.C) {
	s.PatchValue(&rebootstate.UptimeFunc, func() (int64, error) { return int64(2), nil })
	err := rebootstate.New()
	c.Assert(err, gc.IsNil)
	err = rebootstate.Remove()
	c.Assert(err, gc.IsNil)
	c.Assert(s.fileExists(rebootstate.RebootStateFile), jc.IsFalse)
}
