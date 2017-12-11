// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package oci

import (
	"github.com/juju/juju/environs/instances"
	"github.com/juju/juju/instance"
	"github.com/juju/juju/network"
)

type ociInstance struct {
	arch     *string
	instType *instances.InstanceType
	env      *Environ
}


var _ instance.Instance = (*ociInstance)(nil)

// Id implements instance.Instance.
func (o *ociInstance) Id() instance.Id {
	return ""
}

// Status implements instance.Instance.
func (o *ociInstance) Status() instance.InstanceStatus {
	return instance.InstanceStatus{}
}

// Addresses implements instance.Instance.
func (o *ociInstance) Addresses() ([]network.Address, error) {
	return nil, nil
}