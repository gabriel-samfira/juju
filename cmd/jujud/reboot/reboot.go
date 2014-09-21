package reboot

import (
	"os/exec"
	"time"

	"github.com/juju/loggo"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/container"
	"github.com/juju/juju/container/factory"
	"github.com/juju/juju/instance"
	"github.com/juju/juju/utils/rebootstate"
)

var logger = loggo.GetLogger("juju.cmd.jujud.reboot")

func runCommand(args []string) error {
	_, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return err
	}
	return nil
}

type Reboot struct {
	acfg agent.Config
}

func NewRebootWaiter(acfg agent.Config) (*Reboot, error) {
	return &Reboot{
		acfg: acfg,
	}, nil
}

func (r *Reboot) ExecuteReboot(action params.RebootAction) error {
	err := r.waitForContainersOrTimeout()
	if err != nil {
		return err
	}
	err = rebootstate.New()
	if err != nil {
		return err
	}
	// Execute reboot or shutdown.
	return executeAction(action)
}

func (r *Reboot) runningContainers() ([]instance.Instance, error) {
	runningInstances := []instance.Instance{}

	for _, val := range instance.ContainerTypes {
		managerConfig := container.ManagerConfig{container.ConfigName: "juju"}
		if namespace := r.acfg.Value(agent.Namespace); namespace != "" {
			managerConfig[container.ConfigName] = namespace
		}
		cfg := container.ManagerConfig(managerConfig)
		manager, err := factory.NewContainerManager(val, cfg)
		if err != nil {
			logger.Warningf("Failed to get manager for container type %v: %v", val, err)
			continue
		}
		if !manager.IsInitialized() {
			continue
		}
		instances, err := manager.ListContainers()
		if err != nil {
			logger.Warningf("Failed to list containers: %v", err)
		}
		runningInstances = append(runningInstances, instances...)
	}
	return runningInstances, nil
}

func (r *Reboot) waitForContainersOrTimeout() error {
	c := make(chan error, 1)
	quit := make(chan bool, 1)
	go func() {
		for {
			select {
			case <-quit:
				c <- nil
				return
			default:
				containers, err := r.runningContainers()
				if err != nil {
					c <- err
					return
				}
				if len(containers) == 0 {
					c <- nil
					return
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()

	select {
	case <-time.After(10 * time.Minute):

		// Containers are still up after timeout. C'est la vie
		logger.Infof("Timeout reached waiting for containers to shutdown")
		quit <- true
	case err := <-c:
		return err

	}
	return nil
}
