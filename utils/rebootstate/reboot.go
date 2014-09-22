package rebootstate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/juju/errors"
	"github.com/juju/utils/uptime"

	"github.com/juju/juju/agent"
)

var RebootStateFile = filepath.Join(agent.DefaultDataDir, "reboot-state.txt")

// for testing
var UptimeFunc = uptime.Uptime

func New() error {
	if _, err := os.Stat(RebootStateFile); err == nil {
		return errors.Errorf("state file %s already exists", RebootStateFile)
	}
	uptime, err := UptimeFunc()
	if err != nil {
		return err
	}

	contents := []byte(strconv.FormatInt(uptime, 10))
	err = ioutil.WriteFile(RebootStateFile, contents, 400)
	if err != nil {
		return err
	}
	return nil
}

func Remove() error {
	if _, err := os.Stat(RebootStateFile); err == nil {
		err = os.Remove(RebootStateFile)
		return err
	}
	return nil
}

func Read() (int64, error) {
	contents, err := ioutil.ReadFile(RebootStateFile)
	if err != nil {
		return 0, err
	}
	uptime, err := strconv.ParseInt(string(contents), 10, 64)
	if err != nil {
		return 0, err
	}
	return uptime, nil
}
