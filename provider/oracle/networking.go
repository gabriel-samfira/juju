// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package oracle

import (
	oci "github.com/juju/go-oracle-cloud/api"
	ociResponse "github.com/juju/go-oracle-cloud/response"

	"github.com/juju/errors"
	"github.com/juju/juju/environs"
	// "github.com/juju/juju/cloudconfig/cloudinit"
)

var _ environs.NetworkingEnviron = (*oracleEnviron)(nil)

// Only ubuntu for now. There is no CentOS image in the oracle
// compute marketplace
var ubuntuInterfaceTemplate = `
auto %s
iface %s inet dhcp
`

const (
	// defaultNicName is the default network internet card name inside a vm
	defaultNicName = "eth0"
	// nicPrefix si the default network internet card prefix name inside a vm
	nicPrefix = "eth"
	// interfacesConfigDir default path of interfaces.d directory
	interfacesConfigDir = `/etc/network/interfaces.d`
)

// getIPExchangeAndNetworks return all ip networks that are tied with
// the ip exchange networks
func (e *oracleEnviron) getIPExchangesAndNetworks() (map[string][]ociResponse.IpNetwork, error) {
	logger.Infof("Getting ip exchanges and networks")
	ret := map[string][]ociResponse.IpNetwork{}
	exchanges, err := e.client.AllIpNetworkExchanges(nil)
	if err != nil {
		return ret, err
	}
	ipNets, err := e.client.AllIpNetworks(nil)
	if err != nil {
		return ret, err
	}
	for _, val := range exchanges.Result {
		ret[val.Name] = []ociResponse.IpNetwork{}
	}
	for _, val := range ipNets.Result {
		if val.IpNetworkExchange == nil {
			continue
		}
		if _, ok := ret[*val.IpNetworkExchange]; ok {
			ret[*val.IpNetworkExchange] = append(ret[*val.IpNetworkExchange], val)
		}
	}
	return ret, nil
}

// DeleteMachineVnicSet will delete the machine virtual nic and all acl
// rules that are bound with it
func (o *oracleEnviron) DeleteMachineVnicSet(machineId string) error {
	if err := o.RemoveACLAndRules(machineId); err != nil {
		return errors.Trace(err)
	}
	name := o.client.ComposeName(o.namespace.Value(machineId))
	err := o.client.DeleteVnicSet(name)
	if err != nil {
		if !oci.IsNotFound(err) {
			return errors.Trace(err)
		}
	}
	return nil
}

func (o *oracleEnviron) ensureVnicSet(machineId string, tags []string) (ociResponse.VnicSet, error) {
	acl, err := o.CreateDefaultACLAndRules(machineId)
	if err != nil {
		return ociResponse.VnicSet{}, errors.Trace(err)
	}
	name := o.client.ComposeName(o.namespace.Value(machineId))
	details, err := o.client.VnicSetDetails(name)
	if err != nil {
		if !oci.IsNotFound(err) {
			return ociResponse.VnicSet{}, errors.Trace(err)
		}
		logger.Debugf("Creating vnic set %q", name)
		vnicSetParams := oci.VnicSetParams{
			AppliedAcls: []string{
				acl.Name,
			},
			Description: "Juju created vnic set",
			Name:        name,
			Tags:        tags,
		}
		details, err := o.client.CreateVnicSet(vnicSetParams)
		if err != nil {
			return ociResponse.VnicSet{}, errors.Trace(err)
		}
		return details, nil
	}
	return details, nil
}
