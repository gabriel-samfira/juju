// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package oci

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/juju/errors"
	"github.com/juju/utils/arch"
	"github.com/juju/utils/clock"
	"github.com/juju/version"

	"github.com/juju/juju/cloudconfig/instancecfg"
	"github.com/juju/juju/constraints"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/environs/instances"
	"github.com/juju/juju/environs/tags"
	"github.com/juju/juju/instance"
	"github.com/juju/juju/provider/common"
	providerCommon "github.com/juju/juju/provider/oci/common"
	providerNetwork "github.com/juju/juju/provider/oci/network"
	"github.com/juju/juju/storage"
	"github.com/juju/juju/tools"

	ociCore "github.com/oracle/oci-go-sdk/core"
	ociIdentity "github.com/oracle/oci-go-sdk/identity"
)

type Environ struct {
	environs.Networking
	environs.Firewaller

	cli providerCommon.ApiClient
	p   *EnvironProvider

	clock clock.Clock
	cfg   *config.Config

	ecfgMutex sync.Mutex
	ecfgObj   *environConfig

	vcn     ociCore.Vcn
	seclist ociCore.SecurityList
	// subnets contains one subnet for each availability domain
	// these will get created once the environment is spun up, and
	// will never change.
	subnets map[string][]ociCore.Subnet
}

var (
	tcpProtocolNumber  = "6"
	udpProtocolNumber  = "17"
	icmpProtocolNumber = "1"
	allProtocols       = "all"
)

var _ common.ZonedEnviron = (*Environ)(nil)
var _ storage.ProviderRegistry = (*Environ)(nil)
var _ environs.Environ = (*Environ)(nil)
var _ environs.Firewaller = (*Environ)(nil)
var _ environs.Networking = (*Environ)(nil)

// AvailabilityZones is defined in the common.ZonedEnviron interface
func (e *Environ) AvailabilityZones() ([]common.AvailabilityZone, error) {
	ocid, err := e.cli.TenancyOCID()
	if err != nil {
		return nil, errors.Trace(err)
	}
	request := ociIdentity.ListAvailabilityDomainsRequest{
		CompartmentID: &ocid,
	}
	ctx := context.Background()
	domains, err := e.cli.ListAvailabilityDomains(ctx, request)
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
func (e *Environ) InstanceAvailabilityZoneNames(ids []instance.Id) ([]string, error) {
	instances, err := e.Instances(ids)
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
		CompartmentID: compartmentID,
	}

	instances, err := e.cli.ListInstances(context.Background(), request)
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

func (e *Environ) vcnName(controllerUUID string) *string {
	name := fmt.Sprintf("%s-%s", providerNetwork.VcnNamePrefix, controllerUUID)
	return &name
}

func (e *Environ) getVcn(controllerUUID string) (ociCore.Vcn, error) {
	request := ociCore.ListVcnsRequest{
		CompartmentID: e.ecfg().compartmentID(),
	}

	response, err := e.cli.ListVcns(context.Background(), request)
	if err != nil {
		return ociCore.Vcn{}, errors.Trace(err)
	}
	name := e.vcnName(controllerUUID)

	if len(response.Items) > 0 {
		for _, val := range response.Items {
			// NOTE(gsamfira): Display names are not unique. We only care
			// about VCNs that have been created for this controller.
			// While we do include the controller UUID in the name of
			// the VCN, I believe it is worth doing an extra check.
			if *val.DisplayName != *name {
				continue
			}
			if tag, ok := val.FreeFormTags[tags.JujuController]; ok {
				if tag == controllerUUID {
					return val, nil
				}
			}
		}
	}
	return ociCore.Vcn{}, errors.NotFoundf("no such VCN: %s", *name)
}

func (e *Environ) secListName(controllerUUID string) string {
	return fmt.Sprintf("juju-seclist-%s", controllerUUID)
}

func (e *Environ) ensureVCN(controllerUUID string) (vcn ociCore.Vcn, err error) {
	if vcn, err = e.getVcn(controllerUUID); err != nil {
		if !errors.IsNotFound(err) {
			return
		}
	} else {
		return
	}

	name := e.vcnName(controllerUUID)
	if err != nil {
		return
	}
	logger.Infof("creating new VCN %s", *name)
	addressSpace := providerNetwork.DefaultAddressSpace
	vcnDetails := ociCore.CreateVcnDetails{
		CidrBlock:     &addressSpace,
		CompartmentID: e.ecfg().compartmentID(),
		DisplayName:   name,
		FreeFormTags: map[string]string{
			tags.JujuController: controllerUUID,
		},
	}
	request := ociCore.CreateVcnRequest{
		CreateVcnDetails: vcnDetails,
	}

	result, err := e.cli.CreateVcn(context.Background(), request)
	if err != nil {
		return
	}
	vcn = result.Vcn
	return
}

func (e *Environ) getSecurityList(controllerUUID string, vcnid *string) (ociCore.SecurityList, error) {
	name := e.secListName(controllerUUID)
	request := ociCore.ListSecurityListsRequest{
		CompartmentID: e.ecfg().compartmentID(),
		VcnID:         vcnid,
	}

	response, err := e.cli.ListSecurityLists(context.Background(), request)
	if err != nil {
		return ociCore.SecurityList{}, errors.Trace(err)
	}
	if len(response.Items) == 0 {
		return ociCore.SecurityList{}, errors.NotFoundf("security list %s does not exist", name)
	}
	for _, val := range response.Items {
		if *val.DisplayName == name {
			if tag, ok := val.FreeFormTags[tags.JujuController]; ok {
				if tag == controllerUUID {
					return val, nil
				}
			}
		}
	}
	return ociCore.SecurityList{}, errors.NotFoundf("security list %s does not exist", name)
}

func (e *Environ) ensureSecurityList(controllerUUID string, vcnid *string) (ociCore.SecurityList, error) {
	if seclist, err := e.getSecurityList(controllerUUID, vcnid); err != nil {
		if !errors.IsNotFound(err) {
			return ociCore.SecurityList{}, errors.Trace(err)
		}
	} else {
		return seclist, nil
	}

	prefix := "0.0.0.0/0"

	// Hopefully just temporary, open all ingress/egress ports
	details := ociCore.CreateSecurityListDetails{
		CompartmentID: e.ecfg().compartmentID(),
		VcnID:         vcnid,
		DisplayName:   &controllerUUID,
		FreeFormTags: map[string]string{
			tags.JujuController: controllerUUID,
		},
		EgressSecurityRules: []ociCore.EgressSecurityRule{
			ociCore.EgressSecurityRule{
				Destination: &prefix,
				Protocol:    &allProtocols,
			},
		},
		IngressSecurityRules: []ociCore.IngressSecurityRule{
			ociCore.IngressSecurityRule{
				Source:   &prefix,
				Protocol: &allProtocols,
			},
		},
	}

	request := ociCore.CreateSecurityListRequest{
		CreateSecurityListDetails: details,
	}

	response, err := e.cli.CreateSecurityList(context.Background(), request)
	if err != nil {
		return ociCore.SecurityList{}, errors.Trace(err)
	}
	return response.SecurityList, nil
}

func (e *Environ) allSubnets(controllerUUID string, vcnID *string) (map[string][]ociCore.Subnet, error) {
	request := ociCore.ListSubnetsRequest{
		CompartmentID: e.ecfg().compartmentID(),
		VcnID:         vcnID,
	}
	response, err := e.cli.ListSubnets(context.Background(), request)
	if err != nil {
		return nil, err
	}

	ret := map[string][]ociCore.Subnet{}
	for _, val := range response.Items {
		if tag, ok := val.FreeFormTags[tags.JujuController]; ok {
			if tag == controllerUUID {
				cidr := *val.CidrBlock
				if valid, err := e.validateCidrBlock(cidr); err != nil || !valid {
					logger.Warningf("failed to validate CIDR block %s: %s", cidr, err)
					continue
				}
				ret[*val.AvailabilityDomain] = append(ret[*val.AvailabilityDomain], val)
			}
		}
	}
	return ret, nil
}

func (e *Environ) validateCidrBlock(cidr string) (bool, error) {
	_, vncIPNet, err := net.ParseCIDR(providerNetwork.DefaultAddressSpace)
	if err != nil {
		return false, errors.Trace(err)
	}

	subnetIP, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, errors.Trace(err)
	}
	if vncIPNet.Contains(subnetIP) {
		return true, nil
	}
	return false, nil
}

func (e *Environ) getFreeSubnet(existing map[string]bool) (string, error) {
	return "", nil
}

func (e *Environ) createSubnet(controllerUUID, ad, cidr string, vcnID *string, seclists []string) (ociCore.Subnet, error) {
	return ociCore.Subnet{}, nil
}

func (e *Environ) ensureSubnets(vcn ociCore.Vcn, secList ociCore.SecurityList, controllerUUID string) (map[string][]ociCore.Subnet, error) {
	az, err := e.AvailabilityZones()
	if err != nil {
		return nil, errors.Trace(err)
	}

	allSubnets, err := e.allSubnets(controllerUUID, vcn.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	existingCidrBlocks := map[string]bool{}
	missing := map[string]bool{}
	// Check that we have one subnet, and only one subnet in each availability domain
	for _, val := range az {
		name := val.Name()
		subnets, ok := allSubnets[name]
		if !ok {
			missing[name] = true
			continue
		}
		for _, val := range subnets {
			cidr := *val.CidrBlock
			existingCidrBlocks[cidr] = true
		}
	}

	if len(missing) > 0 {
		for ad, _ := range missing {
			newIPNet, err := e.getFreeSubnet(existingCidrBlocks)
			if err != nil {
				return nil, errors.Trace(err)
			}
			newSubnet, err := e.createSubnet(controllerUUID, ad, newIPNet, vcn.ID, []string{*secList.ID})
			if err != nil {
				return nil, errors.Trace(err)
			}
			allSubnets[ad] = []ociCore.Subnet{
				newSubnet,
			}
		}
	}

	return allSubnets, nil
}

// ensureNetworksAndSubnets creates VCNs, security lists and subnets that will
// be used throughout the life-cycle of this juju deployment.
func (e *Environ) ensureNetworksAndSubnets(controllerUUID string) error {
	// if we have the subnets field populated, it means we already checked/created
	// the necessary resources. Simply return.
	if e.subnets != nil {
		return nil
	}
	vcn, err := e.ensureVCN(controllerUUID)
	if err != nil {
		return errors.Trace(err)
	}

	// NOTE(gsamfira): There are some limitations at the moment in regards to
	// security lists:
	// * Security lists can only be applied on subnets
	// * Once subnet is created, you may not attach a new security list to that subnet
	// * there is no way to apply a security list on an instance/VNIC
	// * We cannot create a model level security list, unless we create a new subnet for that model
	// ** that means at least 3 subnets per model, which is something we probably don't want
	// * There is no way to specify the target prefix for an Ingress/Egress rule, thus making
	// instance level firewalling, impossible.
	// For now, we open all ports until we decide how to properly take care of this.
	secList, err := e.ensureSecurityList(controllerUUID, vcn.ID)
	if err != nil {
		return errors.Trace(err)
	}

	subnets, err := e.ensureSubnets(vcn, secList, controllerUUID)
	if err != nil {
		return errors.Trace(err)
	}
	// TODO(gsamfira): should we use a lock here?
	e.subnets = subnets
	return nil
}

// cleanupNetworksAndSubnets destroys all subnets, VCNs and security lists that have
// been used by this juju deployment. This function should only be called when
// destroying the environment
func (e *Environ) cleanupNetworksAndSubnets(controllerUUID string) error {
	return nil
}

// Create implements environs.Environ.
func (e *Environ) Create(params environs.CreateParams) error {
	if err := e.cli.Ping(); err != nil {
		return errors.Trace(err)
	}
	// err := e.ensureNetworksAndSubnets(params.ControllerUUID)
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
		CompartmentID: compartment,
	}
	response, err := e.cli.ListInstances(context.Background(), request)
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
	// var types []instances.InstanceType

	if args.ControllerUUID == "" {
		return nil, errors.NotFoundf("Controller UUID")
	}

	// refresh the global image cache
	// this only hits the API every 30 minutes, otherwise just retrieves
	// from cache
	imgCache, err := refreshImageCache(e.cli, e.ecfg().compartmentID())
	if err != nil {
		return nil, errors.Trace(err)
	}

	// TODO(gsamfira): implement imageCache filter by series, and other attributes
	// TODO(gsamfira): generate []ImageMetadata from filtered images
	// TODO(gsamfira): get []InstanceType for filtered images
	series := args.Tools.OneSeries()
	arches := args.Tools.Arches()

	types := imgCache.supportedShapes(series)
	// check if we find an image that is compliant with the
	// constraints provided in the oracle cloud account
	args.ImageMetadata = imgCache.imageMetadata(series)

	if args.Constraints.VirtType == nil {
		defaultType := string(VirtualMachine)
		args.Constraints.VirtType = &defaultType
	}

	spec, image, err := findInstanceSpec(
		args.ImageMetadata,
		types,
		&instances.InstanceConstraint{
			Series:      series,
			Arches:      arches,
			Constraints: args.Constraints,
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	tools, err := args.Tools.Match(tools.Filter{Arch: spec.Image.Arch})
	if err != nil {
		return nil, errors.Trace(err)
	}
	logger.Tracef("agent binaries: %v", tools)
	if err = args.InstanceConfig.SetTools(tools); err != nil {
		return nil, errors.Trace(err)
	}

	if err = instancecfg.FinishInstanceConfig(
		args.InstanceConfig,
		e.Config(),
	); err != nil {
		return nil, errors.Trace(err)
	}
	hostname := args.InstanceConfig.MachineId
	tags := args.InstanceConfig.Tags

	var apiPort int
	var desiredStatus ociCore.InstanceLifecycleStateEnum
	// Wait for controller to actually be running
	if args.InstanceConfig.Controller != nil {
		apiPort = args.InstanceConfig.Controller.Config.APIPort()
		desiredStatus = ociCore.INSTANCE_LIFECYCLE_STATE_RUNNING
	} else {
		// All ports are the same so pick the first.
		apiPort = args.InstanceConfig.APIInfo.Ports()[0]
		desiredStatus = ociCore.INSTANCE_LIFECYCLE_STATE_STARTING
	}

	// TODO(gsamfira): Setup firewall rules for this instance
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
			if err := inst.deleteInstance(); err != nil {
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
	tags := map[string]string{
		tags.JujuModel: e.Config().UUID(),
	}
	instances, err := e.allInstances(tags)
	if err != nil {
		return nil, err
	}

	ret := []instance.Instance{}
	for _, val := range instances {
		ret = append(ret, val)
	}
	return ret, nil
}

// MaintainInstance implements environs.InstanceBroker.
func (e *Environ) MaintainInstance(args environs.StartInstanceParams) error {
	return nil
}

// Config implements environs.ConfigGetter.
func (e *Environ) Config() *config.Config {
	e.ecfgMutex.Lock()
	defer e.ecfgMutex.Unlock()
	return e.cfg
}

// PrecheckInstance implements environs.InstancePrechecker.
func (e *Environ) PrecheckInstance(environs.PrecheckInstanceParams) error {
	// var i instances.InstanceTypesWithCostMetadata
	return nil
}

// InstanceTypes implements environs.InstancePrechecker.
func (e *Environ) InstanceTypes(constraints.Value) (instances.InstanceTypesWithCostMetadata, error) {
	return instances.InstanceTypesWithCostMetadata{}, nil
}
