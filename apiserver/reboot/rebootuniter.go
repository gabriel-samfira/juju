// Copyright 2014 Cloudbase Solutions SRL
// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package reboot

import (
	"github.com/juju/errors"
	"github.com/juju/names"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/state"
	"github.com/juju/juju/state/watcher"
)

// RebootAPI provides access to the Upgrader API facade.
type UniterRebootAPI struct {
	unit      *state.Unit
	machine   *state.Machine
	auth      common.Authorizer
	st        *state.State
	resources *common.Resources
}

// NewRebootAPI creates a new client-side RebootAPI facade.
func NewUniterRebootAPI(
	st *state.State,
	resources *common.Resources,
	auth common.Authorizer,
) (*UniterRebootAPI, error) {
	if !auth.AuthUnitAgent() {
		return nil, common.ErrPerm
	}

	tag := auth.GetAuthTag().(names.UnitTag)

	unit, err := st.Unit(tag.Id())
	if err != nil {
		return nil, common.ErrPerm
	}
	id, err := unit.AssignedMachineId()
	if err != nil {
		return nil, err
	}
	m, err := st.Machine(id)
	if err != nil {
		return nil, err
	}

	return &UniterRebootAPI{
		st:        st,
		machine:   m,
		unit:      unit,
		resources: resources,
		auth:      auth,
	}, nil
}

// WatchForRebootEvent starts a watcher to track if there is a new
// reboot request on the machine ID of the unit agent or any of its
// parents (in case we are a container).
func (r *UniterRebootAPI) WatchForRebootEvent() (params.NotifyWatchResult, error) {
	err := common.ErrPerm

	var result params.NotifyWatchResult
	if r.auth.AuthOwner(r.unit.Tag()) {
		err = nil
		watch, err := r.machine.WatchForRebootEvent()
		if err != nil {
			return params.NotifyWatchResult{}, errors.Trace(err)
		}
		// Consume the initial event. Technically, API
		// calls to Watch 'transmit' the initial event
		// in the Watch response. But NotifyWatchers
		// have no state to transmit.
		if _, ok := <-watch.Changes(); ok {
			result.NotifyWatcherId = r.resources.Register(watch)
			err = nil
		} else {
			err = watcher.MustErr(watch)
		}
	}
	result.Error = common.ServerError(err)
	return result, nil
}

// RequestReebot sets the reboot flag to true for the machine to which this
// unit agent belongs to.
func (r *UniterRebootAPI) RequestReboot() (params.ErrorResult, error) {
	logger.Infof("Got reboot request from: %v", r.unit.Tag())
	err := r.machine.SetRebootFlag(true)
	if err != nil {
		return params.ErrorResult{}, errors.Trace(err)
	}
	return params.ErrorResult{Error: common.ServerError(err)}, nil
}

// ClearReboot clears the reboot flag for the current machine.
// Uniter is now allowed to clear the reboot flag of its machine. That operation
// should only be done by the machine agent right before it reboots
func (r *UniterRebootAPI) ClearReboot() (params.ErrorResult, error) {
	err := common.ErrPerm
	logger.Infof("Got clear reboot request from: %v", r.unit.Tag())
	return params.ErrorResult{Error: common.ServerError(err)}, nil
}

// GetRebootAction gets the reboot flag for the current machine.
// This will be useful to determine if we need to set a reboot flag or not. If our parent
// already requested a reboot, there is no need to trigger another one on out machine
// agent or its containers
func (r *UniterRebootAPI) GetRebootAction() (params.RebootActionResult, error) {
	rAction, err := r.machine.ShouldRebootOrShutdown()
	if err != nil {
		return params.RebootActionResult{Result: params.ShouldDoNothing}, errors.Trace(err)
	}
	return params.RebootActionResult{Result: rAction}, nil
}
