package reboot

import (
	"github.com/juju/errors"
	"github.com/juju/names"

	"github.com/juju/juju/api/base"
	"github.com/juju/juju/api/watcher"
	"github.com/juju/juju/apiserver/params"
)

// State provides access to an upgrader worker's view of the state.
type State struct {
	machineTag names.Tag
	facade     base.FacadeCaller
}

// NewState returns a version of the state that provides functionality
// required by the upgrader worker.
func NewState(caller base.APICaller, machineTag names.MachineTag) *State {

	return &State{
		facade:     base.NewFacadeCaller(caller, "Reboot"),
		machineTag: machineTag,
	}
}

func (st *State) WatchForRebootEvent() (watcher.NotifyWatcher, error) {
	var result params.NotifyWatchResult

	err := st.facade.FacadeCall("WatchForRebootEvent", nil, &result)
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	w := watcher.NewNotifyWatcher(st.facade.RawAPICaller(), result)
	return w, nil
}

func (st *State) RequestReboot() error {
	var results params.ErrorResults
	args := params.Entities{
		Entities: []params.Entity{{Tag: st.machineTag.String()}},
	}

	err := st.facade.FacadeCall("RequestReboot", args, &results)
	if err != nil {
		return errors.Trace(err)
	}

	if len(results.Results) != 1 {
		return errors.Errorf("expected 1 result, got %d", len(results.Results))
	}

	if results.Results[0].Error != nil {
		return errors.Trace(results.Results[0].Error)
	}
	return nil
}

func (st *State) ClearReboot() error {
	var results params.ErrorResults
	args := params.Entities{
		Entities: []params.Entity{{Tag: st.machineTag.String()}},
	}

	err := st.facade.FacadeCall("ClearReboot", args, &results)
	if err != nil {
		return errors.Trace(err)
	}

	if len(results.Results) != 1 {
		return errors.Errorf("expected 1 result, got %d", len(results.Results))
	}

	if results.Results[0].Error != nil {
		return errors.Trace(results.Results[0].Error)
	}

	return nil
}

func (st *State) GetRebootAction() (params.RebootAction, error) {
	var results params.RebootActionResults
	args := params.Entities{
		Entities: []params.Entity{{Tag: st.machineTag.String()}},
	}

	err := st.facade.FacadeCall("GetRebootAction", args, &results)
	if err != nil {
		return params.ShouldDoNothing, err
	}
	if len(results.Results) != 1 {
		return params.ShouldDoNothing, errors.Errorf("expected 1 result, got %d", len(results.Results))
	}

	if results.Results[0].Error != nil {
		return params.ShouldDoNothing, errors.Trace(results.Results[0].Error)
	}

	return results.Results[0].Result, nil
}
