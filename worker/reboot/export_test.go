// Copyright 2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package reboot

import "github.com/juju/juju/api/reboot"

func (r *Reboot) CheckForRebootState() error {
	return r.checkForRebootState()
}

func NewRebootStruct(st *reboot.State) Reboot {
	return Reboot{st: st}
}
