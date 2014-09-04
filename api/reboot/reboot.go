package reboot

import (
	"github.com/juju/juju/api/base"
	"github.com/juju/juju/api/watcher"
	"github.com/juju/juju/apiserver/params"
)

// State provides access to an upgrader worker's view of the state.
type State struct {
	facade base.FacadeCaller
}

// NewState returns a version of the state that provides functionality
// required by the upgrader worker.
func NewState(caller base.APICaller) *State {
	return &State{base.NewFacadeCaller(caller, "Reboot")}
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
	var result params.ErrorResult

	err := st.facade.FacadeCall("RequestReboot", nil, &result)
	if err != nil {
		return err
	}
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (st *State) ClearReboot() error {
	var result params.ErrorResult

	err := st.facade.FacadeCall("ClearReboot", nil, &result)
	if err != nil {
		return err
	}
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (st *State) GetRebootAction() (params.RebootAction, error) {
	var result params.RebootActionResult

	err := st.facade.FacadeCall("GetRebootAction", nil, &result)
	if err != nil {
		return params.ShouldDoNothing, err
	}

	return result.Result, nil
}
