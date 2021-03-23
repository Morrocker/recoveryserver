package director

import (
	"fmt"
	"sync"
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/disks"
	"github.com/morrocker/recoveryserver/pdf"
	"github.com/morrocker/recoveryserver/recovery"
	"github.com/morrocker/recoveryserver/remotes"
)

// Director orders and decides which recoveries should be executed next
type Director struct {
	Run       bool
	AutoQueue bool

	Clouds     map[string]*remotes.Cloud
	Recoveries map[int]*recovery.Recovery
	devices    map[string]disks.Device
	RunLock    sync.Mutex
	Lock       sync.Mutex
}

// StartDirector starts the Director service and all subservices
func (d *Director) StartDirector() error {
	log.Task("Starting director services")
	ec := make(chan error)

	d.init()
	go d.devicesScanner()
	go d.startWorkers()
	go d.recoveryPicker()
	<-ec
	log.Info("Shutting down director")
	return nil
}

func (d *Director) init() {
	op := "director.init()"
	d.Clouds = make(map[string]*remotes.Cloud)
	d.devices = make(map[string]disks.Device)
	d.initClouds()
	d.Recoveries = make(map[int]*recovery.Recovery)
	if err := d.readRecoveryJSON(); err != nil {
		log.Errorln(errors.New(op, err))
	}
	d.Run = config.Data.AutoRunRecoveries
	if config.Data.AutoQueueRecoveries {
		for _, r := range d.Recoveries {
			if err := d.QueueRecovery(r.Data.ID); err != nil {
				log.Alertln(errors.Extend(op, err))
			}
		}
	}
}

func (d *Director) startWorkers() {
	log.TaskV("Starting recovery workers creator")
	for {
		d.Lock.Lock()
		for key, recover := range d.Recoveries {
			if recover.Status == recovery.Queued {
				go d.Recoveries[key].Run(&d.RunLock)
			}
		}
		d.Lock.Unlock()
		// Maybe put a channel here?
		// |
		// V
		time.Sleep(5 * time.Second)
	}
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
		return errors.New(op, "Metafil parameter empty or missing")
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

	d.Recoveries[data.ID] = recovery.New(data.ID, data)
	if d.AutoQueue {
		if err := d.QueueRecovery(data.ID); err != nil {
			log.Alertln(errors.Extend(op, err))
		}
	}
	if err := d.writeRecoveryJSON(); err != nil {
		return errors.Extend(op, err)
	}
	return nil
}

// PickRecovery decides what recovery must be executed next. It prefers higher priority over lower. Will skip if a recovery is running. Has a low latency by design.
func (d *Director) recoveryPicker() {
	log.TaskV("Starting recovery picker")
	for {
	Start:
		if !d.Run {
			log.InfoD("Director set Run to false. Sleeping")
			time.Sleep(30 * time.Second)
			continue
		}
		log.InfoD("Trying to decide new recovery to run")
		var nextRecovery *recovery.Recovery
		var nextPriority int = -1
		for _, r := range d.Recoveries {
			if r.Status == recovery.Running {
				log.InfoD("A recovery is already running. Sleeping for a while")
				time.Sleep(5 * time.Second)
				goto Start
			}
			if r.Status == recovery.Stopped && nextPriority < r.Priority && r.GetDstn() != "" {
				log.InfoD("Found possible recovery ID:%d", r.Data.ID)
				nextRecovery = r
				nextPriority = r.Priority
			}
		}
		if nextRecovery == nil {
			log.InfoD("No recovery found. Starting again.")
			time.Sleep(5 * time.Second)
			continue
		}
		go nextRecovery.Start()
		// Maybe put a channel here?
		// |
		// V
		time.Sleep(5 * time.Second)
		continue
	}
}

// ChangePriority changes a given recovery priority to a specific value
func (d *Director) ChangePriority(id int, value int) error {
	log.TaskV("Changing recovery %s Priority to %d", id, value)
	op := "director.ChangePriority"
	d.Lock.Lock()
	defer d.Lock.Unlock()
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(op, err)
	}
	r.SetPriority(value)
	return nil
}

// PausePicker stops the PickRecovery so that it does not continue launching recoveries
func (d *Director) PausePicker() {
	log.TaskV("Pausing recovery picker")
	d.Lock.Lock()
	defer d.Lock.Unlock()
	d.Run = false
}

// RunPicker starts or resumes the PickRecovery function
func (d *Director) RunPicker() {
	log.TaskV("Resuming recovery picker")
	d.Lock.Lock()
	defer d.Lock.Unlock()
	d.Run = true
}

// PauseRecovery sets a given recover status to Pause
func (d *Director) PauseRecovery(id int) error {
	log.TaskD("Pausing recovery %s", id)
	op := "director.PauseRecovery()"
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend(op, err)
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
	if err := r.GetLogin(); err != nil {
		return errors.Extend(op, err)
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
		return errors.New(op, "Failed to find match recovery with any existing Cloud")
	}
	r.Queue()
	return nil
}

func (d *Director) initClouds() {
	log.TaskV("Initializing Remote Clouds")
	for _, cloud := range config.Data.Clouds {
		d.Clouds[cloud.FilesAddress] = remotes.NewCloud(cloud)
	}
}

// SetDestination
func (d *Director) SetDestination(id int, dst string) (err error) {
	r, err := d.findRecovery(id)
	if err != nil {
		return errors.Extend("director.SetDestination()", err)
	}
	r.SetDstn(dst)
	return
}

// RETHINKG THE PLACE OF THIS
func (d *Director) WriteDelivery(p *pdf.Delivery) (out string, err error) {
	out, err = p.CreateDeliveryPDF(config.Data.DeliveryDir)
	if err != nil {
		err = errors.Extend("director.WriteDelivery()", err)
		return
	}
	return
}

// Stop sets Run to false
func (d *Director) Stop() {
	log.TaskV("Setting Director.Run to false")
	d.Run = false
}

// Start sets Run to true
func (d *Director) Start() {
	log.TaskV("Setting Director.Run to true")
	d.Run = true
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

// accesory functions

func (d *Director) findRecovery(id int) (*recovery.Recovery, error) {
	Recovery, ok := d.Recoveries[id]
	if !ok {
		return nil, errors.New("recovery.findRecovery()", fmt.Sprintf("Recovery %d not found", id))
	}
	return Recovery, nil

}

func (d *Director) Devices() (map[string]disks.Device, error) {
	return d.devices, nil
}

func (d *Director) devicesScanner() {
	log.TaskV("Starting Devices Scanner")
	for {
		devs, err := disks.FindDevices()
		if err != nil {
			log.Alertln(errors.Extend("director.Devices()", err))
		}
		for key, dev := range devs {
			if oldDev, ok := d.devices[key]; ok {
				dev.Status = oldDev.Status
			}
		}
		d.devices = devs
		time.Sleep(3 * time.Second)
	}
}

func (d *Director) MountDisk(serial string) error {
	if err := d.devices[serial].Mount(); err != nil {
		return errors.Extend("director.MountDisk()", err)
	}
	return nil
}

func (d *Director) UnmountDisk(serial string) error {
	if err := d.devices[serial].Unmount(); err != nil {
		return errors.Extend("director.UnmountDisk()", err)
	}
	return nil
}
