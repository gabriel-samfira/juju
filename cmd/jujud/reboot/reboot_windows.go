package reboot

import (
	"fmt"
	"time"

	"github.com/juju/juju/apiserver/params"
)

// executeAction will do a reboot or shutdown after given number of seconds
// this function executes the operating system's reboot binary with apropriate
// parameters to schedule the reboot
// If action is params.ShouldDoNothing, it will return immediately.
// NOTE: On Windows the shutdown command is async
func executeAction(action params.RebootAction, errChan chan error) {
	if action == params.ShouldDoNothing {
		errChan <- nil
	}
	args := []string{"shutdown"}
	switch action {
	case params.ShouldReboot:
		args = append(args, "-r")
	case params.ShouldShutdown:
		args = append(args, "-s")
	}
	args = append(args, "-t 0")

	errChan <- runCommand(args)
}

func scheduleReboot(action params.RebootAction, seconds int) <-chan error {
	errChan := make(chan struct{})
	time.Sleep(seconds * time.Second)
	go executeAction(action, errChan)
	return errChan
}
