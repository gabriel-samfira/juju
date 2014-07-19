package cloudinit

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/juju/names"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/cloudinit"
	"github.com/juju/juju/version"
)

type UserdataConfig interface {
	Configure() error
	ConfigureBasic() error
	ConfigureJuju() error
	Render() ([]byte, error)
}

// addAgentInfo adds agent-required information to the agent's directory
// and returns the agent directory name.
func addAgentInfo(cfg *MachineConfig, c *cloudinit.Config, tag names.Tag) (agent.Config, error) {
	acfg, err := cfg.agentConfig(tag)
	if err != nil {
		return nil, err
	}
	acfg.SetValue(agent.AgentServiceName, cfg.MachineAgentServiceName)
	cmds, err := acfg.WriteCommands(cfg.Series)
	if err != nil {
		return nil, errors.Annotate(err, "failed to write commands")
	}
	c.AddScripts(cmds...)
	return acfg, nil
}

func NewUserdataConfig(cfg *MachineConfig, c *cloudinit.Config) (UserdataConfig, error) {
	operatingSystem, err := version.GetOSFromSeries(cfg.Series)
	if err != nil {
		return nil, err
	}

	switch operatingSystem {
	case version.Ubuntu:
		return newUbuntuConfig(cfg, c)
	case version.Windows:
		return newWindowsConfig(cfg, c)
	default:
		return nil, fmt.Errorf("Unsupported OS %s", cfg.Series)
	}
}
