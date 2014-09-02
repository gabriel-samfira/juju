// Copyright 2014 Cloudbase Solutions SRL
// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package reboot_test

import (
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/apiserver/reboot"
	apiservertesting "github.com/juju/juju/apiserver/testing"
	"github.com/juju/juju/instance"
	jujutesting "github.com/juju/juju/juju/testing"
	"github.com/juju/juju/state"
	statetesting "github.com/juju/juju/state/testing"
)

type machines struct {
	machine    *state.Machine
	authorizer apiservertesting.FakeAuthorizer
	resources  *common.Resources

	rebootAPI *reboot.RebootAPI

	args params.Entities
}

type rebootSuite struct {
	jujutesting.JujuConnSuite

	machine         *machines
	container       *machines
	nestedContainer *machines
}

var _ = gc.Suite(&rebootSuite{})

func (s *rebootSuite) setUpMachine(c *gc.C, machine *state.Machine) *machines {
	// Create a FakeAuthorizer so we can check permissions,
	// set up assuming we logged in as a machine agent.
	authorizer := apiservertesting.FakeAuthorizer{
		Tag: machine.Tag(),
	}

	resources := common.NewResources()

	rebootAPI, err := reboot.NewRebootAPI(s.State, resources, authorizer)
	c.Assert(err, gc.IsNil)

	args := params.Entities{Entities: []params.Entity{
		{Tag: machine.Tag().String()},
	}}

	return &machines{machine, authorizer, resources, rebootAPI, args}
}

func (s *rebootSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)
	var err error

	template := state.MachineTemplate{
		Series: "quantal",
		Jobs:   []state.MachineJob{state.JobHostUnits},
	}

	machine, err := s.State.AddMachine("quantal", state.JobHostUnits)
	c.Assert(err, gc.IsNil)

	container, err := s.State.AddMachineInsideMachine(template, machine.Id(), instance.LXC)
	c.Assert(err, gc.IsNil)

	nestedContainer, err := s.State.AddMachineInsideMachine(template, container.Id(), instance.KVM)
	c.Assert(err, gc.IsNil)

	s.machine = s.setUpMachine(c, machine)
	s.container = s.setUpMachine(c, container)
	s.nestedContainer = s.setUpMachine(c, nestedContainer)
}

func (s *rebootSuite) TearDownTest(c *gc.C) {
	if s.machine.resources != nil {
		s.machine.resources.StopAll()
	}

	if s.container.resources != nil {
		s.container.resources.StopAll()
	}

	if s.nestedContainer.resources != nil {
		s.nestedContainer.resources.StopAll()
	}
	s.JujuConnSuite.TearDownTest(c)
}

func (s *rebootSuite) TestWatchForRebootEvent(c *gc.C) {
	result, err := s.machine.rebootAPI.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	c.Check(result.NotifyWatcherId, gc.Not(gc.Equals), "")
	c.Check(result.Error, gc.IsNil)

	resource := s.machine.resources.Get(result.NotifyWatcherId)
	c.Check(resource, gc.NotNil)

	w := resource.(state.NotifyWatcher)
	wc := statetesting.NewNotifyWatcherC(c, s.State, w)
	wc.AssertNoChange()

	err = s.machine.machine.SetRebootFlag(true)
	c.Assert(err, gc.IsNil)

	wc.AssertOneChange()

	statetesting.AssertStop(c, w)
	wc.AssertClosed()
}

func (s *rebootSuite) TestRequestReboot(c *gc.C) {
	result, err := s.machine.rebootAPI.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	c.Check(result.NotifyWatcherId, gc.Not(gc.Equals), "")
	c.Check(result.Error, gc.IsNil)

	resource := s.machine.resources.Get(result.NotifyWatcherId)
	c.Check(resource, gc.NotNil)

	w := resource.(state.NotifyWatcher)
	wc := statetesting.NewNotifyWatcherC(c, s.State, w)
	wc.AssertNoChange()

	errResult, err := s.machine.rebootAPI.RequestReboot(s.machine.args)
	c.Assert(err, gc.IsNil)
	c.Assert(errResult, gc.DeepEquals, params.ErrorResults{
		Results: []params.ErrorResult{
			{Error: nil},
		}})

	wc.AssertOneChange()

	res, err := s.machine.rebootAPI.GetRebootAction(s.machine.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldReboot},
		}})

	statetesting.AssertStop(c, w)
	wc.AssertClosed()
}

func (s *rebootSuite) TestClearReboot(c *gc.C) {
	result, err := s.machine.rebootAPI.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	c.Check(result.NotifyWatcherId, gc.Not(gc.Equals), "")
	c.Check(result.Error, gc.IsNil)

	resource := s.machine.resources.Get(result.NotifyWatcherId)
	c.Check(resource, gc.NotNil)

	w := resource.(state.NotifyWatcher)
	wc := statetesting.NewNotifyWatcherC(c, s.State, w)
	wc.AssertNoChange()

	errResult, err := s.machine.rebootAPI.RequestReboot(s.machine.args)
	c.Assert(err, gc.IsNil)
	c.Assert(errResult, gc.DeepEquals, params.ErrorResults{
		Results: []params.ErrorResult{
			{Error: nil},
		},
	})

	wc.AssertOneChange()

	res, err := s.machine.rebootAPI.GetRebootAction(s.machine.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldReboot},
		}})

	errResult, err = s.machine.rebootAPI.ClearReboot(s.machine.args)
	c.Assert(err, gc.IsNil)
	c.Assert(errResult, gc.DeepEquals, params.ErrorResults{
		Results: []params.ErrorResult{
			{Error: nil},
		},
	})

	res, err = s.machine.rebootAPI.GetRebootAction(s.machine.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldDoNothing},
		}})

	statetesting.AssertStop(c, w)
	wc.AssertClosed()
}

func (s *rebootSuite) TestWatchForRebootEventFromContainer(c *gc.C) {
	// Watcher for the machine
	resultMachine, err := s.machine.rebootAPI.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	c.Check(resultMachine.NotifyWatcherId, gc.Not(gc.Equals), "")
	c.Check(resultMachine.Error, gc.IsNil)

	resourceMachine := s.machine.resources.Get(resultMachine.NotifyWatcherId)
	c.Check(resourceMachine, gc.NotNil)

	w := resourceMachine.(state.NotifyWatcher)
	defer statetesting.AssertStop(c, w)

	wc := statetesting.NewNotifyWatcherC(c, s.State, w)
	wc.AssertNoChange()

	// Watcher for the container
	resultContainer, err := s.container.rebootAPI.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	c.Check(resultContainer.NotifyWatcherId, gc.Not(gc.Equals), "")
	c.Check(resultContainer.Error, gc.IsNil)

	resourceContainer := s.container.resources.Get(resultContainer.NotifyWatcherId)
	c.Check(resourceContainer, gc.NotNil)

	wContainer := resourceContainer.(state.NotifyWatcher)
	defer statetesting.AssertStop(c, wContainer)

	wcContainer := statetesting.NewNotifyWatcherC(c, s.State, wContainer)
	wcContainer.AssertNoChange()

	// Watcher for the nestedContainer
	resultNestedContainer, err := s.nestedContainer.rebootAPI.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	c.Check(resultNestedContainer.NotifyWatcherId, gc.Not(gc.Equals), "")
	c.Check(resultNestedContainer.Error, gc.IsNil)

	resourceNestedContainer := s.nestedContainer.resources.Get(resultNestedContainer.NotifyWatcherId)
	c.Check(resourceNestedContainer, gc.NotNil)

	wNestedContainer := resourceNestedContainer.(state.NotifyWatcher)
	defer statetesting.AssertStop(c, wNestedContainer)

	wcNestedContainer := statetesting.NewNotifyWatcherC(c, s.State, wNestedContainer)
	wcNestedContainer.AssertNoChange()

	// Request reboot on the root machine: all machines should see it
	// machine should reboot
	// container should shutdown
	// nested container should shutdown
	errResult, err := s.machine.rebootAPI.RequestReboot(s.machine.args)
	c.Assert(err, gc.IsNil)
	c.Assert(errResult, gc.DeepEquals, params.ErrorResults{
		Results: []params.ErrorResult{
			{Error: nil},
		}})

	wc.AssertOneChange()
	wcContainer.AssertOneChange()
	wcNestedContainer.AssertOneChange()

	res, err := s.machine.rebootAPI.GetRebootAction(s.machine.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldReboot},
		}})

	res, err = s.container.rebootAPI.GetRebootAction(s.container.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldShutdown},
		}})

	res, err = s.nestedContainer.rebootAPI.GetRebootAction(s.nestedContainer.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldShutdown},
		}})

	errResult, err = s.machine.rebootAPI.ClearReboot(s.machine.args)
	c.Assert(err, gc.IsNil)
	c.Assert(errResult, gc.DeepEquals, params.ErrorResults{
		Results: []params.ErrorResult{
			{Error: nil},
		},
	})

	wc.AssertOneChange()
	wcContainer.AssertOneChange()
	wcNestedContainer.AssertOneChange()

	// Request reboot on the container: container and nested container should see it
	// machine should do nothing
	// container should reboot
	// nested container should shutdown
	errResult, err = s.container.rebootAPI.RequestReboot(s.container.args)
	c.Assert(err, gc.IsNil)
	c.Assert(errResult, gc.DeepEquals, params.ErrorResults{
		Results: []params.ErrorResult{
			{Error: nil},
		}})

	wc.AssertNoChange()
	wcContainer.AssertOneChange()
	wcNestedContainer.AssertOneChange()

	res, err = s.machine.rebootAPI.GetRebootAction(s.machine.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldDoNothing},
		}})

	res, err = s.container.rebootAPI.GetRebootAction(s.container.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldReboot},
		}})

	res, err = s.nestedContainer.rebootAPI.GetRebootAction(s.nestedContainer.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldShutdown},
		}})

	errResult, err = s.container.rebootAPI.ClearReboot(s.container.args)
	c.Assert(err, gc.IsNil)
	c.Assert(errResult, gc.DeepEquals, params.ErrorResults{
		Results: []params.ErrorResult{
			{Error: nil},
		},
	})

	wc.AssertNoChange()
	wcContainer.AssertOneChange()
	wcNestedContainer.AssertOneChange()

	// Request reboot on the container: container and nested container should see it
	// machine should do nothing
	// container should do nothing
	// nested container should reboot
	errResult, err = s.nestedContainer.rebootAPI.RequestReboot(s.nestedContainer.args)
	c.Assert(err, gc.IsNil)
	c.Assert(errResult, gc.DeepEquals, params.ErrorResults{
		Results: []params.ErrorResult{
			{Error: nil},
		}})

	wc.AssertNoChange()
	wcContainer.AssertNoChange()
	wcNestedContainer.AssertOneChange()

	res, err = s.machine.rebootAPI.GetRebootAction(s.machine.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldDoNothing},
		}})

	res, err = s.container.rebootAPI.GetRebootAction(s.container.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldDoNothing},
		}})

	res, err = s.nestedContainer.rebootAPI.GetRebootAction(s.nestedContainer.args)
	c.Assert(err, gc.IsNil)
	c.Assert(res, gc.DeepEquals, params.RebootActionResults{
		Results: []params.RebootActionResult{
			{Result: params.ShouldReboot},
		}})

	errResult, err = s.nestedContainer.rebootAPI.ClearReboot(s.nestedContainer.args)
	c.Assert(err, gc.IsNil)
	c.Assert(errResult, gc.DeepEquals, params.ErrorResults{
		Results: []params.ErrorResult{
			{Error: nil},
		},
	})

	wc.AssertNoChange()
	wcContainer.AssertNoChange()
	wcNestedContainer.AssertOneChange()

	// Stop watchers
	statetesting.AssertStop(c, w)
	wc.AssertClosed()

	statetesting.AssertStop(c, wContainer)
	wcContainer.AssertClosed()

	statetesting.AssertStop(c, wNestedContainer)
	wcNestedContainer.AssertClosed()
}
