// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package oci

import (
	"context"
	"time"

	"github.com/juju/errors"
	"github.com/juju/utils/clock"

	"github.com/juju/juju/environs/tags"
	providerCommon "github.com/juju/juju/provider/oci/common"
	"github.com/juju/juju/storage"

	ociCore "github.com/oracle/oci-go-sdk/core"
	// ociIdentity "github.com/oracle/oci-go-sdk/identity"
)

func mibToGib(m uint64) uint64 {
	return (m + 1023) / 1024
}

type volumeSource struct {
	env       *Environ
	envName   string
	modelUUID string
	api       providerCommon.ApiClient
	clock     clock.Clock
}

func newOciVolumeSource(env *Environ, name, uuid string, api providerCommon.ApiClient, clock clock.Clock) (*volumeSource, error) {
	if env == nil {
		return nil, errors.NotFoundf("environ")
	}

	if api == nil {
		return nil, errors.NotFoundf("storage client")
	}

	return &volumeSource{
		env:       env,
		envName:   name,
		modelUUID: uuid,
		api:       api,
		clock:     clock,
	}, nil
}

var _ storage.VolumeSource = (*volumeSource)(nil)

func (v *volumeSource) getVolumeStatus(resourceID *string) (string, error) {
	request := ociCore.GetVolumeRequest{
		VolumeID: resourceID,
	}

	response, err := v.api.GetVolume(context.Background(), request)
	if err != nil {
		if v.env.isNotFound(response.RawResponse) {
			return "", errors.NotFoundf("subnet not found: %s", *resourceID)
		} else {
			return "", err
		}
	}
	return string(response.Volume.LifecycleState), nil
}

func (v *volumeSource) createVolume(p storage.VolumeParams, instanceMap map[string]*ociInstance) (_ *storage.Volume, err error) {
	var details ociCore.CreateVolumeResponse
	defer func() {
		if err != nil && details.ID != nil {
			req := ociCore.DeleteVolumeRequest{
				VolumeID: details.ID,
			}
			nestedErr := v.api.DeleteVolume(context.Background(), req)
			if nestedErr != nil {
				logger.Warningf("failed to cleanup volume: %s", *details.ID)
				return
			}
			nestedErr = v.env.waitForResourceStatus(
				v.getVolumeStatus, details.ID,
				string(ociCore.VOLUME_LIFECYCLE_STATE_TERMINATED),
				5*time.Minute)
			if nestedErr != nil && !errors.IsNotFound(nestedErr) {
				logger.Warningf("failed to cleanup volume: %s", *details.ID)
				return
			}
		}
	}()
	if err := v.ValidateVolumeParams(p); err != nil {
		return nil, errors.Trace(err)
	}
	if p.Attachment == nil {
		return nil, errors.Errorf("volume %s has no attachments", p.Tag.String())
	}
	instanceId := p.Attachment.InstanceId
	instance, ok := instanceMap[instanceId]
	if !ok {
		instance, err = v.env.getOciInstances(instanceId)
		if err != nil {
			return nil, errors.Trace(err)
		}
		instanceMap[instanceId] = instance
	}

	availabilityZone := instance.availabilityZone()
	name := p.Tag.String()
	volTags := p.ResourceTags
	volTags[tags.JujuModel] = v.modelUUID

	requestDetails := ociCore.CreateVolumeDetails{
		AvailabilityDomain: &availabilityZone,
		CompartmentID:      v.env.ecfg().compartmentID(),
		DisplayName:        &name,
		SizeInMBs:          p.Size,
		FreeFormTags:       volTags,
	}

	request := ociCore.CreateVolumeRequest{
		CreateVolumeDetails: requestDetails,
	}

	result, err := v.api.CreateVolume(context.Background(), request)
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = v.env.waitForResourceStatus(
		v.getVolumeStatus, result.Volume.ID,
		ociCore.VOLUME_LIFECYCLE_STATE_AVAILABLE,
		5*time.Minute)
	if err != nil {
		return nil, errors.Trace(err)
	}

	volumeDetails, err := v.api.GetVolume(
		context.Background(), ociCore.GetVolumeRequest{VolumeID: result.Volume.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &storage.Volume{p.Tag, makeVolumeInfo(volumeDetails)}, nil
}

func makeVolumeInfo(vol ociCore.Volume) storage.VolumeInfo {
	return storage.VolumeInfo{
		VolumeId: vol.ID,
		// Oracle returns the size of the volume
		// in bytes, VolumeInfo expects MiB.
		Size:       vol.SizeInMBs,
		Persistent: true,
	}
}

func (v *volumeSource) CreateVolumes(params []storage.VolumeParams) ([]storage.CreateVolumesResult, error) {
	if params == nil {
		return []storage.CreateVolumesResult{}, nil
	}
	results := make([]storage.CreateVolumesResult, len(params))
	instanceMap := map[string]*ociInstance{}
	for i, volume := range params {
		vol, err := v.createVolume(volume, instanceMap)
		if err != nil {
			results[i].Error = errors.Trace(err)
			continue
		}
		results[i].Volume = vol
	}
	return results, nil
}

func (v *volumeSource) allVolumes() (map[string]ociCore.Volume, error) {
	result := map[string]ociCore.Volume{}
	request := ociCore.ListVolumesRequest{
		CompartmentID: v.env.ecfg().compartmentID(),
	}
	response, err := v.api.ListVolumes(context.Background(), request)
	if err != nil {
		return nil, err
	}

	for _, val := range response.Items {
		if t, ok := val.FreeFormTags[tags.JujuModel]; !ok {
			continue
		} else {
			if t != nil && *t != v.modelUUID {
				continue
			}
		}
		result[*val.ID] = val
	}
	return result, nil
}

func (v *volumeSource) ListVolumes() ([]string, error) {
	ids := []string{}
	volumes, err := v.allVolumes()
	if err != nil {
		return nil, err
	}

	for k, _ := range volumes {
		ids = append(ids, k)
	}
	return ids, nil
}

func (v *volumeSource) DescribeVolumes(volIds []string) ([]storage.DescribeVolumesResult, error) {
	result := make([]storage.DescribeVolumesResult, len(volIds), len(volIds))

	allVolumes, err := v.allVolumes()
	if err != nil {
		return nil, errors.Trace(err)
	}

	for i, val := range volIds {
		if volume, ok := allVolumes[val]; ok {
			volumeInfo := makeVolumeInfo(volume)
			result[i].VolumeInfo = &volumeInfo
		} else {
			result[i].Error = errors.NotFoundf("%s", volume)
		}
	}
	return result, nil
}

func (v *volumeSource) DestroyVolumes(volIds []string) ([]error, error) {
	return nil, nil
}

func (v *volumeSource) ReleaseVolumes(volIds []string) ([]error, error) {
	return nil, nil
}

func (v *volumeSource) ValidateVolumeParams(params storage.VolumeParams) error {
	size := mibToGib(params.Size)
	if size < minVolumeSizeInGB || size > maxVolumeSizeInGB {
		return errors.Errorf(
			"invalid volume size %s. Valid range is %s - %s (GiB)", size, minVolumeSizeInGB, maxVolumeSizeInGB)
	}
	return nil
}

func (v *volumeSource) AttachVolumes(params []storage.VolumeAttachmentParams) ([]storage.AttachVolumesResult, error) {
	return nil, nil
}

func (v *volumeSource) DetachVolumes(params []storage.VolumeAttachmentParams) ([]error, error) {
	return nil, nil
}
