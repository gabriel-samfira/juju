package common

import (
	"context"

	ociCommon "github.com/oracle/oci-go-sdk/common"
	ociCore "github.com/oracle/oci-go-sdk/core"
	ociIdentity "github.com/oracle/oci-go-sdk/identity"
)

type ApiClient interface {
	ociCommon.ConfigurationProvider

	Ping() error

	ListShapes(ctx context.Context, request ociCore.ListShapesRequest) (response ociCore.ListShapesResponse, err error)
	ListImages(ctx context.Context, request ociCore.ListImagesRequest) (response ociCore.ListImagesResponse, err error)

	ListAvailabilityDomains(ctx context.Context, request ociIdentity.ListAvailabilityDomainsRequest) (response ociIdentity.ListAvailabilityDomainsResponse, err error)

	ListInstances(ctx context.Context, request ociCore.ListInstancesRequest) (response ociCore.ListInstancesResponse, err error)

	ListVnicAttachments(ctx context.Context, request ociCore.ListVnicAttachmentsRequest) (response ociCore.ListVnicAttachmentsResponse, err error)
	GetVnic(ctx context.Context, request ociCore.GetVnicRequest) (response ociCore.GetVnicResponse, err error)
	TerminateInstance(ctx context.Context, request ociCore.TerminateInstanceRequest) (err error)
	GetInstance(ctx context.Context, request ociCore.GetInstanceRequest) (response ociCore.GetInstanceResponse, err error)

	CreateVcn(ctx context.Context, request ociCore.CreateVcnRequest) (response ociCore.CreateVcnResponse, err error)
	ListVcns(ctx context.Context, request ociCore.ListVcnsRequest) (response ociCore.ListVcnsResponse, err error)
}
