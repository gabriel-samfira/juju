// +build !windows

package reboot

import (
	// "fmt"
	"time"

	"github.com/juju/juju/apiserver/params"
)

// executeAction will do a reboot or shutdown after given number of seconds
// this function executes the operating system's reboot binary with apropriate
// parameters to schedule the reboot
// If action is params.ShouldDoNothing, it will return immediately.
func executeAction(action params.RebootAction, errChan chan error) {
	if action == params.ShouldDoNothing {
		errChan <- nil
		return
	}
	args := []string{"shutdown"}
	switch action {
	case params.ShouldReboot:
		args = append(args, "-r")
	case params.ShouldShutdown:
		args = append(args, "-h")
	}
	args = append(args, "now")

	errChan <- runCommand(args)
}

func scheduleReboot(action params.RebootAction, seconds int) <-chan error {
	errChan := make(chan error)
	time.Sleep(time.Duration(seconds) * time.Second)
	go executeAction(action, errChan)
	return errChan
}
