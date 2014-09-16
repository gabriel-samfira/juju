package reboot

import (
	"github.com/juju/loggo"
	"launchpad.net/tomb"

	"github.com/juju/juju/api/reboot"
	"github.com/juju/juju/api/watcher"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/worker"
)

var logger = loggo.GetLogger("juju.worker.reboot")

var _ worker.NotifyWatchHandler = (*Reboot)(nil)

// The reboot worker listens for changes to the reboot flag and
// exists with worker.ErrRebootMachine if the machine should shutdown
// or reboot. This will be picked up by the machine agent as a fatal error
// and will do the right thing (reboot or shutdown)
type Reboot struct {
	tomb tomb.Tomb
	st   *reboot.State
}

func NewReboot(
	st *reboot.State,
) worker.Worker {
	r := &Reboot{
		st: st,
	}
	return worker.NewNotifyWorker(r)
}

func (r *Reboot) SetUp() (watcher.NotifyWatcher, error) {
	logger.Debugf("Reboot worker setup")
	watcher, err := r.st.WatchForRebootEvent()
	if err != nil {
		return nil, err
	}
	return watcher, nil
}

func (r *Reboot) Handle() error {
	rAction, err := r.st.GetRebootAction()
	if err != nil {
		return err
	}
	logger.Debugf("Reboot worker got action: %v", rAction)
	switch rAction {
	case params.ShouldReboot:
		return worker.ErrRebootMachine
	case params.ShouldShutdown:
		return worker.ErrRebootMachine
	}
	return nil
}

func (r *Reboot) TearDown() error {
	// nothing to teardown.
	return nil
}
