// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package oci

import (
	// "github.com/juju/errors"
	"github.com/juju/jsonschema"
	"github.com/juju/loggo"
	// "github.com/juju/schema"
	// "github.com/juju/utils/clock"

	"github.com/juju/juju/cloud"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
)

var logger = loggo.GetLogger("juju.provider.oracle")

const (
	providerType = "oci"
)

// EnvironProvider type implements environs.EnvironProvider interface
type EnvironProvider struct{}

// Version implements environs.EnvironProvider.
func (e EnvironProvider) Version() int {
	return 0
}

// CloudSchema implements environs.EnvironProvider.
func (e EnvironProvider) CloudSchema() *jsonschema.Schema {
	return nil
}

// Ping implements environs.EnvironProvider.
func (e EnvironProvider) Ping(endpoint string) error {
	return nil
}

// PrepareConfig implements environs.EnvironProvider.
func (e EnvironProvider) PrepareConfig(args environs.PrepareConfigParams) (*config.Config, error) {
	return nil, nil
}

// Open implements environs.EnvironProvider.
func (e EnvironProvider) Open(params environs.OpenParams) (environs.Environ, error) {
	return nil, nil
}

// CredentialSchemas implements environs.ProviderCredentials.
func (e EnvironProvider) CredentialSchemas() map[cloud.AuthType]cloud.CredentialSchema {
	return map[cloud.AuthType]cloud.CredentialSchema{}
}

// DetectCredentials implements environs.ProviderCredentials.
func (e EnvironProvider) DetectCredentials() (*cloud.CloudCredential, error) {
	return nil, nil
}

// FinalizeCredential implements environs.ProviderCredentials.
func (e EnvironProvider) FinalizeCredential(
	ctx environs.FinalizeCredentialContext,
	params environs.FinalizeCredentialParams) (*cloud.Credential, error) {
	return nil, nil
}

// Validate implements config.Validator.
func (e EnvironProvider) Validate(cfg, old *config.Config) (valid *config.Config, err error) {
	return nil, nil
}
