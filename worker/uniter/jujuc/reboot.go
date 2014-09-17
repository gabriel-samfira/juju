package jujuc

import (
	"github.com/juju/cmd"
	"launchpad.net/gnuflag"
)

// JujuRebootCommand implements the juju-reboot command.
type JujuRebootCommand struct {
	cmd.CommandBase
	ctx Context
	Now bool
}

func NewJujuRebootCommand(ctx Context) cmd.Command {
	return &JujuRebootCommand{ctx: ctx}
}

func (c *JujuRebootCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "juju-reboot",
		Args:    "<--now>",
		Purpose: "Reboot the machine we are running on",
	}
}

func (c *JujuRebootCommand) SetFlags(f *gnuflag.FlagSet) {
	f.BoolVar(&c.Now, "now", false, "log at debug level")
}

func (c *JujuRebootCommand) Init(args []string) error {
	return nil
}

func (c *JujuRebootCommand) Run(ctx *cmd.Context) error {
	logger.Debugf("Running juju-reboot for: %v", c.ctx.UnitName())

	rebootPrio := RebootAfterHook
	if c.Now {
		rebootPrio = RebootNow
	}

	return c.ctx.RequestReboot(rebootPrio)
}
