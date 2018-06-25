// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package lxd_test

import (
	"net"
	"os"
	"path/filepath"

	"github.com/golang/mock/gomock"
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/cloud"
	"github.com/juju/juju/environs"
	envtesting "github.com/juju/juju/environs/testing"
	"github.com/juju/juju/juju/osenv"
	"github.com/juju/juju/provider/lxd"
	coretesting "github.com/juju/juju/testing"
)

type credentialsSuite struct {
	lxd.BaseSuite
}

var _ = gc.Suite(&credentialsSuite{})

func (s *credentialsSuite) TestCredentialSchemas(c *gc.C) {
	provider := lxd.NewProvider()
	envtesting.AssertProviderAuthTypes(c, provider, "interactive", "certificate")
}

func (s *credentialsSuite) createProvider(ctrl *gomock.Controller) (environs.EnvironProvider,
	environs.ProviderCredentials,
	*lxd.MockProviderLXDServer,
	*lxd.MockLXDCertificateReadWriter,
	*lxd.MockLXDCertificateGenerator,
) {
	server := lxd.NewMockProviderLXDServer(ctrl)

	certReadWriter := lxd.NewMockLXDCertificateReadWriter(ctrl)
	certGenerator := lxd.NewMockLXDCertificateGenerator(ctrl)
	creds := lxd.NewProviderCredentials(
		certReadWriter,
		certGenerator,
		net.LookupHost,
		net.InterfaceAddrs,
		func() (lxd.ProviderLXDServer, error) {
			return server, nil
		},
	)

	provider := lxd.NewProviderWithMocks(creds, utils.GetAddressForInterface, func() (lxd.ProviderLXDServer, error) {
		return server, nil
	})
	return provider, creds, server, certReadWriter, certGenerator
}

func (s *credentialsSuite) TestDetectCredentialsUsesJujuCert(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	_, provider, server, certsIO, _ := s.createProvider(ctrl)

	server.EXPECT().GetCertificate(gomock.Any()).Return(nil, "", nil)
	server.EXPECT().GetServerEnvironmentCertificate().Return("server-cert", nil)

	path := osenv.JujuXDGDataHomePath("lxd")
	certsIO.EXPECT().Read(path).Return([]byte(coretesting.CACert), []byte(coretesting.CAKey), nil)

	credentials, err := provider.DetectCredentials()

	expected := cloud.NewCredential(
		cloud.CertificateAuthType,
		map[string]string{
			"client-cert": coretesting.CACert,
			"client-key":  coretesting.CAKey,
			"server-cert": "server-cert",
		},
	)
	expected.Label = `LXD credential "localhost"`

	c.Assert(err, jc.ErrorIsNil)
	c.Assert(credentials, jc.DeepEquals, &cloud.CloudCredential{
		AuthCredentials: map[string]cloud.Credential{
			"localhost": expected,
		},
	})
}

func (s *credentialsSuite) TestDetectCredentialsFailsWithJujuCert(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	_, provider, _, certsIO, _ := s.createProvider(ctrl)

	path := osenv.JujuXDGDataHomePath("lxd")
	certsIO.EXPECT().Read(path).Return(nil, nil, errors.NotValidf("certs"))

	_, err := provider.DetectCredentials()
	c.Assert(errors.Cause(err), gc.ErrorMatches, "certs not valid")
}

func (s *credentialsSuite) TestDetectCredentialsUsesLXCCert(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	_, provider, server, certsIO, _ := s.createProvider(ctrl)

	server.EXPECT().GetCertificate(gomock.Any()).Return(nil, "", nil)
	server.EXPECT().GetServerEnvironmentCertificate().Return("server-cert", nil)

	path := osenv.JujuXDGDataHomePath("lxd")
	certsIO.EXPECT().Read(path).Return(nil, nil, os.ErrNotExist)

	path = filepath.Join(utils.Home(), ".config", "lxc")
	certsIO.EXPECT().Read(path).Return([]byte(coretesting.CACert), []byte(coretesting.CAKey), nil)

	credentials, err := provider.DetectCredentials()

	expected := cloud.NewCredential(
		cloud.CertificateAuthType,
		map[string]string{
			"client-cert": coretesting.CACert,
			"client-key":  coretesting.CAKey,
			"server-cert": "server-cert",
		},
	)
	expected.Label = `LXD credential "localhost"`

	c.Assert(err, jc.ErrorIsNil)
	c.Assert(credentials, jc.DeepEquals, &cloud.CloudCredential{
		AuthCredentials: map[string]cloud.Credential{
			"localhost": expected,
		},
	})
}

func (s *credentialsSuite) TestDetectCredentialsFailsWithJujuAndLXCCert(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	_, provider, _, certsIO, _ := s.createProvider(ctrl)

	path := osenv.JujuXDGDataHomePath("lxd")
	certsIO.EXPECT().Read(path).Return(nil, nil, os.ErrNotExist)

	path = filepath.Join(utils.Home(), ".config", "lxc")
	certsIO.EXPECT().Read(path).Return(nil, nil, errors.NotValidf("certs"))

	_, err := provider.DetectCredentials()
	c.Assert(errors.Cause(err), gc.ErrorMatches, "certs not valid")
}

func (s *credentialsSuite) TestDetectCredentialsGeneratesCert(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	_, provider, server, certsIO, certsGen := s.createProvider(ctrl)

	server.EXPECT().GetCertificate(gomock.Any()).Return(nil, "", nil)
	server.EXPECT().GetServerEnvironmentCertificate().Return("server-cert", nil)

	path := osenv.JujuXDGDataHomePath("lxd")
	certsIO.EXPECT().Read(path).Return(nil, nil, os.ErrNotExist)
	certsIO.EXPECT().Write(path, []byte(coretesting.CACert), []byte(coretesting.CAKey)).Return(nil)

	path = filepath.Join(utils.Home(), ".config", "lxc")
	certsIO.EXPECT().Read(path).Return(nil, nil, os.ErrNotExist)

	certsGen.EXPECT().Generate(true).Return([]byte(coretesting.CACert), []byte(coretesting.CAKey), nil)

	credential := cloud.NewCredential(
		cloud.CertificateAuthType,
		map[string]string{
			"client-cert": coretesting.CACert,
			"client-key":  coretesting.CAKey,
			"server-cert": "server-cert",
		},
	)
	credential.Label = `LXD credential "localhost"`

	credentials, err := provider.DetectCredentials()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(credentials, jc.DeepEquals, &cloud.CloudCredential{
		AuthCredentials: map[string]cloud.Credential{
			"localhost": credential,
		},
	})
}

func (s *credentialsSuite) TestDetectCredentialsGeneratesCertFailsToWriteOnError(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	_, provider, _, certsIO, certsGen := s.createProvider(ctrl)

	path := osenv.JujuXDGDataHomePath("lxd")
	certsIO.EXPECT().Read(path).Return(nil, nil, os.ErrNotExist)

	path = filepath.Join(utils.Home(), ".config", "lxc")
	certsIO.EXPECT().Read(path).Return(nil, nil, os.ErrNotExist)

	certsGen.EXPECT().Generate(true).Return(nil, nil, errors.Errorf("bad"))

	credential := cloud.NewCredential(
		cloud.CertificateAuthType,
		map[string]string{
			"client-cert": coretesting.CACert,
			"client-key":  coretesting.CAKey,
			"server-cert": "server-cert",
		},
	)
	credential.Label = `LXD credential "localhost"`

	_, err := provider.DetectCredentials()
	c.Assert(errors.Cause(err), gc.ErrorMatches, "bad")
}

func (s *credentialsSuite) TestDetectCredentialsGeneratesCertFailsToGetCertificateOnError(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	_, provider, _, certsIO, certsGen := s.createProvider(ctrl)

	path := osenv.JujuXDGDataHomePath("lxd")
	certsIO.EXPECT().Read(path).Return(nil, nil, os.ErrNotExist)
	certsIO.EXPECT().Write(path, []byte(coretesting.CACert), []byte(coretesting.CAKey)).Return(errors.Errorf("bad"))

	path = filepath.Join(utils.Home(), ".config", "lxc")
	certsIO.EXPECT().Read(path).Return(nil, nil, os.ErrNotExist)

	certsGen.EXPECT().Generate(true).Return([]byte(coretesting.CACert), []byte(coretesting.CAKey), nil)

	credential := cloud.NewCredential(
		cloud.CertificateAuthType,
		map[string]string{
			"client-cert": coretesting.CACert,
			"client-key":  coretesting.CAKey,
			"server-cert": "server-cert",
		},
	)
	credential.Label = `LXD credential "localhost"`

	_, err := provider.DetectCredentials()
	c.Assert(errors.Cause(err), gc.ErrorMatches, "bad")
}

/*
func (s *credentialsSuite) TestFinalizeCredentialLocal(c *gc.C) {
	cert, _ := s.TestingCert(c)
	out, err := s.Provider.FinalizeCredential(cmdtesting.Context(c), environs.FinalizeCredentialParams{
		CloudEndpoint: "1.2.3.4",
		Credential: cloud.NewCredential(cloud.CertificateAuthType, map[string]string{
			"client-cert": string(cert.CertPEM),
			"client-key":  string(cert.KeyPEM),
		}),
	})
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(out.AuthType(), gc.Equals, cloud.CertificateAuthType)
	c.Assert(out.Attributes(), jc.DeepEquals, map[string]string{
		"client-cert": string(cert.CertPEM),
		"client-key":  string(cert.KeyPEM),
		"server-cert": "server-cert",
	})
	s.Stub.CheckCallNames(c,
		"LookupHost",
		"InterfaceAddrs",
		"GetCertificate",
		"ServerStatus",
	)
}
/*
func (s *credentialsSuite) TestFinalizeCredentialLocalAddCert(c *gc.C) {
	s.Stub.SetErrors(errors.New("not found"))
	cert, _ := s.TestingCert(c)
	out, err := s.Provider.FinalizeCredential(cmdtesting.Context(c), environs.FinalizeCredentialParams{
		CloudEndpoint: "", // skips host lookup
		Credential: cloud.NewCredential(cloud.CertificateAuthType, map[string]string{
			"client-cert": string(cert.CertPEM),
			"client-key":  string(cert.KeyPEM),
		}),
	})
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(out.AuthType(), gc.Equals, cloud.CertificateAuthType)
	c.Assert(out.Attributes(), jc.DeepEquals, map[string]string{
		"client-cert": string(cert.CertPEM),
		"client-key":  string(cert.KeyPEM),
		"server-cert": "server-cert",
	})
	s.Stub.CheckCallNames(c,
		"GetCertificate",
		"CreateClientCertificate",
		"ServerStatus",
	)
}

func (s *credentialsSuite) TestFinalizeCredentialLocalAddCertAlreadyThere(c *gc.C) {
	// If we get back an error from CreateClientCertificate, we'll make another
	// call to GetCertificate. If that call succeeds, then we assume
	// that the CreateClientCertificate failure was due to a concurrent call.
	s.Stub.SetErrors(
		errors.New("not found"),
		errors.New("UNIQUE constraint failed: certificates.fingerprint"),
	)
	cert, _ := s.TestingCert(c)
	out, err := s.Provider.FinalizeCredential(cmdtesting.Context(c), environs.FinalizeCredentialParams{
		CloudEndpoint: "", // skips host lookup
		Credential: cloud.NewCredential(cloud.CertificateAuthType, map[string]string{
			"client-cert": string(cert.CertPEM),
			"client-key":  string(cert.KeyPEM),
		}),
	})
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(out.AuthType(), gc.Equals, cloud.CertificateAuthType)
	c.Assert(out.Attributes(), jc.DeepEquals, map[string]string{
		"client-cert": string(cert.CertPEM),
		"client-key":  string(cert.KeyPEM),
		"server-cert": "server-cert",
	})
	s.Stub.CheckCallNames(c,
		"GetCertificate",
		"CreateClientCertificate",
		"GetCertificate",
		"ServerStatus",
	)
}

func (s *credentialsSuite) TestFinalizeCredentialLocalAddCertFatal(c *gc.C) {
	// If we get back an error from CreateClientCertificate, we'll make another
	// call to GetCertificate. If that call fails with "not found", then
	// we assume that the CreateClientCertificate failure is fatal.
	s.Stub.SetErrors(
		errors.New("not found"),
		errors.New("some fatal error"),
		errors.New("not found"),
	)
	cert, _ := s.TestingCert(c)
	_, err := s.Provider.FinalizeCredential(cmdtesting.Context(c), environs.FinalizeCredentialParams{
		CloudEndpoint: "", // skips host lookup
		Credential: cloud.NewCredential(cloud.CertificateAuthType, map[string]string{
			"client-cert": string(cert.CertPEM),
			"client-key":  string(cert.KeyPEM),
		}),
	})
	c.Assert(err, gc.ErrorMatches, `adding certificate "juju": some fatal error`)
}

func (s *credentialsSuite) TestFinalizeCredentialNonLocal(c *gc.C) {
	// Patch the interface addresses for the calling machine, so
	// it appears that we're not on the LXD server host.
	s.PatchValue(&s.InterfaceAddrs, []net.Addr{&net.IPNet{IP: net.ParseIP("8.8.8.8")}})
	in := cloud.NewCredential(cloud.CertificateAuthType, map[string]string{
		"client-cert": "foo",
		"client-key":  "bar",
	})
	_, err := s.Provider.FinalizeCredential(cmdtesting.Context(c), environs.FinalizeCredentialParams{
		CloudEndpoint: "8.8.8.8",
		Credential:    in,
	})
	c.Assert(err, gc.ErrorMatches, `
cannot auto-generate credential for remote LXD

Until support is added for verifying and authenticating to remote LXD hosts,
you must generate the credential on the LXD host, and add the credential to
this client using "juju add-credential localhost".

See: https://jujucharms.com/docs/stable/clouds-LXD
`[1:])
}

func (s *credentialsSuite) TestFinalizeCredentialLocalInteractive(c *gc.C) {
	cert, _ := s.TestingCert(c)
	home := c.MkDir()
	utils.SetHome(home)
	s.writeFile(c, filepath.Join(home, ".config/lxc/client.crt"), string(cert.CertPEM))
	s.writeFile(c, filepath.Join(home, ".config/lxc/client.key"), string(cert.KeyPEM))

	ctx := cmdtesting.Context(c)
	out, err := s.Provider.FinalizeCredential(ctx, environs.FinalizeCredentialParams{
		CloudEndpoint: "1.2.3.4",
		Credential:    cloud.NewCredential("interactive", map[string]string{}),
	})
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(out.AuthType(), gc.Equals, cloud.CertificateAuthType)
	c.Assert(out.Attributes(), jc.DeepEquals, map[string]string{
		"client-cert": string(cert.CertPEM),
		"client-key":  string(cert.KeyPEM),
		"server-cert": "server-cert",
	})
	s.Stub.CheckCallNames(c,
		"LookupHost",
		"InterfaceAddrs",
		"GetCertificate",
		"ServerStatus",
	)
}

func (s *credentialsSuite) TestFinalizeCredentialNonLocalInteractive(c *gc.C) {
	cert, _ := s.TestingCert(c)
	home := c.MkDir()
	utils.SetHome(home)
	s.writeFile(c, filepath.Join(home, ".config/lxc/client.crt"), string(cert.CertPEM))
	s.writeFile(c, filepath.Join(home, ".config/lxc/client.key"), string(cert.KeyPEM))

	// Patch the interface addresses for the calling machine, so
	// it appears that we're not on the LXD server host.
	s.PatchValue(&s.InterfaceAddrs, []net.Addr{&net.IPNet{IP: net.ParseIP("8.8.8.8")}})
	_, err := s.Provider.FinalizeCredential(cmdtesting.Context(c), environs.FinalizeCredentialParams{
		CloudEndpoint: "8.8.8.8",
		Credential:    cloud.NewCredential("interactive", map[string]string{}),
	})
	c.Assert(err, gc.ErrorMatches, `
certificate upload for remote LXD unsupported

Until support is added for verifying and authenticating to remote LXD hosts,
you must generate the credential on the LXD host, and add the credential to
this client using "juju add-credential localhost".

See: https://jujucharms.com/docs/stable/clouds-LXD
`[1:])
}

func (s *credentialsSuite) writeFile(c *gc.C, path, content string) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	c.Assert(err, jc.ErrorIsNil)
	err = ioutil.WriteFile(path, []byte(content), 0600)
	c.Assert(err, jc.ErrorIsNil)
}
*/
