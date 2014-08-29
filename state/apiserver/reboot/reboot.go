package reboot

import (
	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/names"

	"github.com/juju/juju/state"
	"github.com/juju/juju/state/api/params"
	"github.com/juju/juju/state/apiserver/common"
)

var logger = loggo.GetLogger("juju.state.apiserver.reboot")

func init() {
	common.RegisterStandardFacade("Reboot", 0, NewRebootAPI)
}

// RebootAPI implements the API used by the reboot worker.
type RebootAPI struct {
	*common.LifeGetter

	st           *state.State
	auth         common.Authorizer
	getCanModify common.GetAuthFunc
	getCanRead   common.GetAuthFunc
}

// NewRebootAPI creates a new instance of the Machiner API.
func NewRebootAPI(st *state.State, resources *common.Resources, authorizer common.Authorizer) (*RebootAPI, error) {
	if !authorizer.AuthMachineAgent() {
		return nil, common.ErrPerm
	}
	getCanModify := func() (common.AuthFunc, error) {
		return authorizer.AuthOwner, nil
	}
	getCanRead := func() (common.AuthFunc, error) {
		return authorizer.AuthOwner, nil
	}
	return &RebootAPI{
		LifeGetter:   common.NewLifeGetter(st, getCanRead),
		st:           st,
		auth:         authorizer,
		getCanModify: getCanModify,
	}, nil
}

func (api *RebootAPI) getUnit(tag names.UnitTag) (*state.Unit, error) {
	return api.st.Unit(tag.Id())
}

func (api *RebootAPI) RequestReboot() (params.ErrorResult, error) {
	logger.Infof("Got reboot request from: %v", api.auth.GetAuthTag())
	var tag names.UnitTag
	// Only unit agents will request a reboot via a hook
	switch t := api.auth.GetAuthTag().(type) {
	case names.UnitTag:
		tag = t
	default:
		return params.ErrorResult{}, errors.Errorf("Only units may request a reboot")
	}

	// Get the unit by tag
	unit, err := api.getUnit(tag)
	if err != nil {
		return params.ErrorResult{}, err
	}
	// Get assigned machine ID
	machineId, err := unit.AssignedMachineId()
	if err != nil {
		return params.ErrorResult{}, err
	}
	// Get machine object
	machine, err := api.st.Machine(machineId)
	if err != nil {
		return params.ErrorResult{}, err
	}

	// Set status to StatusRebooting
	err = machine.SetStatus(params.StatusRebooting, "", nil)

	return params.ErrorResult{Error: common.ServerError(err)}, nil
}

// func (api *RebootAPI) RequestContainerShutdown(args params.Entities) (params.ErrorResults, error) {
// 	tag := api.auth.GetAuthTag().(type)
// 	if tag != names.MachineTag {
// 		return params.StringResults{}, errors.Errorf("Only machine agents may request a shutdown")
// 	}
// 	return common.ServerError(err), nil
// }
