package common

import (
	"github.com/juju/errors"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/state"
	"github.com/juju/names"
)

type RebootRequester struct {
	st   state.EntityFinder
	auth GetAuthFunc
}

func NewRebootRequester(st state.EntityFinder, auth GetAuthFunc) *RebootRequester {
	return &LifeGetter{
		st:   st,
		auth: auth,
	}
}

func (lg *LifeGetter) oneRequest(tag names.Tag) error {
	entity0, err := lg.st.FindEntity(tag)
	if err != nil {
		return "", err
	}
	entity, ok := entity0.(state.RebootFlagSetter)
	if !ok {
		return "", NotSupportedError(tag, "request reboot")
	}
	return entity.SetRebootFlag(true)
}

func (lg *LifeGetter) RequestReboot(args params.Entities) (params.ErrorResults, error) {
	result := params.ErrorResults{
		Results: make([]params.ErrorResult, len(args.Entities)),
	}
	if len(args.Entities) == 0 {
		return result, nil
	}
	auth, err := lg.auth()
	if err != nil {
		return params.ErrorResults{}, errors.Trace(err)
	}
	for i, entity := range args.Entities {
		tag, err := names.ParseTag(entity.Tag)
		if err != nil {
			result.Results[i].Error = ServerError(ErrPerm)
			continue
		}
		err = ErrPerm
		if !auth(tag) {
			result.Results[i].Error = ServerError(err)
		}
	}
	return result, nil
}
