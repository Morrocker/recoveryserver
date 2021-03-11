package recovery

import (
	"github.com/morrocker/errors"
	"github.com/morrocker/logger"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/remotes"
)

const (
	// Entry default entry status for a recovery
	Entry = iota
	// Queue recovery in queue to initilize recovery worker
	Queue
	// Stop recovery queued and waiting to start running
	Stop
	// Start Recovery running currently
	Start
	// Pause Recovery temporarily stopped
	Pause
	// Done Recovery finished
	Done
	// Cancel Recovery to be removed
	Cancel
)

const (
	// VeryLowPr just a priority
	VeryLowPr = iota
	// LowPr just a priority
	LowPr
	// MediumPr just a priority
	MediumPr
	// HighPr just a priority
	HighPr
	// VeryHighPr just a priority
	VeryHighPr
	// UrgentPr just a priority
	UrgentPr
)

// Recovery stores a single recovery data
type Recovery struct {
	ID           string
	Data         *Data
	Destination  string
	Status       int
	Priority     int
	Cloud        *remotes.Cloud
	SuperTracker *tracker.SuperTracker
}

// Data stores the data needed to execute a recovery
type Data struct {
	User       string
	Machine    string
	Metafile   string
	Repository string
	Disk       string
	RootGroup  int
	Deleted    bool
	Date       string
	Version    int
	Exclusions map[string]bool
	Server     string
	ClonerKey  string
}

// Multiple stores multiple recoveries
type Multiple struct {
	Recoveries []*Data
}

// New returns a new Recovery object from the given recovery data
func New(id string, data *Data) *Recovery {
	errPath := "recovery.New()"
	newRecovery := &Recovery{Data: data, ID: id, Priority: MediumPr}
	if err := newRecovery.StartTracker(); err != nil {
		logger.Alert("%s", errors.Extend(errPath, err))
	}
	return newRecovery
}

// Pause stops a recovery execution
func (r *Recovery) Pause() {
	r.Status = Pause
}

// Start starts (or resumes) a recovery execution
func (r *Recovery) Start() {
	r.Status = Start
}

// Done sets a recovery status as Done
func (r *Recovery) Done() {
	r.Status = Done
}

// Cancel sets a recovery status as Cancel
func (r *Recovery) Cancel() {

}

// Queue sets a recovery status as Done
func (r *Recovery) Queue() {
	r.Status = Queue
}

// StartTracker starts a new tracker for a Recovery
func (r *Recovery) StartTracker() error {
	errPath := "recovery.StartTracker()"
	st, err := tracker.New()
	r.SuperTracker = st
	if err != nil {
		err = errors.New(errPath, err)
		logger.Error("%v", err)
		return err
	}
	r.SuperTracker.AddGauge("files", "Files", 0)
	r.SuperTracker.AddGauge("blocks", "Blocks", 0)
	r.SuperTracker.AddGauge("size", "Size", 0)
	r.SuperTracker.AddGauge("errors", "Errors", 0)
	return nil
}

func (r *Recovery) SetCloud(rc *remotes.Cloud) {
	r.Cloud = rc
	r.Data.ClonerKey = rc.ClonerKey
}

func (r *Recovery) SetDestination(dst string) {
	r.Destination = dst
}

func (r *Recovery) SetPriority(p int) error {
	errPath := "recovery.SetPriority()"
	if p > VeryHighPr || p < VeryLowPr {
		return errors.New(errPath, "Priority value outside allowed parameters")
	}
	r.Priority = p
	return nil
}

func (r *Recovery) updateTrackerTotals(size int64) {
	blocks := int64(1)                // fileblock
	blocks += (int64(size) / 1024000) // 1 MB blocks
	remainder := size % 1024000
	if remainder != 0 {
		blocks++
	}
	r.SuperTracker.ChangeTotal("size", size)
	r.SuperTracker.ChangeTotal("files", 1)
	r.SuperTracker.ChangeTotal("blocks", blocks)
}

func (r *Recovery) updateTrackerCurrent(size int64) {
	blocks := int64(1)                // fileblock
	blocks += (int64(size) / 1024000) // 1 MB blocks
	remainder := size % 1024000
	if remainder != 0 {
		blocks++
	}
	r.SuperTracker.ChangeCurr("size", size)
	r.SuperTracker.ChangeCurr("files", 1)
	r.SuperTracker.ChangeCurr("blocks", blocks)
}

func (r *Recovery) increaseErrors() {
	r.SuperTracker.IncreaseCurr("errors")
}
