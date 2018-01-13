// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package network

import (
	"github.com/juju/juju/environs"
	"github.com/juju/juju/network"

	"github.com/juju/juju/provider/oci/common"
)

type Firewall struct {
	cli common.ApiClient
}

var _ environs.Firewaller = (*Firewall)(nil)

func NewFirewaller(cli common.ApiClient) (environs.Firewaller, error) {
	return &Firewall{
		cli: cli,
	}, nil
}

func (f Firewall) OpenPorts(rules []network.IngressRule) error {
	return nil
}

func (f Firewall) ClosePorts(rules []network.IngressRule) error {
	return nil
}

func (f Firewall) IngressRules() ([]network.IngressRule, error) {
	return nil, nil
}
