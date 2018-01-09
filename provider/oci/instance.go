// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package oci

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/juju/juju/status"

	"github.com/juju/juju/environs/instances"
	"github.com/juju/juju/instance"
	"github.com/juju/juju/network"

	ociCore "github.com/oracle/oci-go-sdk/core"
)

const (
	// RootDiskSize is the size of the root disk for all instances deployed on OCI
	// This is not configurable. At the time of this writing, there is no way to
	// request a root disk of a different size.
	RootDiskSize = 51200
)

type ociInstance struct {
	arch     *string
	instType *instances.InstanceType
	env      *Environ
	mutex    *sync.Mutex
	ocid     string
	etag     *string
	raw      ociCore.Instance
}

var _ instance.Instance = (*ociInstance)(nil)
var _ instance.InstanceFirewaller = (*ociInstance)(nil)

// newInstance returns a new oracleInstance
func newInstance(raw ociCore.Instance, env *Environ) (*ociInstance, error) {
	if raw.ID != nil || *raw.ID == "" {
		return nil, errors.New(
			"Instance response does not contain an ID",
		)
	}
	mutex := &sync.Mutex{}
	instance := &ociInstance{
		raw:   raw,
		mutex: mutex,
		env:   env,
		ocid:  *raw.ID,
	}

	return instance, nil
}

// Id implements instance.Instance
func (o *ociInstance) Id() instance.Id {
	if o.raw.ID == nil {
		return ""
	}
	return instance.Id(*o.raw.ID)
}

// Status implements instance.Instance
func (o *ociInstance) Status() instance.InstanceStatus {
	if o.raw.ID == nil {
		if err := o.refresh(); err != nil {
			return instance.InstanceStatus{}
		}
	}
	return instance.InstanceStatus{
		Status:  status.Status(o.raw.LifecycleState),
		Message: "",
	}
}

func (o *ociInstance) getInstanceVnicAttachments() (ociCore.ListVnicAttachmentsResponse, error) {
	request := ociCore.ListVnicAttachmentsRequest{
		CompartmentID: o.raw.CompartmentID,
		InstanceID:    o.raw.ID,
	}
	ctx := context.Background()
	response, err := o.env.cli.ComputeClient.ListVnicAttachments(ctx, request)
	if err != nil {
		return ociCore.ListVnicAttachmentsResponse{}, errors.Trace(err)
	}
	return response, nil
}

func (o *ociInstance) getInstanceVnics(vnics []ociCore.VnicAttachment) ([]ociCore.GetVnicResponse, error) {
	result := []ociCore.GetVnicResponse{}

	for _, val := range vnics {
		vnicID := val.VnicID
		request := ociCore.GetVnicRequest{
			VnicID: vnicID,
		}
		response, err := o.env.cli.VirtualNetwork.GetVnic(context.Background(), request)
		if err != nil {
			return nil, errors.Trace(err)
		}
		result = append(result, response)
	}
	return result, nil
}

// Addresses implements instance.Instance
func (o *ociInstance) Addresses() ([]network.Address, error) {
	attachments, err := o.getInstanceVnicAttachments()
	if err != nil {
		return nil, errors.Trace(err)
	}

	vnics, err := o.getInstanceVnics(attachments.Items)
	if err != nil {
		return nil, errors.Trace(err)
	}

	addresses := []network.Address{}

	for _, val := range vnics {
		if val.Vnic.PrivateIp != nil {
			privateAddress := network.NewScopedAddress(*val.Vnic.PrivateIp, network.ScopeCloudLocal)
			addresses = append(addresses, privateAddress)
		}
		if val.Vnic.PublicIp != nil {
			publicAddress := network.NewScopedAddress(*val.Vnic.PrivateIp, network.ScopePublic)
			addresses = append(addresses, publicAddress)
		}
	}
	return addresses, nil
}

func (o *ociInstance) deleteInstanceAndResources() error {
	err := o.refresh()
	if errors.IsNotFound(err) {
		return nil
	}
	request := ociCore.TerminateInstanceRequest{
		InstanceID: &o.ocid,
		IfMatch:    o.etag,
	}
	err = o.env.cli.ComputeClient.TerminateInstance(context.Background(), request)
	if err != nil {
		return err
	}
	iteration := 0
	for {
		if err := o.refresh(); err != nil {
			if errors.IsNotFound(err) {
				break
			}
			return err
		}
		if iteration >= 30 && o.raw.LifecycleState == ociCore.INSTANCE_LIFECYCLE_STATE_RUNNING {
			logger.Warningf("Instance still in running state after %v checks. breaking loop", iteration)
			break
		}
		<-o.env.clock.After(1 * time.Second)
		iteration++
		continue
	}
	// TODO(gsamfira): cleanup firewall rules
	// TODO(gsamfira): cleanup VNIC?
	return nil
}

// OpenPorts implements instance.InstanceFirewaller
func (o *ociInstance) OpenPorts(machineId string, rules []network.IngressRule) error {
	return nil
}

// ClosePorts implements instance.InstanceFirewaller
func (o *ociInstance) ClosePorts(machineId string, rules []network.IngressRule) error {
	return nil
}

// IngressRules implements instance.InstanceFirewaller
func (o *ociInstance) IngressRules(machineId string) ([]network.IngressRule, error) {
	return nil, nil
}

// hardwareCharacteristics returns the hardware characteristics of the current
// instance
func (o *ociInstance) hardwareCharacteristics() *instance.HardwareCharacteristics {
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

func (o *ociInstance) refresh() error {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	request := ociCore.GetInstanceRequest{
		InstanceID: &o.ocid,
	}
	response, err := o.env.cli.ComputeClient.GetInstance(context.Background(), request)
	if err != nil {
		if response.RawResponse != nil && response.RawResponse.StatusCode == http.StatusNotFound {
			// If we care about 404 errors, this makes it easier to test using
			// errors.IsNotFound
			return errors.NotFoundf("instance %s was not found", o.ocid)
		}
		return err
	}
	o.etag = response.Etag
	o.raw = response.Instance
	return nil
}
