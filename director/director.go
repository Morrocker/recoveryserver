package director

import (
	"fmt"
	"sync"
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/logger"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/recovery"
	"github.com/morrocker/recoveryserver/remotes"
	"github.com/morrocker/recoveryserver/utils"
)

// Director orders and decides which recoveries should be executed next
type Director struct {
	Run       bool
	AutoQueue bool

	SuperTracker *tracker.SuperTracker
	Clouds       map[string]*remotes.Cloud
	Recoveries   map[string]*recovery.Recovery
	RunLock      sync.Mutex
	Lock         sync.Mutex
}

// StartDirector starts the Director service and all subservices
func (d *Director) StartDirector(c config.Config) error {
	errPath := "director.StartDirector()"
	logger.TaskV("Starting director services")
	//LOAD CONFIG HERE
	d.Clouds = make(map[string]*remotes.Cloud)
	d.Recoveries = make(map[string]*recovery.Recovery)
	d.Run = c.AutoRunRecoveries
	d.AutoQueue = c.AutoQueueRecoveries
	d.InitClouds()

	if err := d.ReadRecoveryJSON(); err != nil {
		err = errors.New(errPath, err)
		logger.Error("%v", err)
	}
	if d.AutoQueue {
		for id := range d.Recoveries {
			if err := d.QueueRecovery(id); err != nil {
				logger.Alert("%s", errors.New(errPath, err))
			}
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
				go d.Recoveries[key].Run(&d.RunLock)
			}
		}
		d.Lock.Unlock()
		time.Sleep(10 * time.Second)
	}
}

// AddRecovery adds the given recovery data to create a new entry on the Recoveries map
func (d *Director) AddRecovery(data *recovery.Data) (string, error) {
	logger.TaskV("Adding new recovery")
	errPath := ("director.AddRecovery()")

	// Sanitizing parameters
	if data.User == "" {
		return "", errors.New(errPath, "User parameter empty or missing")
	}
	if data.Machine == "" {
		return "", errors.New(errPath, "Machine parameter empty or missing")
	}
	if data.Metafile == "" {
		return "", errors.New(errPath, "Metafil parameter empty or missing")
	}
	if data.RootGroup == 0 {
		return "", errors.New(errPath, "Organization parameter empty or missing")
	}
	if data.Repository == "" {
		return "", errors.New(errPath, "Repository parameter empty or missing")
	}
	if data.Disk == "" {
		return "", errors.New(errPath, "Disk parameter empty or missing")
	}

	id := utils.RandString(8)
	d.Recoveries[id] = recovery.New(id, data)
	if d.AutoQueue {
		if err := d.QueueRecovery(id); err != nil {
			logger.Alert("%s", errors.New(errPath, err))
		}
	}
	if err := d.WriteRecoveryJSON(); err != nil {
		return "", errors.Extend(errPath, err)
	}
	return id, nil
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
			logger.InfoV("Director set Run to false. Sleeping")
			time.Sleep(30 * time.Second)
			continue
		}
		logger.InfoV("Trying to decide new recovery to run")
		var nextRecovery *recovery.Recovery = &recovery.Recovery{Priority: -1}
		var nextRecoveryHash string
		for hash, Recovery := range d.Recoveries {
			if Recovery.Status == recovery.Start {
				logger.InfoV("A recovery is already running. Sleeping for a while")
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
			logger.InfoV("No recovery found. Starting again.")
			time.Sleep(30 * time.Second)
			continue
		}
		d.Recoveries[nextRecoveryHash].Start()
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
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(errPath, err)
	}
	r.SetPriority(value)
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
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(errPath, err)
	}
	r.Pause()
	return nil
}

// StartRecovery sets a given recovery status to Start // TODO: See how resuming works
func (d *Director) StartRecovery(id string) error {
	logger.TaskD("Starting/Resuming recovery %s", id)
	errPath := "director.StartRecovery()"
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(errPath, err)
	}
	r.Start()
	return nil
}

// QueueRecovery sets a recovery status to Queue. Needed when autoqueue is off.
func (d *Director) QueueRecovery(id string) error {
	logger.TaskV("Queueing recovery %s", id)
	errPath := "director.QueueRecovery()"
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(errPath, err)
	}
	if err := r.GetLogin(); err != nil {
		return errors.Extend(errPath, err)
	}
	found := false
	for _, cloud := range d.Clouds {
		if cloud.FilesAddress == r.Data.Server {
			r.SetCloud(cloud)
			found = true
			break
		}
	}
	if !found {
		return errors.New(errPath, "Failed to find match recovery with any existing Cloud")
	}
	r.Queue()
	return nil
}

// InitClouds
func (d *Director) InitClouds() {
	for _, cloud := range config.Data.Clouds {
		d.Clouds[cloud.FilesAddress] = remotes.NewCloud(cloud)
	}
}

// SetDestination
func (d *Director) SetDestination(id, dst string) error {
	errPath := "director.SetDestination()"
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(errPath, err)
	}
	r.SetDestination(dst)
	return nil
}

func (d *Director) findRecovery(id string) (*recovery.Recovery, error) {
	errPath := "recovery.findRecovery()"
	Recovery, ok := d.Recoveries[id]
	if !ok {
		return nil, errors.New(errPath, fmt.Sprintf("Recovery %s not found", id))
	}
	return Recovery, nil

}
