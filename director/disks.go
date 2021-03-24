package director

import (
	"reflect"
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/disks"
)

func (d *Director) Devices() (map[string]disks.Device, error) {
	return d.devices, nil
}

func (d *Director) devicesScanner() {
	log.TaskV("Starting Devices Scanner")
	for {
		var previous map[string]disks.Device
		devs, err := disks.FindDevices()
		if err != nil {
			log.Alertln(errors.Extend("director.Devices()", err))
		}
		if !reflect.DeepEqual(previous, devs) {
			for key, dev := range devs {
				if oldDev, ok := d.devices[key]; ok {
					dev.Status = oldDev.Status
				}
			}
		}
		d.devices = devs
		previous = devs
		time.Sleep(time.Second)
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
