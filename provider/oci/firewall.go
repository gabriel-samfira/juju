package oci

import (
	// "github.com/juju/juju/environs"
	"github.com/juju/juju/network"
	// "github.com/juju/juju/provider/oci/common"
)

func (e *Environ) OpenPorts(rules []network.IngressRule) error {
	return nil
}

func (e *Environ) ClosePorts(rules []network.IngressRule) error {
	return nil
}

func (e *Environ) IngressRules() ([]network.IngressRule, error) {
	return nil, nil
}
