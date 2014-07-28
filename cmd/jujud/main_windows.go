package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"bitbucket.org/kardianos/service"
)

func runService() {
	var name = "juju"
	var displayName = "juju service"
	var desc = "juju service"

	var s, err = service.NewService(name, displayName, desc)
	if err != nil {
		fmt.Errorf("%s", err)
	}

	err = s.Run(func() error {
		// start
		go Main(os.Args)
		return nil
	}, func() error {
		// stop
		os.Exit(0)
		return nil
	})

	if err != nil {
		s.Error(err.Error())
	}
}

func runConsole() {
	Main(os.Args)
}

func main() {
	var mode uint32
	err := syscall.GetConsoleMode(syscall.Stdin, &mode)

	isConsole := err == nil

	if isConsole == true {
		runConsole()
	} else {
		runService()
	}
}
