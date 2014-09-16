package reboot_test

import (
	stdtesting "testing"

	// "github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/api"
	"github.com/juju/juju/api/reboot"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/instance"
	"github.com/juju/juju/juju/testing"
	"github.com/juju/juju/state"
	statetesting "github.com/juju/juju/state/testing"
	coretesting "github.com/juju/juju/testing"
)

func TestAll(t *stdtesting.T) {
	coretesting.MgoTestPackage(t)
}

type machineUnit struct {
	machineStateAPI *api.State
	machineSt       *reboot.State
	unitStateAPI    *api.State
	unitSt          *reboot.State
	st              *reboot.State
	machine         *state.Machine
	unit            *state.Unit
}

type machineRebootSuite struct {
	testing.JujuConnSuite

	machine         *machineUnit
	container       *machineUnit
	nestedContainer *machineUnit
}

var _ = gc.Suite(&machineRebootSuite{})

func (s *machineRebootSuite) addMachineUnit(c *gc.C, machineID, serviceName string) *machineUnit {
	var machine *state.Machine
	var err error

	if machineID == "" {
		machine, err = s.State.AddMachine("quantal", state.JobHostUnits)
	} else {
		template := state.MachineTemplate{
			Series: "quantal",
			Jobs:   []state.MachineJob{state.JobHostUnits},
		}
		machine, err = s.State.AddMachineInsideMachine(template, machineID, instance.LXC)
	}
	c.Assert(err, gc.IsNil)
	machinePass, err := utils.RandomPassword()
	c.Assert(err, gc.IsNil)
	err = machine.SetPassword(machinePass)
	c.Assert(err, gc.IsNil)
	err = machine.SetInstanceInfo("i-manager", "fake_nonce", nil, nil, nil)
	c.Assert(err, gc.IsNil)
	machineStateAPI := s.OpenAPIAsMachine(c, machine.Tag(), machinePass, "fake_nonce")
	c.Assert(machineStateAPI, gc.NotNil)
	machineSt := machineStateAPI.Reboot()

	charm := s.AddTestingCharm(c, serviceName)
	service := s.AddTestingService(c, serviceName, charm)

	unit, err := service.AddUnit()
	c.Assert(err, gc.IsNil)
	err = unit.AssignToMachine(machine)
	c.Assert(err, gc.IsNil)
	password, err := utils.RandomPassword()
	c.Assert(err, gc.IsNil)
	err = unit.SetPassword(password)
	c.Assert(err, gc.IsNil)

	unitStateAPI := s.OpenAPIAs(c, unit.Tag(), password)
	unitSt := unitStateAPI.Reboot()

	return &machineUnit{
		machine:         machine,
		unit:            unit,
		machineStateAPI: machineStateAPI,
		machineSt:       machineSt,
		unitStateAPI:    unitStateAPI,
		unitSt:          unitSt,
	}
}

func (s *machineRebootSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)

	s.machine = s.addMachineUnit(c, "", "wordpress")
	s.container = s.addMachineUnit(c, s.machine.machine.Id(), "mysql")
	s.nestedContainer = s.addMachineUnit(c, s.container.machine.Id(), "varnish")
}

func (s *machineRebootSuite) TestWatchForRebootEvent(c *gc.C) {
	w, err := s.machine.machineSt.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, w)
	wc := statetesting.NewNotifyWatcherC(c, s.BackingState, w)
	// Initial event
	wc.AssertOneChange()

	//Trigger reboot event
	err = s.machine.machine.SetRebootFlag(true)
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()
}

func (s *machineRebootSuite) TestWatchForRebootEventFromUnit(c *gc.C) {
	w, err := s.machine.unitSt.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, w)
	wc := statetesting.NewNotifyWatcherC(c, s.BackingState, w)
	// Initial event
	wc.AssertOneChange()

	//Trigger reboot event
	err = s.machine.machine.SetRebootFlag(true)
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()
}

func (s *machineRebootSuite) TestRequestRebootFromUnit(c *gc.C) {
	w, err := s.machine.machineSt.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, w)
	wc := statetesting.NewNotifyWatcherC(c, s.BackingState, w)
	// Initial event
	wc.AssertOneChange()

	//Trigger reboot event
	err = s.machine.unitSt.RequestReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()
}

func (s *machineRebootSuite) TestRequestReboot(c *gc.C) {
	w, err := s.machine.machineSt.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, w)
	wc := statetesting.NewNotifyWatcherC(c, s.BackingState, w)
	// Initial event
	wc.AssertOneChange()

	//Trigger reboot event
	err = s.machine.machineSt.RequestReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()
}

func (s *machineRebootSuite) TestClearReboot(c *gc.C) {
	w, err := s.machine.machineSt.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, w)
	wc := statetesting.NewNotifyWatcherC(c, s.BackingState, w)
	// Initial event
	wc.AssertOneChange()

	//Trigger reboot event
	err = s.machine.machineSt.RequestReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()

	// Clear reboot flag
	err = s.machine.machineSt.ClearReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()
}

func (s *machineRebootSuite) TestClearRebootFromUnit(c *gc.C) {
	w, err := s.machine.machineSt.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, w)
	wc := statetesting.NewNotifyWatcherC(c, s.BackingState, w)
	// Initial event
	wc.AssertOneChange()

	//Trigger reboot event
	err = s.machine.machineSt.RequestReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()

	// Clear reboot flag
	err = s.machine.unitSt.ClearReboot()
	c.Assert(err, gc.ErrorMatches, "permission denied")
	c.Assert(err, jc.Satisfies, params.IsCodeUnauthorized)
	wc.AssertNoChange()
}

func (s *machineRebootSuite) TestGetRebootFlag(c *gc.C) {
	w, err := s.machine.machineSt.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, w)
	wc := statetesting.NewNotifyWatcherC(c, s.BackingState, w)
	// Initial event
	wc.AssertOneChange()

	rAction, err := s.machine.machineSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldDoNothing)

	//Trigger reboot event
	err = s.machine.machineSt.RequestReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()

	rAction, err = s.machine.machineSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldReboot)
}

func (s *machineRebootSuite) TestGetRebootFlagFromUnit(c *gc.C) {
	w, err := s.machine.machineSt.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, w)
	wc := statetesting.NewNotifyWatcherC(c, s.BackingState, w)
	// Initial event
	wc.AssertOneChange()

	rAction, err := s.machine.unitSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldDoNothing)

	//Trigger reboot event
	err = s.machine.unitSt.RequestReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()

	rAction, err = s.machine.unitSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldReboot)
}

func (s *machineRebootSuite) TestWatchForRebootEventFromcontainer(c *gc.C) {
	// Set up watcher on machine
	w, err := s.machine.machineSt.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, w)
	wc := statetesting.NewNotifyWatcherC(c, s.BackingState, w)
	wc.AssertOneChange()

	// Set up watcher on container
	wC1, err := s.container.machineSt.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, wC1)
	wcC1 := statetesting.NewNotifyWatcherC(c, s.BackingState, wC1)
	wcC1.AssertOneChange()

	// Set up watcher on nestedContainer
	wC2, err := s.nestedContainer.machineSt.WatchForRebootEvent()
	c.Assert(err, gc.IsNil)
	defer statetesting.AssertStop(c, wC2)
	wcC2 := statetesting.NewNotifyWatcherC(c, s.BackingState, wC2)
	// Initial event
	wcC2.AssertOneChange()

	// Trigger reboot event from unit on container
	// machine should reboot
	// container should shutdown
	// nestedContainer should shutdown
	err = s.machine.unitSt.RequestReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()
	wcC1.AssertOneChange()
	wcC2.AssertOneChange()

	rAction, err := s.machine.machineSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldReboot)

	rAction, err = s.container.machineSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldShutdown)

	rAction, err = s.nestedContainer.machineSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldShutdown)

	err = s.machine.machineSt.ClearReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertOneChange()
	wcC1.AssertOneChange()
	wcC2.AssertOneChange()

	// Trigger reboot event from unit on container
	// machine should do nothing
	// container should reboot
	// nestedContainer should shutdown
	err = s.container.unitSt.RequestReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertNoChange()
	wcC1.AssertOneChange()
	wcC2.AssertOneChange()

	rAction, err = s.machine.machineSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldDoNothing)

	rAction, err = s.container.machineSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldReboot)

	rAction, err = s.nestedContainer.machineSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldShutdown)

	err = s.container.machineSt.ClearReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertNoChange()
	wcC1.AssertOneChange()
	wcC2.AssertOneChange()

	// Trigger reboot event from unit on container
	// machine should do nothing
	// container should do nothing
	// nestedContainer should reboot
	err = s.nestedContainer.unitSt.RequestReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertNoChange()
	wcC1.AssertNoChange()
	wcC2.AssertOneChange()

	rAction, err = s.machine.machineSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldDoNothing)

	rAction, err = s.container.machineSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldDoNothing)

	rAction, err = s.nestedContainer.machineSt.GetRebootAction()
	c.Assert(err, gc.IsNil)
	c.Assert(rAction, gc.Equals, params.ShouldReboot)

	err = s.nestedContainer.machineSt.ClearReboot()
	c.Assert(err, gc.IsNil)
	wc.AssertNoChange()
	wcC1.AssertNoChange()
	wcC2.AssertOneChange()

}
