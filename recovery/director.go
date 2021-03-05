package recovery

import (
	"errors"
	"sync"
	"time"

	"github.com/morrocker/logger"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/utils"
)

// Director orders and decides which recoveries should be executed next
type Director struct {
	Run       bool
	AutoQueue bool

	Order      int
	Clouds     map[string]*Cloud
	Recoveries map[string]*Recovery
	Lock       sync.Mutex
}

// StartDirector starts the Director service and all subservices
func (d *Director) StartDirector(c config.Config) {
	logger.TaskV("Starting director services")
	//LOAD CONFIG HERE
	d.Clouds = make(map[string]*Cloud)
	d.Recoveries = make(map[string]*Recovery)
	d.Run = c.AutoRunRecoveries
	d.AutoQueue = c.AutoQueueRecoveries
	logger.TaskV("Running recovery workers creator")
	go d.StartWorkers()
	logger.TaskV("Running recovery picker")
	go d.PickRecovery()
}

// StartWorkers continually tries to start workers for each recovery added
func (d *Director) StartWorkers() {
	for {
		d.Lock.Lock()
		for key, recover := range d.Recoveries {
			if recover.Status == Queue {
				go d.Recoveries[key].Run()
			}
		}
		d.Lock.Unlock()
		time.Sleep(10 * time.Second)
	}
}

// AddRecovery adds the given recovery data to create a new entry on the Recoveries map
func (d *Director) AddRecovery(r Data) (hash string) {
	hash = utils.RandString(8)
	d.Recoveries[hash] = &Recovery{Info: r, ID: hash, Priority: Medium}
	if d.AutoQueue {
		d.Recoveries[hash].Status = Queue
	}
	return
}

// Stop sets Run to false
func (d *Director) Stop() {
	d.Run = false
}

// Start sets Run to true
func (d *Director) Start() {
	d.Run = true
}

// PickRecovery decides what recovery must be executed next. It prefers higher priority over lower. Will skip if a recovery is running. Has a low latency by design.
func (d *Director) PickRecovery() {
	logger.TaskV("Starting PickPecovery")
	for {
	Start:
		if !d.Run {
			logger.Info("Director set Run to false. Sleeping")
			time.Sleep(30 * time.Second)
			continue
		}
		logger.Info("Trying to decide new recovery to run")
		var nextRecovery *Recovery = &Recovery{Priority: -1}
		var nextRecoveryHash string
		for hash, recovery := range d.Recoveries {
			if recovery.Status == Start {
				logger.Info("A recovery is already running. Sleeping for a while")
				time.Sleep(30 * time.Second)
				goto Start
			}

			if recovery.Status == Stop && nextRecovery.Priority < recovery.Priority {
				logger.Info("Found possible recovery %s", nextRecoveryHash)
				nextRecovery = recovery
				nextRecoveryHash = hash
			}
		}
		if nextRecoveryHash == "" {
			logger.Info("No recovery found. Starting again.")
			time.Sleep(30 * time.Second)
			continue
		}
		d.Recoveries[nextRecoveryHash].Status = Start
		// 30 seconds is for test purposes. Change later
		time.Sleep(30 * time.Second)
		continue
	}
}

// ChangePriority changes a given recovery priority to a specific value
func (d *Director) ChangePriority(id string, value int) {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	d.Recoveries[id].Priority = value
}

// PausePicker stops the PickRecovery so that it does not continue launching recoveries
func (d *Director) PausePicker() {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	d.Run = false
}

// RunPicker starts or resumes the PickRecovery function
func (d *Director) RunPicker() {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	d.Run = true
}

// PauseRecovery sets a given recover status to Pause
func (d *Director) PauseRecovery(id string) error {
	recovery, ok := d.Recoveries[id]
	if !ok {
		return errors.New("Recovery not found")
	}
	recovery.Status = Pause
	return nil
}

// StartRecovery sets a given recovery status to Start // TODO: See how resuming works
func (d *Director) StartRecovery(id string) error {
	recovery, ok := d.Recoveries[id]
	if !ok {
		return errors.New("Recovery not found")
	}
	recovery.Status = Start
	return nil
}

// QueueRecovery sets a recovery status to Queue. Needed when autoqueue is off.
func (d *Director) QueueRecovery(id string) error {
	recovery, ok := d.Recoveries[id]
	if !ok {
		return errors.New("Recovery not found")
	}
	recovery.Status = Queue
	return nil
}
