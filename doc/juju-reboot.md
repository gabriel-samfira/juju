Rebooting a machine
====================

In some cases, the need may arise to have the machine execute a reboot at some point. Be it because the charm itself needs to install a new kernel, or because we have to upgrade the entire system before we continue. Windows workloads for example, frequently require reboots before an install is finished, and sometimes, require multiple reboots in the same install procedure.

To be able to do this without erring out hooks that require a reboot, there needed to be a way to do a reboot, and possibly re-queue the hook to be run after the reboot has happened.


juju-reboot
============

The juju-reboot enables the charm author to:

* queue a reboot to be run after the current hook finishes
* request an immediate reboot

There are two use cases to talk about here. The first one is the easiest one, in which for example, the charm author may want to run setup of needed services or components, and at the end, request a reboot for those changes to take effect. After the reboot happens, the rest of the hooks can fire and do everything else that is needed.

In this scenario, the juju-reboot tool can be called in any part of the hook, and the reboot will be queued, provided the hook does not err out. Reboots will never be executed if the hook exits with an error. 

The second scenario is a bit more complicated, and basically takes into account multi-step setups. These types of setups are more common in windows workloads, where installing a service may require multiple reboots, and thus, you may end up with an install hook that needs to be run several times. There are a few such examples when dealing with windows workloads, but this may be a need on any OS at some point.

In this scenario, the charm author would have to call:

```shell
juju-reboot --now
```

This is a blocking call. If we use the --now flag, juju-reboot will hang until the unit agent kills the hook and re-queues it for next run. This will allow you to create multi-step install hooks. Please note, that this tool can be used in any hook. There is no limitation in this respect.

The charm author should also take great care to wrap any call to juju-reboot in a check to see if its actually necessary, otherwise we risk entering a reboot loop where the charm continuously requests reboots, thinking it still needs to do so. So the preferred work-flow (in pseudo-code) would be:

```shell
if [ $FEATURE_IS_INSTALLED  == "false" ]
then
	install_feature
	juju-reboot --now
fi
```

Where can I use juju-reboot?
============================

Currently juju-reboot can only be used in hooks and via the juju-run command. Support for actions and debug-hooks will be added later. Actions have a different enough behavior to warrant greater consideration when actually executing a reboot.  