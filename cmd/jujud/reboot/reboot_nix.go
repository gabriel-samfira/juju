// +build !windows

package reboot

import (
	"github.com/juju/juju/apiserver/params"
)

// executeAction will do a reboot or shutdown after given number of seconds
// this function executes the operating system's reboot binary with apropriate
// parameters to schedule the reboot
// If action is params.ShouldDoNothing, it will return immediately.
func executeAction(action params.RebootAction) error {
	if action == params.ShouldDoNothing {
		return nil
	}
	args := []string{"shutdown"}
	switch action {
	case params.ShouldReboot:
		args = append(args, "-r")
	case params.ShouldShutdown:
		args = append(args, "-h")
	}
	args = append(args, "now")

	return runCommand(args)
}
