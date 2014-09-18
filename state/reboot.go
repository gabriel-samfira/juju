package state

import (
	"fmt"

	"github.com/juju/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2/txn"

	"github.com/juju/juju/apiserver/params"
)

// rebootDoc will hold the reboot flag for a machine.
type rebootDoc struct {
	Id string `bson:"_id"`
}

func addRebootDocOps(machineId string) []txn.Op {
	ops := []txn.Op{{
		C:      machinesC,
		Id:     machineId,
		Assert: notDeadDoc,
	}, {
		C:      rebootC,
		Id:     machineId,
		Insert: rebootDoc{Id: machineId},
	}}
	return ops
}

func removeRebootDocOps(machineId string) []txn.Op {
	ops := []txn.Op{{
		C:      rebootC,
		Id:     machineId,
		Remove: true,
	}}
	return ops
}

// SetRebootFlag sets the reboot flag of a machine to a boolean value. It will also
// do a lazy create of a reboot document if needed; i.e. If a document
// does not exist yet for this machine, it will create it.
func (m *Machine) SetRebootFlag(flag bool) error {
	reboot, closer := m.st.getCollection(rebootC)
	defer closer()
	buildTxn := func(attempt int) ([]txn.Op, error) {
		return addRebootDocOps(m.Id()), nil
	}

	var rDoc rebootDoc
	err := reboot.FindId(m.Id()).One(&rDoc)
	if flag == false {
		if err == mgo.ErrNotFound {
			return nil
		} else {
			buildTxn = func(attempt int) ([]txn.Op, error) {
				return removeRebootDocOps(m.Id()), nil
			}
		}
	}

	// Run the transaction using the state transaction runner.
	err = m.st.run(buildTxn)
	if err != nil {
		return errors.Annotate(err, "Failed to run mongo transaction")
	}
	return nil
}

// GetRebootFlag returns the reboot flag for this machine.
func (m *Machine) GetRebootFlag() (bool, error) {
	rebootCol, closer := m.st.getCollection(rebootC)
	defer closer()

	var rDoc rebootDoc

	err := rebootCol.FindId(m.Id()).One(&rDoc)
	if err == mgo.ErrNotFound {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to get reboot doc %v: %v", m.Id(), err)
	}
	return true, nil
}

func (m *Machine) machinesToCareAboutRebootsFor() []string {
	var possibleIds []string
	for currentId := m.Id(); currentId != ""; {
		possibleIds = append(possibleIds, currentId)
		currentId = ParentId(currentId)
	}
	return possibleIds
}

// ShouldRebootOrShutdown check if the current node should reboot or shutdown
// If we are a container, and our parent needs to reboot, this should return:
// ShouldShutdown
func (m *Machine) ShouldRebootOrShutdown() (params.RebootAction, error) {
	rebootCol, closer := m.st.getCollection(rebootC)
	defer closer()

	machines := m.machinesToCareAboutRebootsFor()

	docs := []rebootDoc{}
	sel := bson.D{{"_id", bson.D{{"$in", machines}}}}
	if err := rebootCol.Find(sel).All(&docs); err != nil {
		return params.ShouldDoNothing, errors.Trace(err)
	}

	iNeedReboot := false
	for _, val := range docs {
		if val.Id != m.doc.Id {
			return params.ShouldShutdown, nil
		}
		iNeedReboot = true
	}
	if iNeedReboot {
		return params.ShouldReboot, nil
	}
	return params.ShouldDoNothing, nil
}
