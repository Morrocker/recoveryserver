package director

import (
	"fmt"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/recovery"
)

// PickRecovery decides what recovery must be executed next. It prefers higher priority over lower.
// Will skip if a recovery is running. Has a low latency by design.
func (d *Director) recoveryPicker() {
	log.TaskV("Starting Recovery Picker")
	for {
		<-d.statusMonitor
		log.Alertln("Status change! Running picker")
		if !d.Run {
			log.InfoD("Director set Run to false")
			continue
		}
		log.InfoD("Trying to decide new recovery to run")
		var nextRecovery *recovery.Recovery
		var nextPriority int = -1
		for _, r := range d.Recoveries {
			if r.Status == recovery.Running {
				log.InfoD("A recovery is already running. Sleeping for a while")
				break
			}
			if r.Status == recovery.Queued && nextPriority < r.Priority && r.GetOutput() != "" {
				log.InfoD("Found possible recovery ID:%d", r.Data.ID)
				nextRecovery = r
				nextPriority = r.Priority
			}
		}
		if nextRecovery != nil {
			go nextRecovery.Start()
		}
	}
}

// ChangePriority changes a given recovery priority to a specific value
func (d *Director) ChangePriority(id int, value int) error {
	log.TaskV("Changing recovery %s Priority to %d", id, value)
	d.Lock.Lock()
	defer d.Lock.Unlock()
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend("director.ChangePriority", err)
	}
	r.SetPriority(value)
	return nil
}

// AddRecovery adds the given recovery data to create a new entry on the Recoveries map
func (d *Director) AddRecovery(data *recovery.Data) error {
	log.TaskV("Adding new recovery")
	op := ("director.AddRecovery()")

	// Sanitizing parameters
	if data.ID == 0 {
		return errors.New(op, "ID parameter empty or missing")
	}
	if data.User == "" {
		return errors.New(op, "User parameter empty or missing")
	}
	if data.Machine == "" {
		return errors.New(op, "Machine parameter empty or missing")
	}
	if data.Metafile == "" {
		return errors.New(op, "Metafile parameter empty or missing")
	}
	if data.Org == "" {
		return errors.New(op, "Organization parameter empty or missing")
	}
	if data.Repository == "" {
		return errors.New(op, "Repository parameter empty or missing")
	}
	if data.Disk == "" {
		return errors.New(op, "Disk parameter empty or missing")
	}

	d.Recoveries[data.ID] = recovery.New(data.ID, data, d.statusMonitor)
	if err := d.QueueRecovery(data.ID); err != nil {
		return errors.Extend(op, err)
	}
	return nil
}

// PauseRecovery sets a given recover status to Pause
func (d *Director) PauseRecovery(id int) error {
	log.TaskD("Pausing recovery %s", id)
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend("director.PauseRecovery()", err)
	}
	r.Pause()
	return nil
}

// StartRecovery sets a given recovery status to Start // TODO: See how resuming works
func (d *Director) StartRecovery(id int) error {
	log.TaskD("Starting/Resuming recovery %s", id)
	op := "director.StartRecovery()"
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(op, err)
	}
	if r.Status != recovery.Paused {
		return errors.New(op, fmt.Sprintf("Recovery %d is not Paused. Returning", id))
	}
	r.Start()
	return nil
}

// QueueRecovery sets a recovery status to Queue. Needed when autoqueue is off.
func (d *Director) QueueRecovery(id int) error {
	log.TaskV("Queueing recovery %s", id)
	op := "director.QueueRecovery()"

	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(op, err)
	}
	login, err := recovery.GetLogin(config.Data.SrvLogDir, r.Data.User)
	if err != nil {
		return errors.Extend(op, err)
	}
	for _, cloud := range config.Data.Clouds {
		if cloud.FilesAddress == login {
			r.SetCloud(cloud)
			goto Found
		}
	}
	return errors.New(op, "Failed to find match recovery with any existing Cloud")
Found:
	r.Queue()
	return nil
}

// CancelRecovery sets a given recovery status to Start // TODO: See how resuming works
func (d *Director) CancelRecovery(id int) error {
	log.TaskD("Canceling recovery %d", id)
	op := "director.StartRecovery()"
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(op, err)
	}
	if r.Status == recovery.Canceled {
		return errors.New(op, "Recovery is already Canceled")
	} else if r.Status == recovery.Done {
		return errors.New(op, "Recovery is already Done")
	} else if r.Status == recovery.Entry {
		return errors.New(op, "Recovery is at the Entry point")
	}
	r.Cancel()
	return nil
}

// SetDestination
func (d *Director) SetDestination(id int, dst string) (err error) {
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend("director.SetDestination()", err)
	}
	r.SetOutput(dst)
	return
}

// accesory functions

func (d *Director) findRecovery(id int) (*recovery.Recovery, error) {
	r, ok := d.Recoveries[id]
	if !ok {
		return nil, errors.New("recovery.findRecovery()", fmt.Sprintf("Recovery %d not found", id))
	}
	return r, nil
}
