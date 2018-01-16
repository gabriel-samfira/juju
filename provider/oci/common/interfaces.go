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
	LaunchInstance(ctx context.Context, request ociCore.LaunchInstanceRequest) (response ociCore.LaunchInstanceResponse, err error)

	CreateVcn(ctx context.Context, request ociCore.CreateVcnRequest) (response ociCore.CreateVcnResponse, err error)
	DeleteVcn(ctx context.Context, request ociCore.DeleteVcnRequest) (err error)
	ListVcns(ctx context.Context, request ociCore.ListVcnsRequest) (response ociCore.ListVcnsResponse, err error)
	GetVcn(ctx context.Context, request ociCore.GetVcnRequest) (response ociCore.GetVcnResponse, err error)

	CreateSecurityList(ctx context.Context, request ociCore.CreateSecurityListRequest) (response ociCore.CreateSecurityListResponse, err error)
	ListSecurityLists(ctx context.Context, request ociCore.ListSecurityListsRequest) (response ociCore.ListSecurityListsResponse, err error)
	DeleteSecurityList(ctx context.Context, request ociCore.DeleteSecurityListRequest) (err error)
	GetSecurityList(ctx context.Context, request ociCore.GetSecurityListRequest) (response ociCore.GetSecurityListResponse, err error)

	CreateSubnet(ctx context.Context, request ociCore.CreateSubnetRequest) (response ociCore.CreateSubnetResponse, err error)
	ListSubnets(ctx context.Context, request ociCore.ListSubnetsRequest) (response ociCore.ListSubnetsResponse, err error)
	DeleteSubnet(ctx context.Context, request ociCore.DeleteSubnetRequest) (err error)
	GetSubnet(ctx context.Context, request ociCore.GetSubnetRequest) (response ociCore.GetSubnetResponse, err error)
}
