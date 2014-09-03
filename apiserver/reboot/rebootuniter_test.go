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

type unit struct {
	unit       *state.Unit
	authorizer apiservertesting.FakeAuthorizer
	resources  *common.Resources

	rebootAPI reboot.Rebooter
}

type uniterMachine struct {
	unit       *unit
	machine    *state.Machine
	authorizer apiservertesting.FakeAuthorizer
	resources  *common.Resources

	rebootAPI reboot.Rebooter
}

type uniterRebootSuite struct {
	jujutesting.JujuConnSuite

	machine         *uniterMachine
	container       *uniterMachine
	nestedContainer *uniterMachine
}

var _ = gc.Suite(&uniterRebootSuite{})

func (s *uniterRebootSuite) setUpMachine(c *gc.C, machine *state.Machine, serviceName string) *uniterMachine {
	// Create a FakeAuthorizer so we can check permissions,
	// set up assuming we logged in as a machine agent.
	authorizer := apiservertesting.FakeAuthorizer{
		Tag: machine.Tag(),
	}

	resources := common.NewResources()

	rebootAPI, err := reboot.NewRebootFacade(s.State, resources, authorizer)
	c.Assert(err, gc.IsNil)

	unit := s.setUpUnit(c, machine, serviceName)

	return &uniterMachine{
		machine:    machine,
		authorizer: authorizer,
		resources:  resources,
		rebootAPI:  rebootAPI,
		unit:       unit,
	}
}

func (s *uniterRebootSuite) setUpUnit(c *gc.C, machine *state.Machine, serviceName string) *unit {
	svc := s.AddTestingService(c, serviceName, s.AddTestingCharm(c, serviceName))
	rawUnit, err := svc.AddUnit()
	c.Assert(err, gc.IsNil)

	// Assign the unit to the machine.
	err = rawUnit.AssignToMachine(machine)
	c.Assert(err, gc.IsNil)

	// The default auth is as the unit agent
	authorizer := apiservertesting.FakeAuthorizer{
		Tag: rawUnit.Tag(),
	}

	resources := common.NewResources()

	rebootAPI, err := reboot.NewRebootFacade(s.State, resources, authorizer)
	c.Assert(err, gc.IsNil)

	return &unit{rawUnit, authorizer, resources, rebootAPI}
}

func (s *uniterRebootSuite) SetUpTest(c *gc.C) {
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

	s.machine = s.setUpMachine(c, machine, "wordpress")
	s.container = s.setUpMachine(c, container, "mysql")
	s.nestedContainer = s.setUpMachine(c, nestedContainer, "varnish")
}

func (s *uniterRebootSuite) TearDownTest(c *gc.C) {
	if s.machine != nil && s.machine.resources != nil {
		s.machine.resources.StopAll()
	}

	if s.container != nil && s.container.resources != nil {
		s.container.resources.StopAll()
	}

	if s.nestedContainer != nil && s.nestedContainer.resources != nil {
		s.nestedContainer.resources.StopAll()
	}
	s.JujuConnSuite.TearDownTest(c)
}

func (s *uniterRebootSuite) TestWatchForRebootEventFromUniter(c *gc.C) {
	result, err := s.machine.unit.rebootAPI.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	c.Check(result.NotifyWatcherId, gc.Not(gc.Equals), "")
	c.Check(result.Error, gc.IsNil)

	resource := s.machine.unit.resources.Get(result.NotifyWatcherId)
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

func (s *uniterRebootSuite) TestRequestRebootFromUniter(c *gc.C) {
	// Watch for reboot event on the machine.
	result, err := s.machine.rebootAPI.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	c.Check(result.NotifyWatcherId, gc.Not(gc.Equals), "")
	c.Check(result.Error, gc.IsNil)

	resource := s.machine.resources.Get(result.NotifyWatcherId)
	c.Check(resource, gc.NotNil)

	w := resource.(state.NotifyWatcher)
	wc := statetesting.NewNotifyWatcherC(c, s.State, w)
	wc.AssertNoChange()

	// Request reboot from unit inside machine.
	errResult, err := s.machine.unit.rebootAPI.RequestReboot()
	c.Assert(err, gc.IsNil)
	c.Assert(errResult.Error, gc.IsNil)

	wc.AssertOneChange()

	res, err := s.machine.rebootAPI.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(res.Result, gc.Equals, params.ShouldReboot)

	statetesting.AssertStop(c, w)
	wc.AssertClosed()
}

func (s *uniterRebootSuite) TestClearRebootFromUniter(c *gc.C) {
	result, err := s.machine.rebootAPI.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	c.Check(result.NotifyWatcherId, gc.Not(gc.Equals), "")
	c.Check(result.Error, gc.IsNil)

	resource := s.machine.resources.Get(result.NotifyWatcherId)
	c.Check(resource, gc.NotNil)

	w := resource.(state.NotifyWatcher)
	wc := statetesting.NewNotifyWatcherC(c, s.State, w)
	wc.AssertNoChange()

	errResult, err := s.machine.unit.rebootAPI.RequestReboot()
	c.Assert(err, gc.IsNil)
	c.Assert(errResult.Error, gc.IsNil)

	wc.AssertOneChange()

	res, err := s.machine.rebootAPI.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(res.Result, gc.Equals, params.ShouldReboot)

	// Uniter is not allowed to clear reboot flag
	errResult, err = s.machine.unit.rebootAPI.ClearReboot()
	c.Assert(err, gc.IsNil)
	c.Assert(errResult.Error, gc.DeepEquals, common.ServerError(common.ErrPerm))

	statetesting.AssertStop(c, w)
	wc.AssertClosed()
}

func (s *uniterRebootSuite) TestUniterTriggersCorrectReboot(c *gc.C) {
	// Watcher for the machine
	resultMachine, err := s.machine.rebootAPI.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	c.Check(resultMachine.NotifyWatcherId, gc.Not(gc.Equals), "")
	c.Check(resultMachine.Error, gc.IsNil)

	resourceMachine := s.machine.resources.Get(resultMachine.NotifyWatcherId)
	c.Check(resourceMachine, gc.NotNil)

	w := resourceMachine.(state.NotifyWatcher)
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
	wcNestedContainer := statetesting.NewNotifyWatcherC(c, s.State, wNestedContainer)
	wcNestedContainer.AssertNoChange()

	// Request reboot on the root machine: all machines should see it
	// machine should reboot
	// container should shutdown
	// nested container should shutdown
	errResult, err := s.machine.unit.rebootAPI.RequestReboot()
	c.Assert(err, gc.IsNil)
	c.Assert(errResult.Error, gc.IsNil)

	wc.AssertOneChange()
	wcContainer.AssertOneChange()
	wcNestedContainer.AssertOneChange()

	res, err := s.machine.rebootAPI.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(res.Result, gc.Equals, params.ShouldReboot)

	res, err = s.container.rebootAPI.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(res.Result, gc.Equals, params.ShouldShutdown)

	res, err = s.nestedContainer.rebootAPI.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(res.Result, gc.Equals, params.ShouldShutdown)

	errResult, err = s.machine.rebootAPI.ClearReboot()
	c.Assert(err, gc.IsNil)
	c.Assert(errResult.Error, gc.IsNil)

	wc.AssertOneChange()
	wcContainer.AssertOneChange()
	wcNestedContainer.AssertOneChange()

	// Request reboot on the container: container and nested container should see it
	// machine should do nothing
	// container should reboot
	// nested container should shutdown
	errResult, err = s.container.unit.rebootAPI.RequestReboot()
	c.Assert(err, gc.IsNil)
	c.Assert(errResult.Error, gc.IsNil)

	wc.AssertNoChange()
	wcContainer.AssertOneChange()
	wcNestedContainer.AssertOneChange()

	res, err = s.machine.rebootAPI.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(res.Result, gc.Equals, params.ShouldDoNothing)

	res, err = s.container.rebootAPI.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(res.Result, gc.Equals, params.ShouldReboot)

	res, err = s.nestedContainer.rebootAPI.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(res.Result, gc.Equals, params.ShouldShutdown)

	errResult, err = s.container.rebootAPI.ClearReboot()
	c.Assert(err, gc.IsNil)
	c.Assert(errResult.Error, gc.IsNil)

	wc.AssertNoChange()
	wcContainer.AssertOneChange()
	wcNestedContainer.AssertOneChange()

	// Request reboot on the container: container and nested container should see it
	// machine should do nothing
	// container should do nothing
	// nested container should reboot
	errResult, err = s.nestedContainer.unit.rebootAPI.RequestReboot()
	c.Assert(err, gc.IsNil)
	c.Assert(errResult.Error, gc.IsNil)

	wc.AssertNoChange()
	wcContainer.AssertNoChange()
	wcNestedContainer.AssertOneChange()

	res, err = s.machine.rebootAPI.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(res.Result, gc.Equals, params.ShouldDoNothing)

	res, err = s.container.rebootAPI.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(res.Result, gc.Equals, params.ShouldDoNothing)

	res, err = s.nestedContainer.rebootAPI.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(res.Result, gc.Equals, params.ShouldReboot)

	errResult, err = s.nestedContainer.rebootAPI.ClearReboot()
	c.Assert(err, gc.IsNil)
	c.Assert(errResult.Error, gc.IsNil)

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
