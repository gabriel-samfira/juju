package reboot

import (
	"os/exec"
	"regexp"
	"runtime"
	"time"

	"github.com/juju/loggo"
	"github.com/juju/names"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/api"
	"github.com/juju/juju/api/reboot"
	"github.com/juju/juju/container"
	"github.com/juju/juju/container/factory"
	"github.com/juju/juju/container/kvm"
	"github.com/juju/juju/instance"
	"github.com/juju/juju/service"
	"github.com/juju/juju/service/common"
	"github.com/juju/juju/service/upstart"
	"github.com/juju/juju/state"
)

var logger = loggo.GetLogger("juju.cmd.jujud.reboot")
var unitRe = regexp.MustCompile("^(jujud-.*unit-([a-z0-9-]+)-([0-9]+))$")

func runCommand(args []string) error {
	_, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return err
	}
	return nil
}

func isUnit(val string) bool {
	if groups := unitRe.FindStringSubmatch(val); len(groups) > 0 {
		return true
	}
	return false
}

type Reboot struct {
	acfg     agent.Config
	st       *reboot.State
	tag      names.MachineTag
	apistate *api.State
}

func NewRebootWaiter(apistate *api.State, acfg agent.Config, tag names.MachineTag) *Reboot {
	rebootState := apistate.Reboot()
	return &Reboot{
		acfg:     acfg,
		st:       rebootState,
		tag:      tag,
		apistate: apistate,
	}
}

func (r *Reboot) ExecuteReboot() error {
	st := r.apistate.Reboot()
	action, err := st.GetRebootAction()
	if err != nil {
		logger.Errorf("Reboot: Error getting reboot action: %v", err)
		return err
	}

	err = r.waitForContainersOrTimeout()
	if err != nil {
		return err
	}

	// TODO (gsamfira): Maybe we should clear the flag before issuing the reboot?
	//
	// Execute reboot or shutdown. We do this after 10 seconds
	// to allow the machine agent to clear its reboot flag
	delayBeforeAction := 10
	c := scheduleReboot(action, delayBeforeAction)

	// Clear the reboot flag.
	err = r.st.ClearReboot()
	if err != nil {
		return err
	}
	// Wait for the reboot to return
	err = <-c
	if err != nil {
		return err
	}
	return nil
}

func (r *Reboot) StopAllUnits() error {
	services, err := service.ListServices(upstart.InitDir)
	if err != nil {
		return err
	}
	logger.Infof("Trying to stop units")
	for _, val := range services {
		if !isUnit(val) {
			continue
		}
		cfg := common.Conf{InitDir: upstart.InitDir}
		svc := service.NewService(val, cfg)
		logger.Infof("Stopping unit: %v", val)
		err = svc.Stop()
		if err != nil {
			logger.Warningf("Failed to stop service %q: %q", val, err)
		}
	}
	return nil
}

func (r *Reboot) supportedContainers() ([]instance.ContainerType, error) {
	var supportedContainers []instance.ContainerType

	entity, err := r.apistate.Agent().Entity(r.tag)
	if err != nil {
		return nil, err
	}
	if entity.ContainerType() != instance.LXC && runtime.GOOS != "windows" {
		supportedContainers = append(supportedContainers, instance.LXC)
	}
	supportsKvm, err := kvm.IsKVMSupported()
	if err == nil && supportsKvm {
		supportedContainers = append(supportedContainers, instance.KVM)
	}
	return supportedContainers, nil
}

func (r *Reboot) runningContainers() ([]instance.Instance, error) {
	runningInstances := []instance.Instance{}
	supportedContainers, err := r.supportedContainers()
	if err != nil {
		return runningInstances, err
	}
	if len(supportedContainers) == 0 {
		return runningInstances, nil
	}

	for _, val := range supportedContainers {
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
		instances, err := manager.ListContainers()
		if err != nil {
			logger.Warningf("Failed to list containers: %v", err)
		}
		runningInstances = append(runningInstances, instances...)
	}
	return runningInstances, nil
}

func (r *Reboot) waitForContainersOrTimeout() error {
	timeout := 0 * time.Minute
	nestingLvl := state.NestingLevel(r.tag.Id())
	switch nestingLvl {
	case 0:
		// wait a maximum of 10 minutes if we are bare metal
		// We need to wait for our containers to shutdown, which
		// in turn may have containers of its own to shutdown
		timeout = 10 * time.Minute
	case 1:
		// We are first level containers. Allow for out containers
		// to shutdown before returning
		timeout = 5 * time.Minute
	case 2:
		// Maximum nesting level. No containers can exist here (yet)
		return nil
	}

	c := make(chan error, 1)
	go func() {
		for {
			containers, err := r.runningContainers()
			if err != nil {
				c <- err
				return
			}
			logger.Infof("The following containers are still up: %v", containers)
			if len(containers) == 0 {
				c <- nil
				return
			}
			time.Sleep(5 * time.Second)
		}
	}()

	select {
	case <-time.After(timeout):
		// Containers are still up after timeout. C'est la vie
		logger.Infof("Timeout reached waiting for containers to shutdown")
		return nil
	case err := <-c:
		return err

	}
	return nil
}
