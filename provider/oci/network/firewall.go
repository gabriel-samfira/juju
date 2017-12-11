// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package network

import (
	"github.com/juju/juju/environs"
	"github.com/juju/juju/network"
)

type Firewall struct {}

var _ environs.Firewaller = (*Firewall)(nil)

func (f Firewall) OpenPorts(rules []network.IngressRule) error {
	return nil
}

func (f Firewall) ClosePorts(rules []network.IngressRule) error {
	return nil
}

func (f Firewall) IngressRules() ([]network.IngressRule, error) {
	return nil, nil
}