// Copyright 2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package reboot_test

import (
	"fmt"
	"path/filepath"
	stdtesting "testing"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/api"
	apireboot "github.com/juju/juju/api/reboot"
	"github.com/juju/juju/apiserver/params"
	jujutesting "github.com/juju/juju/juju/testing"
	coretesting "github.com/juju/juju/testing"
	"github.com/juju/juju/utils/rebootstate"
	"github.com/juju/juju/state"
	"github.com/juju/juju/worker"
	"github.com/juju/juju/worker/reboot"
)

func TestAll(t *stdtesting.T) {
	coretesting.MgoTestPackage(t)
}

type RebootSuite struct {
	jujutesting.JujuConnSuite

	stateMachine	*state.Machine
	apiState		*api.State
	rebootState		*apireboot.State
}

var _ = gc.Suite(&RebootSuite{})

func (s *RebootSuite) SetUpSuite(c *gc.C) {
	s.JujuConnSuite.SetUpSuite(c)
}

func (s *RebootSuite) SetUpTest(c *gc.C) {
	var err error
	s.PatchValue(&reboot.LockDir, c.MkDir())
	s.PatchValue(&rebootstate.RebootStateFile, 
		filepath.Join(c.MkDir(), "reboot-state.txt"))

	s.JujuConnSuite.SetUpTest(c)
	s.apiState, s.stateMachine = s.OpenAPIAsNewMachine(c)
	s.rebootState, err = s.apiState.Reboot()
	c.Assert(s.rebootState, gc.NotNil)
	c.Assert(err, gc.IsNil)
}

func (s *RebootSuite) TearDownTest(c *gc.C) {
	s.JujuConnSuite.TearDownTest(c)
}

func (s *RebootSuite) TearDownSuite(c *gc.C) {
	s.JujuConnSuite.TearDownSuite(c)
}


func (s *RebootSuite) TestCheckForRebootState(c *gc.C) {
	rebootWorker := reboot.NewRebootStruct(s.rebootState)

	// test immediate return when no state files are found
	err := rebootWorker.CheckForRebootState()
	c.Assert(err, gc.IsNil)

	// test that an error in a GetRebootAction call triggers abort 
	// while leaving the state file intact
	err = rebootstate.New()
	c.Assert(err, gc.IsNil)
	apireboot.PatchFacadeCall(s, s.rebootState, func(name string, p, resp interface{}) error {
			return fmt.Errorf("GetRebootAction call error!")
		})

	err = rebootWorker.CheckForRebootState()
	c.Assert(err, gc.ErrorMatches, "GetRebootAction call error!")
	c.Assert(rebootstate.IsPresent(), jc.IsTrue)


	// test for succesful GetRebootAction call with returned ShouldDoNothing
	// flag and that reboot state file is properly cleared afterwards
	apireboot.PatchFacadeCall(s, s.rebootState, func(name string, p, resp interface{}) error {
			if resp, ok := resp.(*params.RebootActionResults); ok {
				resp.Results = []params.RebootActionResult {
					{ Result: params.ShouldDoNothing },
				}
			}
			return nil
		})

	err = rebootWorker.CheckForRebootState()
	c.Assert(err, gc.IsNil)
	c.Assert(rebootstate.IsPresent(), jc.IsFalse)

	// test for succesful call of GetRebootAction call but failed ClearReboot
	// and that state file is left intact
	err = rebootstate.New()
	c.Assert(err, gc.IsNil)
	apireboot.PatchFacadeCall(s, s.rebootState, func(name string, p, resp interface{}) error {
			if name == "GetRebootAction" {
				if resp, ok := resp.(*params.RebootActionResults); ok {
					resp.Results = []params.RebootActionResult {
						{ Result: params.ShouldReboot },
					}	
				}
			} else {
				if resp, ok := resp.(*params.ErrorResults); ok {
					resp.Results = []params.ErrorResult {
						{ Error: &params.Error{ Message: "ClearReboot call error!" } },
					}
				}
			}
			return nil
		})

	err = rebootWorker.CheckForRebootState()
	c.Assert(err, gc.ErrorMatches, "ClearReboot call error!")
	c.Assert(rebootstate.IsPresent(), jc.IsTrue)

	// test for succesful calls of GetRebootAction and ClearReboot
	// assuring that the state files were cleared
	apireboot.PatchFacadeCall(s, s.rebootState, func(name string, p, resp interface{}) error {
			if name == "GetRebootAction" {
				if resp, ok := resp.(*params.RebootActionResults); ok {
					resp.Results = []params.RebootActionResult {
						{ Result: params.ShouldReboot },
					}
				}
			} else {
				if resp, ok := resp.(*params.ErrorResults); ok {
					resp.Results = []params.ErrorResult {
						{ Error: nil },
					}
				}
			}
			return nil
		})

	err = rebootWorker.CheckForRebootState()
	c.Assert(err, gc.IsNil)
	c.Assert(rebootstate.IsPresent(), jc.IsFalse)
}

func (s *RebootSuite)TestHandleFacadeCallError(c *gc.C) {
	rebootWorker := reboot.NewRebootStruct(s.rebootState)

	apireboot.PatchFacadeCall(s, s.rebootState, func(name string, p, resp interface{}) error {
			return fmt.Errorf("GetRebootAction call error!")
		})

	err := rebootWorker.Handle()
	c.Assert(err, gc.ErrorMatches, "GetRebootAction call error!")
}

func (s *RebootSuite) TestHandleShouldDoNothing(c *gc.C) {
	rebootWorker := reboot.NewRebootStruct(s.rebootState)

	apireboot.PatchFacadeCall(s, s.rebootState, func(name string, p, resp interface{}) error {
			if resp, ok := resp.(*params.RebootActionResults); ok {
				resp.Results = []params.RebootActionResult {
					{ Result: params.ShouldDoNothing },
				}
			}
			return nil
		})

	err := rebootWorker.Handle()
	c.Assert(err, gc.IsNil)
}

func (s *RebootSuite) TestHandleShouldShutdown(c *gc.C) {
	rebootWorker := reboot.NewRebootStruct(s.rebootState)

	apireboot.PatchFacadeCall(s, s.rebootState, func(name string, p, resp interface{}) error {
			if resp, ok := resp.(*params.RebootActionResults); ok {
				resp.Results = []params.RebootActionResult {
					{ Result: params.ShouldShutdown },
				}
			}
			return nil
		})

	err := rebootWorker.Handle()
	c.Assert(err, gc.Equals, worker.ErrShutdownMachine)
}

func (s *RebootSuite) TestHandleShouldReboot(c *gc.C) {
	rebootWorker := reboot.NewRebootStruct(s.rebootState)

	apireboot.PatchFacadeCall(s, s.rebootState, func(name string, p, resp interface{}) error {
			if resp, ok := resp.(*params.RebootActionResults); ok {
				resp.Results = []params.RebootActionResult {
					{ Result: params.ShouldReboot },
				}
			}
			return nil
		})

	err := rebootWorker.Handle()
	c.Assert(err, gc.Equals, worker.ErrRebootMachine)
}
