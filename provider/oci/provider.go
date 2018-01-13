// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package oci

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	"github.com/juju/errors"
	"github.com/juju/jsonschema"
	"github.com/juju/loggo"
	"github.com/juju/schema"
	"github.com/juju/utils/clock"
	"gopkg.in/juju/environschema.v1"

	// "github.com/juju/utils/clock"

	"github.com/juju/juju/cloud"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
	providerCommon "github.com/juju/juju/provider/oci/common"
	providerNet "github.com/juju/juju/provider/oci/network"
	// ociNet "github.com/juju/juju/provider/oci/network"

	"gopkg.in/ini.v1"
)

var logger = loggo.GetLogger("juju.provider.oracle")

var configSchema = environschema.Fields{
	"compartment-id": {
		Description: "The OCID of the compartment in which juju has access to create resources.",
		Type:        environschema.Tstring,
	},
}

var configDefaults = schema.Defaults{
	"compartment-id": "",
}

var configFields = func() schema.Fields {
	fs, _, err := configSchema.ValidationSchema()
	if err != nil {
		panic(err)
	}
	return fs
}()

// EnvironProvider type implements environs.EnvironProvider interface
type EnvironProvider struct{}

type environConfig struct {
	*config.Config
	attrs map[string]interface{}
}

func (p EnvironProvider) newConfig(cfg *config.Config) (*environConfig, error) {
	valid, err := p.Validate(cfg, nil)
	if err != nil {
		return nil, err
	}
	return &environConfig{valid, valid.UnknownAttrs()}, nil
}

func (c *environConfig) compartmentID() *string {
	compartmentID := c.attrs["compartment-id"].(string)
	return &compartmentID
}

var _ config.ConfigSchemaSource = (*EnvironProvider)(nil)
var _ environs.ProviderSchema = (*EnvironProvider)(nil)

// Schema implements environs.ProviderSchema
func (o *EnvironProvider) Schema() environschema.Fields {
	fields, err := config.Schema(configSchema)
	if err != nil {
		panic(err)
	}
	return fields
}

// ConfigSchema implements config.ConfigSchemaSource
func (o *EnvironProvider) ConfigSchema() schema.Fields {
	return configFields
}

// ConfigDefaults implements config.ConfigSchemaSource
func (o *EnvironProvider) ConfigDefaults() schema.Defaults {
	return configDefaults
}

// credentialSection holds the keys present in one section of the OCI
// config file, as created by the OCI command line. This is only used
// during credential detection
type credentialSection struct {
	User        string
	Tenancy     string
	KeyFile     string
	PassPhrase  string
	Fingerprint string
	Region      string
}

var credentialSchema = map[cloud.AuthType]cloud.CredentialSchema{
	cloud.HTTPSigAuthType: {
		{
			"user", cloud.CredentialAttr{
				Description: "Username OCID",
			},
		},
		{
			"tenancy", cloud.CredentialAttr{
				Description: "Tenancy OCID",
			},
		},
		{
			"key-file", cloud.CredentialAttr{
				Description: "Path to private key",
			},
		},
		{
			"pass-phrase", cloud.CredentialAttr{
				Description: "Passphrase used to unlock key-file",
				Hidden:      true,
			},
		},
		{
			"fingerprint", cloud.CredentialAttr{
				Description: "Private key fingerprint",
			},
		},
		{
			"region", cloud.CredentialAttr{
				Description: "Region to log into",
			},
		},
	},
}

// Version implements environs.EnvironProvider.
func (e EnvironProvider) Version() int {
	return 0
}

// CloudSchema implements environs.EnvironProvider.
// OCI provider does not support custom schema.
func (e EnvironProvider) CloudSchema() *jsonschema.Schema {
	return nil
}

// Ping implements environs.EnvironProvider.
func (e *EnvironProvider) Ping(endpoint string) error {
	return errors.NotImplementedf("Ping")
}

// PrepareConfig implements environs.EnvironProvider.
func (e EnvironProvider) PrepareConfig(args environs.PrepareConfigParams) (*config.Config, error) {
	if err := validateCloudSpec(args.Cloud); err != nil {
		return nil, errors.Annotate(err, "validating cloud spec")
	}
	// TODO(gsamfira): Set default block storage backend
	return args.Config, nil
}

func validateCloudSpec(c environs.CloudSpec) error {
	if err := c.Validate(); err != nil {
		return errors.Trace(err)
	}
	if c.Credential == nil {
		return errors.NotValidf("missing credential")
	}
	if authType := c.Credential.AuthType(); authType != cloud.HTTPSigAuthType {
		return errors.NotSupportedf("%q auth-type", authType)
	}
	return nil
}

// Open implements environs.EnvironProvider.
func (e *EnvironProvider) Open(params environs.OpenParams) (environs.Environ, error) {
	logger.Infof("opening model %q", params.Config.Name())

	if err := validateCloudSpec(params.Cloud); err != nil {
		return nil, errors.Trace(err)
	}
	// TODO(gsamfira): error out if compartment-id is empty

	creds := params.Cloud.Credential.Attributes()

	provider := providerCommon.NewJujuConfigProvider(
		creds["user"], creds["tenancy"],
		creds["key-file"], creds["fingerprint"],
		creds["pass-phrase"], creds["region"])

	client, err := providerCommon.NewOciClient(provider)
	if err != nil {
		return nil, errors.Trace(err)
	}

	netEnviron, err := providerNet.NewNetworkEnviron(client)
	if err != nil {
		return nil, errors.Trace(err)
	}

	firewaller, err := providerNet.NewFirewaller(client)
	if err != nil {
		return nil, errors.Trace(err)
	}

	env := &Environ{
		cli:   client,
		p:     e,
		clock: clock.WallClock,
	}

	env.Networking = netEnviron
	env.Firewaller = firewaller

	if err := env.SetConfig(params.Config); err != nil {
		return nil, err
	}

	cfg := env.ecfg()
	if cfg.compartmentID() == nil {
		return nil, errors.New("compartment-id may not be empty")
	}

	return env, nil
}

// CredentialSchemas implements environs.ProviderCredentials.
func (e EnvironProvider) CredentialSchemas() map[cloud.AuthType]cloud.CredentialSchema {
	return credentialSchema
}

func validateKey(key, passphrase string) error {
	data, err := ioutil.ReadFile(key)
	if err != nil {
		return errors.Trace(err)
	}

	keyBlock, _ := pem.Decode(data)
	if keyBlock == nil {
		return errors.Errorf("invalid private key %s", key)
	}

	if x509.IsEncryptedPEMBlock(keyBlock) {
		if _, err := x509.DecryptPEMBlock(keyBlock, []byte(passphrase)); err != nil {
			return errors.Annotatef(err, "decrypting private key")
		}
	}

	return nil
}

// DetectCredentials implements environs.ProviderCredentials.
func (e EnvironProvider) DetectCredentials() (*cloud.CloudCredential, error) {
	cfg_file, err := ociConfigFile()
	if err != nil {
		return nil, errors.Trace(err)
	}

	cfg, err := ini.LooseLoad(cfg_file)
	if err != nil {
		return nil, errors.Trace(err)
	}
	cfg.NameMapper = ini.TitleUnderscore

	result := cloud.CloudCredential{
		AuthCredentials: make(map[string]cloud.Credential),
	}

	var defaultRegion string

	for _, val := range cfg.SectionStrings() {
		values := new(credentialSection)
		if err := cfg.Section(val).MapTo(values); err != nil {
			return nil, errors.Annotatef(err, "invalid value in section %s", val)
		}

		if values.User == "" || values.Tenancy == "" ||
			values.KeyFile == "" || values.Fingerprint == "" || values.Region == "" {
			return nil, errors.Errorf("missing required fields in config section %s", val)
		}
		if err := validateKey(values.KeyFile, values.PassPhrase); err != nil {
			return nil, errors.Trace(err)
		}

		if val == "DEFAULT" {
			defaultRegion = values.Region
		}
		httpSigCreds := cloud.NewCredential(
			cloud.HTTPSigAuthType,
			map[string]string{
				"user":        values.User,
				"tenancy":     values.Tenancy,
				"key-file":    values.KeyFile,
				"pass-phrase": values.PassPhrase,
				"fingerprint": values.Fingerprint,
				"region":      values.Region,
			},
		)
		httpSigCreds.Label = fmt.Sprintf("OCI credential %q", val)
		result.AuthCredentials[val] = httpSigCreds
	}
	result.DefaultRegion = defaultRegion
	return &result, nil
}

// FinalizeCredential implements environs.ProviderCredentials.
func (e EnvironProvider) FinalizeCredential(
	ctx environs.FinalizeCredentialContext,
	params environs.FinalizeCredentialParams) (*cloud.Credential, error) {

	return &params.Credential, nil
}

// Validate implements config.Validator.
func (e EnvironProvider) Validate(cfg, old *config.Config) (valid *config.Config, err error) {
	if err := config.Validate(cfg, old); err != nil {
		return nil, err
	}
	newAttrs, err := cfg.ValidateUnknownAttrs(
		schema.Fields{}, schema.Defaults{},
	)
	if err != nil {
		return nil, err
	}

	return cfg.Apply(newAttrs)
}
