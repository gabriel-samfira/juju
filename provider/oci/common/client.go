package common

import (
	"context"
	"crypto/rsa"
	"fmt"
	"io/ioutil"

	"github.com/juju/errors"

	ociCommon "github.com/oracle/oci-go-sdk/common"
	ociCore "github.com/oracle/oci-go-sdk/core"
	ociIdentity "github.com/oracle/oci-go-sdk/identity"
)

type jujuConfigProvider struct {
	keyFile        string
	keyFingerprint string
	passphrase     string
	tenancyOCID    string
	userOCID       string
	region         string
}

type ociClient struct {
	// TODO(gsamfira): See which functions we use from the bellow clients
	// and create interfaces, to be better able to mock them during testing
	ociCore.ComputeClient
	ociCore.BlockstorageClient
	ociCore.VirtualNetworkClient
	ociIdentity.IdentityClient

	ociCommon.ConfigurationProvider
}

// NewJujuConfigProvider returns a new ociCommon.ConfigurationProvider instance
func NewJujuConfigProvider(user, tenant, keyFile, fingerprint, passphrase, region string) ociCommon.ConfigurationProvider {
	return &jujuConfigProvider{
		keyFile:        keyFile,
		keyFingerprint: fingerprint,
		passphrase:     passphrase,
		tenancyOCID:    tenant,
		userOCID:       user,
		region:         region,
	}
}

func NewOciClient(provider ociCommon.ConfigurationProvider) (ApiClient, error) {
	computeClient, err := ociCore.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, errors.Trace(err)
	}

	blockStorage, err := ociCore.NewBlockstorageClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, errors.Trace(err)
	}

	virtualNetwork, err := ociCore.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ident, err := ociIdentity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &ociClient{
		computeClient,
		blockStorage,
		virtualNetwork,
		ident,
		provider,
	}, nil
}

// Ping validates that the client can access the OCI API successfully
func (o *ociClient) Ping() error {
	tenancyID, err := o.TenancyOCID()
	if err != nil {
		return errors.Trace(err)
	}
	request := ociIdentity.ListCompartmentsRequest{
		CompartmentID: &tenancyID,
	}
	ctx := context.Background()
	_, err = o.ListCompartments(ctx, request)
	return err
}

func (j jujuConfigProvider) TenancyOCID() (string, error) {
	if j.tenancyOCID == "" {
		return "", errors.Errorf("tenancyOCID is not set")
	}
	return j.tenancyOCID, nil
}

func (j jujuConfigProvider) UserOCID() (string, error) {
	if j.userOCID == "" {
		return "", errors.Errorf("userOCID is not set")
	}
	return j.userOCID, nil
}

func (j jujuConfigProvider) KeyFingerprint() (string, error) {
	if j.keyFingerprint == "" {
		return "", errors.Errorf("keyFingerprint is not set")
	}
	return j.keyFingerprint, nil
}

func (j jujuConfigProvider) Region() (string, error) {
	return j.region, nil
}

func (j jujuConfigProvider) PrivateRSAKey() (*rsa.PrivateKey, error) {
	if j.keyFile == "" {
		return nil, errors.Errorf("private key file is not set")
	}
	pemFileContent, err := ioutil.ReadFile(j.keyFile)
	if err != nil {
		return nil, errors.Trace(err)
	}

	key, err := ociCommon.PrivateKeyFromBytes(
		pemFileContent, &j.passphrase)
	return key, err
}

func (j jujuConfigProvider) KeyID() (string, error) {
	if j.tenancyOCID == "" || j.userOCID == "" || j.keyFingerprint == "" {
		return "", errors.Errorf("config provider is not properly initialized")
	}
	return fmt.Sprintf("%s/%s/%s", j.tenancyOCID, j.userOCID, j.keyFingerprint), nil
}
