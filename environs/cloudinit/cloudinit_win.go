package cloudinit

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/juju/names"

	"github.com/juju/juju/agent/tools"
	"github.com/juju/juju/cloudinit"
	"github.com/juju/juju/juju/paths"
)

type windowsConfigure struct {
	mcfg   *MachineConfig
	conf   *cloudinit.Config
	render cloudinit.Renderer
}

func (w *windowsConfigure) init() error {
	renderer, err := cloudinit.NewRenderer(w.mcfg.Series)
	if err != nil {
		return err
	}
	w.render = renderer
	return nil
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

	series := w.mcfg.Series
	tmpDir, err := paths.TempDir(series)
	if err != nil {
		return err
	}
	dataDir := w.mcfg.DataDir
	baseDir := filepath.Dir(tmpDir)
	binDir := w.render.PathJoin(baseDir, "bin")

	w.conf.AddScripts(
		fmt.Sprintf(`%s`, winPowershellHelperFunctions),
		fmt.Sprintf(`icacls "%s" /grant "jujud:(OI)(CI)(F)" /T`, w.render.FromSlash(baseDir)),
		fmt.Sprintf(`mkdir %s`, w.render.FromSlash(tmpDir)),
		fmt.Sprintf(`mkdir "%s"`, binDir),
		fmt.Sprintf(`%s`, winSetPasswdScript),
		fmt.Sprintf(`Start-ProcessAsUser -Command $powershell -Arguments "-File C:\juju\bin\save_pass.ps1 $juju_passwd" -Credential $jujuCreds`),
		fmt.Sprintf(`mkdir "%s\locks"`, w.render.FromSlash(dataDir)),
		fmt.Sprintf(`Start-ProcessAsUser -Command $cmdExe -Arguments '/C setx PATH "%%PATH%%;C:\Juju\bin"' -Credential $jujuCreds`),
	)
	noncefile := w.render.PathJoin(dataDir, NonceFile)
	w.conf.AddScripts(
		fmt.Sprintf(`Set-Content "%s" "%s"`, noncefile, shquote(w.mcfg.MachineNonce)),
	)
	return nil
}

func (w *windowsConfigure) ConfigureJuju() error {
	if err := verifyConfig(w.mcfg); err != nil {
		return err
	}
	toolsJson, err := json.Marshal(w.mcfg.Tools)
	if err != nil {
		return err
	}
	var python string = `${env:ProgramFiles(x86)}\Cloudbase Solutions\Cloudbase-Init\Python27\python.exe`
	w.conf.AddScripts(
		fmt.Sprintf(`$binDir="%s"`, w.render.FromSlash(w.mcfg.jujuTools())),
		`$tmpBinDir=$binDir.Replace('\', '\\')`,
		fmt.Sprintf(`mkdir '%s'`, w.render.FromSlash(w.mcfg.LogDir)),
		`mkdir $binDir`,
		`$WebClient = New-Object System.Net.WebClient`,
		`[System.Net.ServicePointManager]::ServerCertificateValidationCallback = {$true}`,
		fmt.Sprintf(`ExecRetry { $WebClient.DownloadFile('%s', "$binDir\tools.tar.gz") }`, w.mcfg.Tools.URL),
		`$dToolsHash = (Get-FileHash -Algorithm SHA256 "$binDir\tools.tar.gz").hash`,
		fmt.Sprintf(`$dToolsHash > "$binDir\juju%s.sha256"`,
			w.mcfg.Tools.Version),
		fmt.Sprintf(`if ($dToolsHash.ToLower() -ne "%s"){ Throw "Tools checksum mismatch"}`,
			w.mcfg.Tools.SHA256),
		fmt.Sprintf(`& "%s" -c "import tarfile;archive = tarfile.open('$tmpBinDir\\tools.tar.gz');archive.extractall(path='$tmpBinDir')"`, python),
		`rm "$binDir\tools.tar*"`,
		fmt.Sprintf(`Set-Content $binDir\downloaded-tools.txt '%s'`, string(toolsJson)),
	)

	machineTag := names.NewMachineTag(w.mcfg.MachineId)
	_, err = addAgentInfo(w.mcfg, w.conf, machineTag)
	if err != nil {
		return err
	}
	return w.addMachineAgentToBoot(machineTag.String())
}

// MachineAgentWindowsService returns the powershell command for a machine agent service
// based on the tag and machineId passed in.
// TODO: find a better place for this
func (w *windowsConfigure) machineAgentWindowsService(name, toolsDir, tag string) []string {
	jujud := filepath.Join(toolsDir, "jujud.exe")

	serviceString := fmt.Sprintf(`"%s" machine --data-dir "%s" --machine-id "%s" --debug`,
		w.render.FromSlash(jujud),
		w.render.FromSlash(w.mcfg.DataDir),
		tag)

	cmd := []string{
		fmt.Sprintf(`New-Service -Credential $jujuCreds -Name '%s' -DisplayName 'Jujud machine agent' '%s'`, name, serviceString),
		fmt.Sprintf(`cmd.exe /C sc config %s start=delayed-auto`, name),
		fmt.Sprintf(`Start-Service %s`, name),
	}
	return cmd
}

func (w *windowsConfigure) addMachineAgentToBoot(tag string) error {
	// Make the agent run via a symbolic link to the actual tools
	// directory, so it can upgrade itself without needing to change
	// the upstart script.
	toolsDir := tools.ToolsDir(w.mcfg.DataDir, tag)
	w.conf.AddScripts(
		fmt.Sprintf(
			`cmd.exe /C mklink %s %v`,
			w.render.FromSlash(toolsDir),
			w.mcfg.Tools.Version),
	)
	name := w.mcfg.MachineAgentServiceName
	cmds := w.machineAgentWindowsService(name, toolsDir, tag)
	w.conf.AddScripts(cmds...)
	return nil
}

func (w *windowsConfigure) Render() ([]byte, error) {
	return w.render.Render(w.conf)
}

func newWindowsConfig(mcfg *MachineConfig, conf *cloudinit.Config) (*windowsConfigure, error) {
	cfg := &windowsConfigure{
		mcfg: mcfg,
		conf: conf,
	}
	err := cfg.init()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
