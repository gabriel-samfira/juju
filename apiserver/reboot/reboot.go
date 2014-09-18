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
	*common.RebootActionGetter
	*common.RebootRequester
	*common.RebootFlagClearer

	auth      common.Authorizer
	st        *state.State
	machine   *state.Machine
	resources *common.Resources
}

func init() {
	common.RegisterStandardFacade("Reboot", 0, NewRebootAPI)
}

// NewRebootAPI creates a new client-side RebootAPI facade.
func NewRebootAPI(
	st *state.State,
	resources *common.Resources,
	auth common.Authorizer,
) (*RebootAPI, error) {
	if !auth.AuthMachineAgent() {
		return nil, common.ErrPerm
	}

	tag := auth.GetAuthTag().(names.MachineTag)
	machine, err := st.Machine(tag.Id())
	if err != nil {
		return nil, err
	}

	canAccess := func() (common.AuthFunc, error) {
		return auth.AuthOwner, nil
	}

	return &RebootAPI{
		RebootActionGetter: common.NewRebootActionGetter(st, canAccess),
		RebootRequester:    common.NewRebootRequester(st, canAccess),
		RebootFlagClearer:  common.NewRebootFlagClearer(st, canAccess),
		st:                 st,
		machine:            machine,
		resources:          resources,
		auth:               auth,
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
