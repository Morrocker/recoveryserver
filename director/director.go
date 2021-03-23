package director

import (
	"sync"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/disks"
	"github.com/morrocker/recoveryserver/pdf"
	"github.com/morrocker/recoveryserver/recovery"
)

// Director orders and decides which recoveries should be executed next
type Director struct {
	Run bool

	statusMonitor chan interface{}
	Recoveries    map[int]*recovery.Recovery
	devices       map[string]disks.Device
	RunLock       sync.Mutex
	Lock          sync.Mutex
}

// StartDirector starts the Director service and all subservices
func (d *Director) StartDirector() error {
	log.Task("Starting director services")
	ec := make(chan error)

	d.init()
	go d.devicesScanner()
	go d.recoveryPicker()
	<-ec
	log.Info("Shutting down director")
	return nil
}

func (d *Director) init() {
	d.devices = make(map[string]disks.Device)
	d.Recoveries = make(map[int]*recovery.Recovery)
	d.statusMonitor = make(chan interface{})
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

// RETHINKG THE PLACE OF THIS
func (d *Director) WriteDelivery(p *pdf.Delivery) (out string, err error) {
	out, err = p.CreateDeliveryPDF(config.Data.DeliveryDir)
	if err != nil {
		err = errors.Extend("director.WriteDelivery()", err)
		return
	}
	return
}
