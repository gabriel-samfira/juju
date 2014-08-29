package reboot

import (
	// "github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/names"

	"github.com/juju/juju/state"
	// "github.com/juju/juju/state/api/params"
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

// func (api *RebootAPI) RequestContainerShutdown(args params.Entities) (params.ErrorResults, error) {
// 	tag := api.auth.GetAuthTag().(type)
// 	if tag != names.MachineTag {
// 		return params.StringResults{}, errors.Errorf("Only machine agents may request a shutdown")
// 	}
// 	return common.ServerError(err), nil
// }
