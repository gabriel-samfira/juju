package main

import (
	"os"

	"bitbucket.org/kardianos/service"
)

var exit = make(chan struct{})

func main() {
	var name = "Jujud"
	var displayName = "Juju agent"
	var desc = "Juju agent"

	s, err := service.NewService(name, displayName, desc)
	if err != nil {
		fmt.Errorf("%s unable to start: %s", displayName, err)
		return
	}
	err = s.Run(func() error {
		// start
		go startService()
		return nil
	}, func() error {
		// stop
		stopService()
		return nil
	})
	if err != nil {
		s.Error(err.Error())
	}
}

func startService() {
	select {
	case <-exit:
		return
	default:
		Main(os.Args)
	}
}

func stopService() {
	exit <- struct{}{}
}
