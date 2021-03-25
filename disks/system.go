package disks

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
)

type Device struct {
	Status  string
	DevData Data
}

// Device represents a block device in tree-like format.
type Data struct {
	ID         int
	Serial     string
	Vendor     string
	Type       string
	Model      string
	Size       int64
	Status     string
	Partitions map[int]Partition
}

type Partition struct {
	Dev        string `json:"dev"`
	UUID       string `json:"uuid"`
	FsType     string `json:"fstype"`
	MountPoint string `json:"mount"`
	Size       int64  `json:"size"`
}

// Mount mounts partition on the given mount point.
func (d Device) Mount() error {
	op := "disks.Mount()"

	var part Partition
	for _, p := range d.DevData.Partitions {
		if p.FsType == "ntfs" && p.UUID != "" && part.Size < p.Size {
			part = p
		}
	}

	mountpoint := config.Data.MountRoot + d.DevData.Serial
	makeMountPoint(mountpoint)
	cmd := exec.Command(
		"sudo",
		"mount",
		filepath.Join(part.Dev),
		mountpoint,
	)

	if err := cmd.Run(); err != nil {
		return errors.Extend(op, err)
	}
	d.Status = "mounted"

	log.Notice("Mounted disk [Serial: %s], partition with UUID=%s on mountpoint %s", d.DevData.Serial, part.UUID, mountpoint)
	return nil
}

// Unmount unmounts the given partition.
func (d Device) Unmount() error {
	mountpoint := config.Data.MountRoot + d.DevData.Serial
	cmd := exec.Command(
		"sudo",
		"umount",
		mountpoint,
	)

	if err := cmd.Run(); err != nil {
		return errors.Extend("disks.Unmount()", err)
	}
	d.Status = "unmounted"
	removeMountPoint(mountpoint)

	log.Notice("Unmounted disk [Serial:%s]", d.DevData.Serial)
	return nil
}

// device represents the output fields of the lsblk command.
type device struct {
	MountPoint string   `json:"mountpoint"`
	FsType     string   `json:"fstype"`
	Serial     string   `json:"serial"`
	Name       string   `json:"name"`
	Vendor     string   `json:"vendor"`
	Type       string   `json:"type"`
	Model      string   `json:"model"`
	Size       int64    `json:"size,string"`
	UUID       string   `json:"uuid"`
	Children   []device `json:"children"`
}

func unmarshalDevices(raw []byte) (map[string]Device, error) {
	var out struct {
		BlockDevices []*device `json:"blockdevices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, errors.Extend("disks.unmarshallDevices", err)
	}

	devs := make(map[string]Device)
	for i, d := range out.BlockDevices {
		// Skip devices with no serial.
		if d.Serial == "" {
			continue
		}
		dev := Data{
			Serial:     d.Serial,
			Size:       d.Size,
			Vendor:     d.Vendor,
			Type:       d.Type,
			Model:      d.Model,
			Partitions: make(map[int]Partition),
		}
		for n, p := range out.BlockDevices[i].Children {
			dev.Partitions[n] = Partition{
				Dev:        path.Join("/dev", p.Name),
				UUID:       p.UUID,
				FsType:     p.FsType,
				MountPoint: p.MountPoint,
				Size:       p.Size,
			}
		}
		devs[d.Serial] = Device{
			DevData: dev,
		}
	}

	return devs, nil
}

// Devices retrieves a list of all devices that have only one children and a serial
// number.
func FindDevices() (map[string]Device, error) {
	raw, err := exec.Command("lsblk", "-JOb").Output()
	if err != nil {
		return nil, errors.Extend("disks.Devices()", err)
	}
	return unmarshalDevices(raw)
}

// MakeNTFS creates a NTFS file system on the given device.
func (d Device) MakeNTFS(p int) error {
	cmd := exec.Command(
		"sudo",
		"mkfs.ntfs",
		"-q",
		filepath.Join(d.DevData.Partitions[p].Dev),
	)

	d.Status = "formatting"
	if err := cmd.Run(); err != nil {
		return errors.Extend("disks.MakeNTFS()", err)
	}
	d.Status = "unmounted"

	return nil
}

func makeMountPoint(path string) error {
	log.Task("Creating mountpoint %s", path)
	if err := os.MkdirAll(path, 0700); err != nil {
		return errors.Extend("disks.makeMountPoint()", err)
	}
	return nil
}

func removeMountPoint(path string) error {
	log.Task("Removing mountpoint %s", path)
	if err := os.RemoveAll(path); err != nil {
		return errors.Extend("disks.removeMountPoint()", err)
	}
	return nil
}
