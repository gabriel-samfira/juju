// Copyright 2012, 2013, 2014, 2015 Canonical Ltd.
// Copyright 2014, 2015 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package cloudconfig

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/juju/errors"
	"github.com/juju/names"
	"github.com/juju/utils/featureflag"

	"github.com/juju/juju/juju/osenv"
	"github.com/juju/juju/juju/paths"
	"github.com/juju/juju/version"
)

type aclType string

const (
	fileSystem    aclType = "FileSystem"
	registryEntry aclType = "Registry"
)

type windowsConfigure struct {
	baseConfigure
}

// Configure updates the provided cloudinit.Config with
// configuration to initialize a Juju machine agent.
func (w *windowsConfigure) Configure() error {
	if err := w.ConfigureBasic(); err != nil {
		return err
	}
	return w.ConfigureJuju()
}

func (w *windowsConfigure) ConfigureBasic() error {

	series := w.icfg.Series
	tmpDir, err := paths.TempDir(series)
	if err != nil {
		return err
	}
	renderer := w.conf.ShellRenderer()
	dataDir := renderer.FromSlash(w.icfg.DataDir)
	baseDir := renderer.FromSlash(filepath.Dir(tmpDir))
	binDir := renderer.Join(baseDir, "bin")

	w.conf.AddScripts(
		fmt.Sprintf(`%s`, winPowershellHelperFunctions),
	)
	if version.IsNanoSeries(w.icfg.Series) == false {
		w.conf.AddScripts(
			fmt.Sprintf(`%s`, addJujudUser),
			fmt.Sprintf(`%s`, addTgzCapability),
		)
	}

	w.conf.AddScripts(
		// Some providers create a baseDir before this step, but we need to
		// make sure it exists before applying icacls
		fmt.Sprintf(`mkdir -Force "%s"`, renderer.FromSlash(baseDir)),
		fmt.Sprintf(`mkdir %s`, renderer.FromSlash(tmpDir)),
		fmt.Sprintf(`mkdir "%s"`, binDir),
		fmt.Sprintf(`mkdir "%s\locks"`, renderer.FromSlash(dataDir)),
	)

	// This is necessary for setACLs to work
	w.conf.AddScripts(`$adminsGroup = (New-Object System.Security.Principal.SecurityIdentifier("S-1-5-32-544")).Translate([System.Security.Principal.NTAccount])`)
	if version.IsNanoSeries(w.icfg.Series) == false {
		w.conf.AddScripts(setACLs(renderer.FromSlash(baseDir), fileSystem)...)
	}
	w.conf.AddScripts(`setx /m PATH "$env:PATH;C:\Juju\bin\"`)
	noncefile := renderer.Join(dataDir, NonceFile)
	w.conf.AddScripts(
		fmt.Sprintf(`Set-Content "%s" "%s"`, noncefile, shquote(w.icfg.MachineNonce)),
	)
	return nil
}

func (w *windowsConfigure) ConfigureJuju() error {
	if err := w.icfg.VerifyConfig(); err != nil {
		return errors.Trace(err)
	}
	if w.icfg.Bootstrap == true {
		// Bootstrap machine not supported on windows
		return errors.Errorf("bootstrapping is not supported on windows")
	}

	toolsJson, err := json.Marshal(w.icfg.Tools)
	if err != nil {
		return errors.Annotate(err, "while serializing the tools")
	}
	w.downloadAndUnzipTools(toolsJson)

	for _, cmd := range CreateJujuRegistryKeyCmds(w.icfg.Series) {
		w.conf.AddRunCmd(cmd)
	}

	machineTag := names.NewMachineTag(w.icfg.MachineId)
	_, err = w.addAgentInfo(machineTag)
	if err != nil {
		return errors.Trace(err)
	}
	return w.addMachineAgentToBoot()
}

func (w windowsConfigure) downloadAndUnzipTools(toolsJson []byte) {
	renderer := w.conf.ShellRenderer()
	w.conf.AddScripts(
		fmt.Sprintf(`$binDir="%s"`, renderer.FromSlash(w.icfg.JujuTools())),
		fmt.Sprintf(`mkdir '%s'`, renderer.FromSlash(w.icfg.LogDir)),
		`mkdir $binDir`,
	)
	if version.IsNanoSeries(w.icfg.Series) == true {
		w.conf.AddScripts(
			`$tmpBinDir=$binDir.Replace('\', '\\')`,
			fmt.Sprintf(`& "C:\Cloudbase-Init\Python\python.exe" -c "from urllib import request; import ssl; ssl._create_default_https_context = ssl._create_unverified_conte
xt; request.urlretrieve('%s', '$tmpBinDir\\tools.tar.gz')"`, w.icfg.Tools.URL),
		)
	} else {
		w.conf.AddScripts(
			`$WebClient = New-Object System.Net.WebClient`,
			`[System.Net.ServicePointManager]::ServerCertificateValidationCallback = {$true}`,
			fmt.Sprintf(`ExecRetry { $WebClient.DownloadFile('%s', "$binDir\tools.tar.gz") }`, w.icfg.Tools.URL),
		)
	}
	w.conf.AddScripts(
		`$dToolsHash = Get-FileSHA256 -FilePath "$binDir\tools.tar.gz"`,
		fmt.Sprintf(`$dToolsHash > "$binDir\juju%s.sha256"`,
			w.icfg.Tools.Version),
		fmt.Sprintf(`if ($dToolsHash.ToLower() -ne "%s"){ Throw "Tools checksum mismatch"}`,
			w.icfg.Tools.SHA256),
	)
	if version.IsNanoSeries(w.icfg.Series) == true {
		w.conf.AddScripts(
			`$tmpBinDir=$binDir.Replace('\', '\\')`,
			`& "C:\Cloudbase-Init\Python\python.exe" -c "import tarfile;archive = tarfile.open('$tmpBinDir\\tools.tar.gz');archive.extractall(path='$tmpBinDir')"`,
		)
	} else {
		w.conf.AddRunCmd(
			fmt.Sprintf(`GUnZip-File -infile $binDir\tools.tar.gz -outdir $binDir`),
		)
	}
	w.conf.AddScripts(
		`rm "$binDir\tools.tar*"`,
		fmt.Sprintf(`Set-Content $binDir\downloaded-tools.txt '%s'`, string(toolsJson)),
	)
}

// CreateJujuRegistryKey is going to create a juju registry key and set
// permissions on it such that it's only accessible to administrators
// It is exported because it is used in an upgrade step
func CreateJujuRegistryKeyCmds(series string) []string {
	aclCmds := setACLs(osenv.JujuRegistryKey, registryEntry)

	regCmds := []string{
		fmt.Sprintf(`New-Item -Path '%s'`, osenv.JujuRegistryKey),
	}

	regCmds = append(regCmds, fmt.Sprintf(`New-ItemProperty -Path '%s' -Name '%s'`,
		osenv.JujuRegistryKey,
		osenv.JujuFeatureFlagEnvKey))

	if version.IsNanoSeries(series) == false {
		regCmds = append(regCmds, aclCmds...)
	}

	regCmds = append(regCmds, fmt.Sprintf(`Set-ItemProperty -Path '%s' -Name '%s' -Value '%s'`,
		osenv.JujuRegistryKey,
		osenv.JujuFeatureFlagEnvKey,
		featureflag.AsEnvironmentValue()))

	return regCmds
}

func setACLs(path string, permType aclType) []string {
	ruleModel := `$rule = New-Object System.Security.AccessControl.%sAccessRule %s`
	permModel := `%s = "%s", "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow"`
	adminPermVar := `$adminPerm`
	jujudPermVar := `$jujudPerm`
	return []string{
		fmt.Sprintf(`$acl = Get-Acl -Path '%s'`, path),

		// Reset the ACL's on it and add administrator access only.
		`$acl.SetAccessRuleProtection($true, $false)`,

		// $adminsGroup must be defined before calling setACLs
		fmt.Sprintf(permModel, adminPermVar, `$adminsGroup`),
		fmt.Sprintf(permModel, jujudPermVar, `jujud`),
		fmt.Sprintf(ruleModel, permType, adminPermVar),
		`$acl.AddAccessRule($rule)`,
		fmt.Sprintf(ruleModel, permType, jujudPermVar),
		`$acl.AddAccessRule($rule)`,
		fmt.Sprintf(`Set-Acl -Path '%s' -AclObject $acl`, path),
	}
}
