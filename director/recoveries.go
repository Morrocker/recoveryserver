package director

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/recovery"
)

// PickRecovery decides what recovery must be executed next. It prefers higher priority over lower.
// Will skip if a recovery is running. Has a low latency by design.
func (d *Director) recoveryPicker() {
	log.TaskV("Starting Recovery Picker")
	l := d.broadcaster.Listen()
Loop:
	for {
		<-l.C
		if !d.run {
			log.InfoD("Director set Run to false")
			continue
		}
		log.InfoD("Trying to decide new recovery to run")
		var nextRecovery *recovery.Recovery
		var nextPriority recovery.Priority = -1
		for _, r := range d.Recoveries {
			if r.Status == recovery.Running {
				log.InfoD("A recovery is already running")
				continue Loop
			}
			if r.Status == recovery.Queued && nextPriority < r.Priority && r.GetOutput() != "" {
				log.InfoD("Found possible recovery ID:%d", r.Data.ID)
				nextRecovery = r
				nextPriority = r.Priority
			}
		}
		if nextRecovery != nil {
			go nextRecovery.Run()
		}
	}
}

// ChangePriority changes a given recovery priority to a specific value
func (d *Director) ChangePriority(id int, value int) error {
	log.TaskV("Changing recovery %s Priority to %d", id, value)
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend("director.ChangePriority", err)
	}
	return r.SetPriority(value)
}

// AddRecovery adds the given recovery data to create a new entry on the Recoveries map
func (d *Director) AddRecovery(data *recovery.Data) error {
	log.TaskV("Adding new recovery")
	op := ("director.AddRecovery()")

	if err := checkEmptyData(data); err != nil {
		return errors.Extend(op, err)
	}
	if _, ok := d.Recoveries[data.ID]; ok {
		return errors.New(op, fmt.Sprintf("Recovery #%d already exists. Remove first", data.ID))
	}

	login, err := getLogin(config.Data.LoginAddr, data.User)
	if err != nil {
		return errors.Extend(op, err)
	}

	var newCloud config.Cloud
	var found bool
	for _, cloud := range config.Data.Clouds {
		if cloud.FilesAddress == login {
			newCloud = cloud
			found = true
			break
		}
	}
	if !found {
		return errors.New(op, fmt.Sprintf("Could not find cloud to match login %s", login))
	}

	d.Recoveries[data.ID] = recovery.New(data.ID, data, d.broadcaster, newCloud)
	return nil
}

// PauseRecovery sets a given recover status to Pause
func (d *Director) PauseRecovery(id int) error {
	log.TaskD("Pausing recovery %s", id)
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend("director.PauseRecovery()", err)
	}
	return r.Pause()
}

// StartRecovery sets a given recovery status to Start // TODO: See how resuming works
func (d *Director) StartRecovery(id int) error {
	log.TaskD("Starting/Resuming recovery %s", id)
	op := "director.StartRecovery()"
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(op, err)
	}
	return r.Start()
}

// QueueRecovery sets a recovery status to Queue. Needed when autoqueue is off.
func (d *Director) QueueRecovery(id int) error {
	op := "director.QueueRecovery()"

	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(op, err)
	}
	return r.Queue()
}

// CancelRecovery sets a given recovery status to Start // TODO: See how resuming works
func (d *Director) CancelRecovery(id int) error {
	log.TaskD("Canceling recovery %d", id)
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend("director.StartRecovery()", err)
	}
	return r.Cancel()
}

// SetDestination
func (d *Director) SetDestination(id int, dst string) error {
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend("director.SetDestination()", err)
	}
	r.SetOutput(dst)
	return nil
}

// PauseRecovery sets a given recover status to Pause
func (d *Director) PreCalculate(id int) error {
	log.TaskV("Precalculating recovery #%d size", id)
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend("director.PauseRecovery()", err)
	}
	r.PreCalculate()
	return nil
}

// accesory functions

func (d *Director) findRecovery(id int) (*recovery.Recovery, error) {
	r, ok := d.Recoveries[id]
	if !ok {
		return nil, errors.New("recovery.findRecovery()", fmt.Sprintf("Recovery %d not found", id))
	}
	return r, nil
}

func checkEmptyData(d *recovery.Data) error {
	op := "director.checkData()"
	if d.ID == 0 {
		return errors.New(op, "ID parameter empty")
	}
	switch "" {
	case d.User, d.Metafile, d.Repository, d.Org, d.Disk, d.Machine:
		return errors.New(op, "User, Metafile, Repository, Org, Disk or Machine parameter empty")
	}
	return nil
}

// GetLogin finds the server that the users belongs to
func getLogin(addr, login string) (string, error) {
	op := "recovery.GetLogin()"

	uLogin := url.QueryEscape(login)
	query := fmt.Sprintf("%s?login=%s", addr, uLogin)
	req, err := http.NewRequest("GET", query, nil)
	if err != nil {
		return "", errors.New(op, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.New(op, err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorln(*resp)
		return "", errors.New(op, "Response status not OK")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New(op, err)
	}
	resp.Body.Close()

	s := string(body)
	out := strings.Trim(s, "\"")
	return out, nil
}
