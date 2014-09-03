// Copyright 2014 Cloudbase Solutions SRL
// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package reboot

import (
	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/names"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/state"
	"github.com/juju/juju/state/watcher"
)

var logger = loggo.GetLogger("juju.apiserver.reboot")

// RebootAPI provides access to the Upgrader API facade.
type RebootAPI struct {
	auth      common.Authorizer
	st        *state.State
	machine   *state.Machine
	resources *common.Resources
}

func init() {
	common.RegisterStandardFacade("Reboot", 0, NewRebootAPI)
}

type Rebooter interface {
	WatchForRebootEvent() (params.NotifyWatchResult, error)
	RequestReboot() (params.ErrorResult, error)
	ClearReboot() (params.ErrorResult, error)
	GetRebootAction() (params.RebootActionResult, error)
}

// NewRebootAPI creates a new client-side RebootAPI facade.
func NewRebootAPI(
	st *state.State,
	resources *common.Resources,
	auth common.Authorizer,
) (Rebooter, error) {
	if !auth.AuthMachineAgent() {
		return nil, common.ErrPerm
	}

	tag := auth.GetAuthTag().(names.MachineTag)
	machine, err := st.Machine(tag.Id())
	if err != nil {
		return nil, err
	}
	return &RebootAPI{
		st:        st,
		machine:   machine,
		resources: resources,
		auth:      auth,
	}, nil
}

// WatchForRebootEvent starts a watcher to track if there is a new
// reboot request on the machines ID or any of its parents (in case we are a container).
func (r *RebootAPI) WatchForRebootEvent() (params.NotifyWatchResult, error) {
	err := common.ErrPerm

	var result params.NotifyWatchResult

	if r.auth.AuthOwner(r.machine.Tag()) {
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

// RequestReebot sets the reboot flag to true for the current machine.
func (r *RebootAPI) RequestReboot() (params.ErrorResult, error) {
	logger.Infof("Got reboot request from: %v", r.machine.Tag())
	err := r.machine.SetRebootFlag(true)
	if err != nil {
		return params.ErrorResult{}, errors.Trace(err)
	}
	return params.ErrorResult{Error: common.ServerError(err)}, nil
}

// ClearReboot clears the reboot flag for the current machine.
func (r *RebootAPI) ClearReboot() (params.ErrorResult, error) {
	logger.Infof("Got clear reboot request from: %v", r.machine.Tag())
	err := r.machine.SetRebootFlag(false)
	if err != nil {
		return params.ErrorResult{}, errors.Trace(err)
	}
	return params.ErrorResult{Error: common.ServerError(err)}, nil
}

// GetRebootAction gets the reboot flag for the current machine.
func (r *RebootAPI) GetRebootAction() (params.RebootActionResult, error) {
	rAction, err := r.machine.ShouldRebootOrShutdown()
	if err != nil {
		return params.RebootActionResult{Result: params.ShouldDoNothing}, errors.Trace(err)
	}
	return params.RebootActionResult{Result: rAction}, nil
}
