// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package store

import (
	"github.com/juju/charm/v8"
	csparams "github.com/juju/charmrepo/v6/csclient/params"
	apicharm "github.com/juju/juju/api/charms"
	"gopkg.in/macaroon.v2"
)

// CharmAdder defines a subset of the api client needed to add a
// charm.
type CharmAdder interface {
	AddLocalCharm(*charm.URL, charm.Charm, bool) (*charm.URL, error) // not used in utils
	AddCharm(*charm.URL, csparams.Channel, bool) error
	AddCharmWithAuthorization(*charm.URL, csparams.Channel, *macaroon.Macaroon, bool) error
}

// charmrepoForDeploy is a stripped-down version of the
// gopkg.in/juju/charmrepo.v4 Interface interface. It is
// used by tests that embed a DeploySuiteBase.
type CharmrepoForDeploy interface {
	GetBundle(bundleURL *charm.URL, path string) (charm.Bundle, error)
	ResolveWithPreferredChannel(*charm.URL, csparams.Channel) (*charm.URL, csparams.Channel, []string, error)
}

// MacaroonGetter defines a subset of a charmstore client,
// as required by different application commands.
type MacaroonGetter interface {
	Get(endpoint string, extra interface{}) error
}

// CharmsAPI is functionality needed by the CharmAdapter from the Charms API.
type CharmsAPI interface {
	ResolveCharms(charms []apicharm.CharmToResolve) ([]apicharm.ResolvedCharm, error)
}
