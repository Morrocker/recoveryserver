package director

import (
	"fmt"
	"sync"
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/logger"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/recovery"
	"github.com/morrocker/recoveryserver/utils"
)

// Director orders and decides which recoveries should be executed next
type Director struct {
	Run       bool
	AutoQueue bool

	Clouds     map[string]*Cloud
	Recoveries map[string]*recovery.Recovery
	Lock       sync.Mutex
}

// StartDirector starts the Director service and all subservices
func (d *Director) StartDirector(c config.Config) error {
	errPath := "director.StartDirector()"
	logger.TaskV("Starting director services")
	//LOAD CONFIG HERE
	d.Clouds = make(map[string]*Cloud)
	d.Recoveries = make(map[string]*recovery.Recovery)
	d.Run = c.AutoRunRecoveries
	d.AutoQueue = c.AutoQueueRecoveries
	d.SetClouds()

	if err := d.ReadRecoveryJSON(); err != nil {
		err = errors.New(errPath, err)
		logger.Error("%v", err)
		return err
	}
	if d.AutoQueue {
		for hash := range d.Recoveries {
			d.Recoveries[hash].Status = recovery.Queue
		}
	}
	go d.StartWorkers()
	go d.PickRecovery()
	return nil
}

// StartWorkers continually tries to start workers for each recovery added
func (d *Director) StartWorkers() {
	logger.TaskV("Starting recovery workers creator")
	for {
		d.Lock.Lock()
		for key, recover := range d.Recoveries {
			if recover.Status == recovery.Queue {
				go d.Recoveries[key].Run()
			}
		}
		d.Lock.Unlock()
		time.Sleep(10 * time.Second)
	}
}

// AddRecovery adds the given recovery data to create a new entry on the Recoveries map
func (d *Director) AddRecovery(r recovery.Data) (string, error) {
	logger.TaskV("Adding new recovery")
	errPath := ("director.AddRecovery()")

	// Sanitizing parameters
	if r.User == "" {
		return "", errors.New(errPath, "User parameter empty or missing")
	}
	if r.Machine == "" {
		return "", errors.New(errPath, "Machine parameter empty or missing")
	}
	if r.Metafile == "" {
		return "", errors.New(errPath, "Metafil parameter empty or missing")
	}
	if r.Organization == 0 {
		return "", errors.New(errPath, "Organization parameter empty or missing")
	}
	if r.Repository == "" {
		return "", errors.New(errPath, "Repository parameter empty or missing")
	}
	if r.Disk == "" {
		return "", errors.New(errPath, "Disk parameter empty or missing")
	}

	hash := utils.RandString(8)
	d.Recoveries[hash] = &recovery.Recovery{Info: r, ID: hash, Priority: recovery.Medium}
	if d.AutoQueue {
		d.Recoveries[hash].Status = recovery.Queue
	}
	if err := d.WriteRecoveryJSON(); err != nil {
		return "", errors.Extend(errPath, err)
	}
	return hash, nil
}

// Stop sets Run to false
func (d *Director) Stop() {
	logger.TaskV("Setting Director.Run to false")
	d.Run = false
}

// Start sets Run to true
func (d *Director) Start() {
	logger.TaskV("Setting Director.Run to true")
	d.Run = true
}

// PickRecovery decides what recovery must be executed next. It prefers higher priority over lower. Will skip if a recovery is running. Has a low latency by design.
func (d *Director) PickRecovery() {
	logger.TaskV("Starting recovery picker")
	for {
	Start:
		if !d.Run {
			logger.Info("Director set Run to false. Sleeping")
			time.Sleep(30 * time.Second)
			continue
		}
		logger.Info("Trying to decide new recovery to run")
		var nextRecovery *recovery.Recovery = &recovery.Recovery{Priority: -1}
		var nextRecoveryHash string
		for hash, Recovery := range d.Recoveries {
			if Recovery.Status == recovery.Start {
				logger.Info("A recovery is already running. Sleeping for a while")
				time.Sleep(30 * time.Second)
				goto Start
			}

			if Recovery.Status == recovery.Stop && nextRecovery.Priority < Recovery.Priority && Recovery.Destination != "" {
				logger.Info("Found possible recovery %s", nextRecoveryHash)
				nextRecovery = Recovery
				nextRecoveryHash = hash
			}
		}
		if nextRecoveryHash == "" {
			logger.Info("No recovery found. Starting again.")
			time.Sleep(30 * time.Second)
			continue
		}
		d.Recoveries[nextRecoveryHash].Status = recovery.Start
		// 30 seconds is for test purposes. Change later
		time.Sleep(30 * time.Second)
		continue
	}
}

// ChangePriority changes a given recovery priority to a specific value
func (d *Director) ChangePriority(id string, value int) error {
	logger.TaskV("Changing recovery %s Priority to %d", id, value)
	errPath := "director.ChangePriority"
	d.Lock.Lock()
	defer d.Lock.Unlock()
	if value > recovery.VeryHigh || value < recovery.VeryLow {
		return errors.New(errPath, "Priority value outside allowed parameters")
	}
	d.Recoveries[id].Priority = value
	return nil
}

// PausePicker stops the PickRecovery so that it does not continue launching recoveries
func (d *Director) PausePicker() {
	logger.TaskV("Pausing recovery picker")
	d.Lock.Lock()
	defer d.Lock.Unlock()
	d.Run = false
}

// RunPicker starts or resumes the PickRecovery function
func (d *Director) RunPicker() {
	logger.TaskV("Resuming recovery picker")
	d.Lock.Lock()
	defer d.Lock.Unlock()
	d.Run = true
}

// PauseRecovery sets a given recover status to Pause
func (d *Director) PauseRecovery(id string) error {
	logger.TaskD("Pausing recovery %s", id)
	errPath := "director.PauseRecovery()"
	Recovery, ok := d.Recoveries[id]
	if !ok {
		msg := fmt.Sprintf("Recovery %s not found", id)
		return errors.New(errPath, msg)
	}
	Recovery.Pause()
	return nil
}

// StartRecovery sets a given recovery status to Start // TODO: See how resuming works
func (d *Director) StartRecovery(id string) error {
	logger.TaskD("Starting/Resuming recovery %s", id)
	errPath := "director.StartRecovery()"
	Recovery, ok := d.Recoveries[id]
	if !ok {
		msg := fmt.Sprintf("Recovery %s not found", id)
		return errors.New(errPath, msg)
	}
	Recovery.Start()
	return nil
}

// QueueRecovery sets a recovery status to Queue. Needed when autoqueue is off.
func (d *Director) QueueRecovery(id string) error {
	logger.TaskD("Queueing recovery %s", id)
	errPath := "director.QueueRecovery()"
	Recovery, ok := d.Recoveries[id]
	if !ok {
		msg := fmt.Sprintf("Recovery %s not found", id)
		return errors.New(errPath, msg)
	}
	Recovery.Queue()
	return nil
}

// SetClouds asdfsa asdf a
func (d *Director) SetClouds() {
	for _, cloud := range config.Data.Clouds {
		d.Clouds[cloud.FilesAddress] = NewCloud(cloud)
	}
}
