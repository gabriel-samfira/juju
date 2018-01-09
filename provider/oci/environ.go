// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package oci

import (
	"context"
	"sync"

	"github.com/juju/errors"
	"github.com/juju/utils/arch"
	"github.com/juju/utils/clock"
	"github.com/juju/version"

	"github.com/juju/juju/constraints"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/environs/instances"
	"github.com/juju/juju/environs/tags"
	"github.com/juju/juju/instance"
	"github.com/juju/juju/provider/common"
	// "github.com/juju/juju/provider/oci/network"
	"github.com/juju/juju/storage"

	ociCore "github.com/oracle/oci-go-sdk/core"
	ociIdentity "github.com/oracle/oci-go-sdk/identity"
)

type Environ struct {
	cli *ociClient
	p   *EnvironProvider

	clock clock.Clock

	ecfgMutex sync.Mutex
	ecfgObj   *environConfig
}

var _ common.ZonedEnviron = (*Environ)(nil)
var _ storage.ProviderRegistry = (*Environ)(nil)
var _ environs.Environ = (*Environ)(nil)

// AvailabilityZones is defined in the common.ZonedEnviron interface
func (o *Environ) AvailabilityZones() ([]common.AvailabilityZone, error) {
	ocid, err := o.cli.ConfigProvider.TenancyOCID()
	if err != nil {
		return nil, errors.Trace(err)
	}
	request := ociIdentity.ListAvailabilityDomainsRequest{
		CompartmentID: &ocid,
	}
	ctx := context.Background()
	domains, err := o.cli.Identity.ListAvailabilityDomains(ctx, request)
	if err != nil {
		return nil, errors.Trace(err)
	}

	zones := []common.AvailabilityZone{}

	for _, val := range domains.Items {
		zones = append(zones, NewAvailabilityZone(*val.Name))
	}
	return zones, nil
}

// InstanceAvailabilityzoneNames implements common.ZonedEnviron.
func (o *Environ) InstanceAvailabilityZoneNames(ids []instance.Id) ([]string, error) {
	instances, err := o.Instances(ids)
	if err != nil && err != environs.ErrPartialInstances {
		return nil, err
	}
	zones := make([]string, len(instances))
	for idx, _ := range instances {
		zones[idx] = "default"
	}
	return zones, nil
}

// DeriveAvailabilityZones implements common.ZonedEnviron.
func (e *Environ) DeriveAvailabilityZones(args environs.StartInstanceParams) ([]string, error) {
	return nil, nil
}

func (e *Environ) getOciInstances(ids ...instance.Id) ([]*ociInstance, error) {
	ret := []*ociInstance{}

	compartmentID := e.ecfg().compartmentID()
	request := ociCore.ListInstancesRequest{
		CompartmentID: &compartmentID,
	}

	instances, err := e.cli.ComputeClient.ListInstances(context.Background(), request)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(instances.Items) == 0 {
		return nil, environs.ErrNoInstances
	}

	for _, val := range instances.Items {
		oInstance, err := newInstance(val, e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		for _, id := range ids {
			if oInstance.Id() == id {
				ret = append(ret, oInstance)
			}
		}
	}

	if len(ret) < len(ids) {
		return ret, environs.ErrPartialInstances
	}
	return ret, nil
}

// Instances implements environs.Environ.
func (e *Environ) Instances(ids []instance.Id) ([]instance.Instance, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	instances, err := e.getOciInstances(ids...)
	if err != nil {
		return nil, err
	}

	ret := []instance.Instance{}
	for _, val := range instances {
		ret = append(ret, val)
	}
	return ret, nil
}

// PrepareForBootstrap implements environs.Environ.
func (e *Environ) PrepareForBootstrap(ctx environs.BootstrapContext) error {
	if ctx.ShouldVerifyCredentials() {
		logger.Infof("Logging into the oracle cloud infrastructure")
		if err := e.cli.Ping(); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// Bootstrap implements environs.Environ.
func (e *Environ) Bootstrap(ctx environs.BootstrapContext, params environs.BootstrapParams) (*environs.BootstrapResult, error) {
	return common.Bootstrap(ctx, e, params)
}

// Create implements environs.Environ.
func (e *Environ) Create(params environs.CreateParams) error {
	if err := e.cli.Ping(); err != nil {
		return errors.Trace(err)
	}
	// TODO(gsamfira): Create networks, security lists, and possibly containers

	return nil
}

// AdoptResources implements environs.Environ.
func (e *Environ) AdoptResources(controllerUUID string, fromVersion version.Number) error {
	return nil
}

// ConstraintsValidator implements environs.Environ.
func (e *Environ) ConstraintsValidator() (constraints.Validator, error) {
	// list of unsupported OCI provider constraints
	unsupportedConstraints := []string{
		constraints.Container,
		constraints.CpuPower,
		constraints.RootDisk,
		constraints.VirtType,
		constraints.Tags,
	}

	validator := constraints.NewValidator()
	validator.RegisterUnsupported(unsupportedConstraints)
	validator.RegisterVocabulary(constraints.Arch, []string{arch.AMD64})
	logger.Infof("Returning constraints validator: %v", validator)
	return validator, nil
}

// SetConfig implements environs.Environ.
func (e *Environ) SetConfig(cfg *config.Config) error {
	ecfg, err := e.p.newConfig(cfg)
	if err != nil {
		return err
	}

	e.ecfgMutex.Lock()
	defer e.ecfgMutex.Unlock()
	e.ecfgObj = ecfg

	return nil
}

func (e *Environ) ecfg() *environConfig {
	e.ecfgMutex.Lock()
	defer e.ecfgMutex.Unlock()
	return e.ecfgObj
}

func (e *Environ) allInstances(tags map[string]string) ([]*ociInstance, error) {
	compartment := e.ecfg().compartmentID()
	request := ociCore.ListInstancesRequest{
		CompartmentID: &compartment,
	}
	response, err := e.cli.ComputeClient.ListInstances(context.Background(), request)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ret := []*ociInstance{}
	for _, val := range response.Items {
		missingTag := false
		for i, j := range tags {
			tagVal, ok := val.FreeFormTags[i]
			if !ok || tagVal != j {
				missingTag = true
				break
			}
		}
		if missingTag {
			// One of the tags was not found
			continue
		}
		inst, err := newInstance(val, e)
		if err != nil {
			return nil, errors.Trace(err)
		}
		ret = append(ret, inst)
	}
	return ret, nil
}

func (e *Environ) allControllerManagedInstances(controllerUUID string) ([]*ociInstance, error) {
	tags := map[string]string{
		tags.JujuController: controllerUUID,
	}
	return e.allInstances(tags)
}

// ControllerInstances implements environs.Environ.
func (e *Environ) ControllerInstances(controllerUUID string) ([]instance.Id, error) {
	tags := map[string]string{
		tags.JujuController:   controllerUUID,
		tags.JujuIsController: "true",
	}
	instances, err := e.allInstances(tags)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ids := []instance.Id{}
	for _, val := range instances {
		ids = append(ids, val.Id())
	}
	return ids, nil
}

// Destroy implements environs.Environ.
func (e *Environ) Destroy() error {
	return common.Destroy(e)
}

// DestroyController implements environs.Environ.
func (e *Environ) DestroyController(controllerUUID string) error {
	err := e.Destroy()
	if err != nil {
		logger.Errorf("Failed to destroy environment through controller: %s", errors.Trace(err))
	}
	instances, err := e.allControllerManagedInstances(controllerUUID)
	if err != nil {
		if err == environs.ErrNoInstances {
			return nil
		}
		return errors.Trace(err)
	}
	ids := make([]instance.Id, len(instances))
	for i, val := range instances {
		ids[i] = val.Id()
	}
	return e.StopInstances(ids...)
}

// Provider implements environs.Environ.
func (e *Environ) Provider() environs.EnvironProvider {
	return e.p
}

// StorageProviderTypes implements storage.ProviderRegistry.
func (e *Environ) StorageProviderTypes() ([]storage.ProviderType, error) {
	return nil, nil
}

// StorageProvider implements storage.ProviderRegistry.
func (e *Environ) StorageProvider(storage.ProviderType) (storage.Provider, error) {
	return nil, nil
}

// StartInstance implements environs.InstanceBroker.
func (e *Environ) StartInstance(args environs.StartInstanceParams) (*environs.StartInstanceResult, error) {
	return nil, nil
}

// StopInstances implements environs.InstanceBroker.
func (e *Environ) StopInstances(ids ...instance.Id) error {
	ociInstances, err := e.getOciInstances(ids...)
	if err == environs.ErrNoInstances {
		return nil
	} else if err != nil {
		return err
	}

	logger.Debugf("terminating instances %v", ids)
	if err := e.terminateInstances(ociInstances...); err != nil {
		return err
	}

	return nil
}

func (o *Environ) terminateInstances(instances ...*ociInstance) error {
	wg := sync.WaitGroup{}
	wg.Add(len(instances))
	errs := []error{}
	instIds := []instance.Id{}
	for _, oInst := range instances {
		go func(inst *ociInstance) {
			defer wg.Done()
			if err := inst.deleteInstanceAndResources(); err != nil {
				instIds = append(instIds, inst.Id())
				errs = append(errs, err)
			}
		}(oInst)
	}
	wg.Wait()
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errors.Annotatef(errs[0], "failed to stop instance %s", instIds[0])
	default:
		return errors.Errorf(
			"failed to stop instances %s: %s",
			instIds, errs,
		)
	}
}

// AllInstances implements environs.InstanceBroker.
func (e *Environ) AllInstances() ([]instance.Instance, error) {
	return nil, nil
}

// MaintainInstance implements environs.InstanceBroker.
func (e *Environ) MaintainInstance(args environs.StartInstanceParams) error {
	return nil
}

// Config implements environs.ConfigGetter.
func (e *Environ) Config() *config.Config {
	return nil
}

// PrecheckInstance implements environs.InstancePrechecker.
func (e *Environ) PrecheckInstance(environs.PrecheckInstanceParams) error {
	return nil
}

// InstanceTypes implements environs.InstancePrechecker.
func (e *Environ) InstanceTypes(constraints.Value) (instances.InstanceTypesWithCostMetadata, error) {
	return instances.InstanceTypesWithCostMetadata{}, nil
}
