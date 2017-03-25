// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package oracle

import (
	"strings"
	"sync"
	"time"

	jujuerrors "github.com/juju/errors"

	oci "github.com/juju/go-oracle-cloud/api"
	ociCommon "github.com/juju/go-oracle-cloud/common"
	"github.com/juju/go-oracle-cloud/response"
	ociResponse "github.com/juju/go-oracle-cloud/response"
	"github.com/pkg/errors"

	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/environs/instances"
	"github.com/juju/juju/instance"
	"github.com/juju/juju/network"
	"github.com/juju/juju/status"
)

// oracleInstance type holds the actual running machine
// instance inside the oracle cloud infrastrcture
// this will imlement the instance.Instance interface
type oracleInstance struct {
	// name of the instance, generated after the vm creation
	name string
	// status holds the status of the instance
	status instance.InstanceStatus
	// machine will hold the raw response returned
	// from launching a machine inside
	// the oracle infrastructure
	machine response.Instance
	// arch will hold the architecture information of the instance
	arch *string
	// instType will hold the shape of the instance
	// in a complaint form that juju will understand
	instType *instances.InstanceType
	// mutex used for synchronization between goroutines
	// some methods will require this
	mutex *sync.Mutex
	// env will hold the env that the instance was created from
	env *oracleEnviron
	// machineId is the uuid of the machine
	machineId string
}

// hardwareCharacteristics will return hardware specifications
// based on the instance that is running
func (o *oracleInstance) hardwareCharacteristics() *instance.HardwareCharacteristics {
	if o.arch == nil {
		return nil
	}

	hc := &instance.HardwareCharacteristics{Arch: o.arch}
	if o.instType != nil {
		hc.Mem = &o.instType.Mem
		hc.RootDisk = &o.instType.RootDisk
		hc.CpuCores = &o.instType.CpuCores
	}

	return hc
}

// newInstance returns a new oracleInstance based on the
// instance response of the api and the current juju environment
func newInstance(params response.Instance, env *oracleEnviron) (*oracleInstance, error) {
	if params.Name == "" {
		return nil, errors.New(
			"Instance response does not contain a name",
		)
	}
	//gsamfira: there must be a better way to do this.
	splitMachineName := strings.Split(params.Label, "-")
	machineId := splitMachineName[len(splitMachineName)-1]
    mutex := &sync.Mutex{}
	instance := &oracleInstance{
		name: params.Name,
		status: instance.InstanceStatus{
			Status:  status.Status(params.State),
			Message: "",
		},
		machine:   params,
		mutex:     mutex,
		env:       env,
		machineId: machineId,
	}

	return instance, nil
}

// Id returns a provider generated indentifier for the Instance
func (o *oracleInstance) Id() instance.Id {
	if o.machine.Name != "" {
		return instance.Id(o.machine.Name)
	}

	return instance.Id(o.name)
}

// Status represents the provider specific status for the instance
func (o *oracleInstance) Status() instance.InstanceStatus {
	return o.status
}

// refresh will refresh the instance raw details from the oracle api
// this method is mutex protected
func (o *oracleInstance) refresh() error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	machine, err := o.env.client.InstanceDetails(o.name)
	// if the request failed of any reason
	// we should not update the information and
	// let the old one persist
	if err != nil {
		return err
	}

	o.machine = machine
	return nil
}

// waitForMachineStatus will ping the machine untile the timeout duration is reached or an error appeared
func (o *oracleInstance) waitForMachineStatus(state ociCommon.InstanceState, timeout time.Duration) error {
	// chan user for errors
	errChan := make(chan error)
	// chan used for timeout
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				err := o.refresh()
				if err != nil {
					errChan <- err
					return
				}
				if o.machine.State == ociCommon.StateError {
					errChan <- errors.Errorf(
						"Machine %v entered error state",
						o.machine.Name,
					)
					return
				}
				if o.machine.State == state {
					errChan <- nil
					return
				}
				time.Sleep(10 * time.Second)
			}
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-time.After(timeout):
		done <- true
		return errors.Errorf(
			"Timed out waiting for instance to transition from %v to %v",
			o.machine.State, state,
		)
	}

	return nil
}

// delete will delete the instance and all other things
// that the instances created to functional correctly
// if cleanup is true it will dissasociate the public ips from
// the instance
func (o *oracleInstance) delete(cleanup bool) error {
	if cleanup {
		err := o.disassociatePublicIps(true)
		if err != nil {
			return err
		}
	}

	if err := o.env.client.DeleteInstance(o.name); err != nil {
		return jujuerrors.Trace(err)
	}

	if cleanup {
		// Wait for instance to be deleted. The oracle API does not allow us to
		// delete a security list if there is still a VM associated with it.
		iteration := 0
		for {
			if instance, err := o.env.client.InstanceDetails(o.name); !oci.IsNotFound(err) {
				if instance.State == ociCommon.StateError {
					logger.Warningf("Instance %s entered error state", o.name)
					break
				}
				if iteration >= 30 && instance.State == ociCommon.StateRunning {
					logger.Warningf("Instance still in running state after %q checks. breaking loop", iteration)
					break
				}
				time.Sleep(1 * time.Second)
				iteration++
				continue
			}
			logger.Debugf("Machine %v successfully deleted", o.name)
			break
		}

		// the VM association is now gone, now we can delete the
		// machine sec list also
		if err := o.env.firewall.DeleteMachineSecList(o.machineId); err != nil {
			return jujuerrors.Trace(err)
		}
	}
	return nil
}

// deletePublicIps from the instance
func (o *oracleInstance) deletePublicIps() error {
	ipAssoc, err := o.publicAddresses()
	if err != nil {
		return err
	}

	for _, ip := range ipAssoc {
		if err := o.env.client.DeleteIpReservation(ip.Reservation); err != nil {
			logger.Errorf("Failed to delete IP: %s", err)
			if oci.IsNotFound(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func (o *oracleInstance) unusedPublicIps() ([]ociResponse.IpReservation, error) {
	filter := []oci.Filter{
		oci.Filter{
			Arg:   "permanent",
			Value: "true",
		},
		oci.Filter{
			Arg:   "used",
			Value: "false",
		},
	}

	res, err := o.env.client.AllIpReservations(filter)
	if err != nil {
		return nil, err
	}

	return res.Result, nil
}

// associatePublicIP returns and allocs a new public ip in the oracle cloud
func (o *oracleInstance) associatePublicIP() error {
	// return all unesed public ips
	unusedIps, err := o.unusedPublicIps()
	if err != nil {
		return err
	}

	// for every unused public ip
	for _, val := range unusedIps {
		// alloc a new ip pool for it
		assocPoolName := ociCommon.NewIPPool(ociCommon.IPPool(val.Name), ociCommon.IPReservationType)
		// create the association for it
		if _, err := o.env.client.CreateIpAssociation(assocPoolName, o.machine.Vcable_id); err != nil {
			if oci.IsBadRequest(err) {
				// the IP probably got allocated after we fetched it
				// from the API. Move on to the next one.
				continue
			}

			return err
		} else {
			//TODO(gsamfira): update IP tags
			return nil
		}
	}

	//no unused IP reservations found. Allocate a new one.
	reservation, err := o.env.client.CreateIpReservation(
		o.machine.Name, ociCommon.PublicIPPool, true, o.machine.Tags)
	if err != nil {
		return err
	}
	// alloc a new ip pool fo it
	assocPoolName := ociCommon.NewIPPool(ociCommon.IPPool(reservation.Name), ociCommon.IPReservationType)
	if _, err := o.env.client.CreateIpAssociation(assocPoolName, o.machine.Vcable_id); err != nil {
		return err
	}

	return nil
}

// dissasociatePublicIps returns an errors if the public ip cannot be removed
// if remove is true then it will delete all ip reservation along side with the public one
func (o *oracleInstance) disassociatePublicIps(remove bool) error {
	// get all public ips in from the cloud
	publicIps, err := o.publicAddresses()
	if err != nil {
		return err
	}

	// range every ip
	for _, ipAssoc := range publicIps {
		// take the reservation info and name from the ip assoication
		reservation := ipAssoc.Reservation
		name := ipAssoc.Name
		if err := o.env.client.DeleteIpAssociation(name); err != nil {
			if oci.IsNotFound(err) {
				continue
			}

			return err
		}

		if remove {
			if err := o.env.client.DeleteIpReservation(reservation); err != nil {
				if oci.IsNotFound(err) {
					return nil
				}
				return err
			}
		}
	}

	return nil
}

// publicAddresses returns a slice of all ip associations that are found in the
// oracle cloud account
// this method is mutex protected
func (o *oracleInstance) publicAddresses() ([]response.IpAssociation, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	filter := []oci.Filter{
		oci.Filter{
			Arg:   "vcable",
			Value: string(o.machine.Vcable_id),
		},
	}

	assoc, err := o.env.client.AllIpAssociations(filter)
	if err != nil {
		return nil, jujuerrors.Trace(err)
	}

	return assoc.Result, nil
}

// Addresses returns a list of hostnames or ip addresses
// associated with the instance.
func (o *oracleInstance) Addresses() ([]network.Address, error) {
	//TODO (gsamfira): also include addresses on vNics
	addresses := []network.Address{}

	ips, err := o.publicAddresses()
	if err != nil {
		return nil, jujuerrors.Trace(err)
	}

	if o.machine.Ip != "" {
		address := network.NewScopedAddress(o.machine.Ip, network.ScopeCloudLocal)
		addresses = append(addresses, address)
	}

	for _, val := range ips {
		address := network.NewScopedAddress(val.Ip, network.ScopePublic)
		addresses = append(addresses, address)
	}

	return addresses, nil
}

// OpenPorts opens the given port ranges on the instance, which
// should have been started with the given machine id.
func (o *oracleInstance) OpenPorts(machineId string, rules []network.IngressRule) error {

	if o.env.Config().FirewallMode() != config.FwInstance {
		return errors.Errorf("invalid firewall mode %q for opening ports on instance",
			o.env.Config().FirewallMode(),
		)
	}

	return o.env.firewall.OpenPortsOnInstance(machineId, rules)
}

// ClosePorts closes the given port ranges on the instance, which
// should have been started with the given machine id.
func (o *oracleInstance) ClosePorts(machineId string, rules []network.IngressRule) error {

	if o.env.Config().FirewallMode() != config.FwInstance {
		return errors.Errorf("invalid firewall mode %q for closing ports on instance",
			o.env.Config().FirewallMode(),
		)
	}

	return o.env.firewall.ClosePortsOnInstance(machineId, rules)
}

// IngressRules returns the set of ingress rules for the instance,
// which should have been applied to the given machine id. The
// rules are returned as sorted by network.SortIngressRules().
// It is expected that there be only one ingress rule result for a given
// port range - the rule's SourceCIDRs will contain all applicable source
// address rules for that port range.
func (o *oracleInstance) IngressRules(machineId string) ([]network.IngressRule, error) {
	return o.env.firewall.MachineIngressRules(machineId)
}
