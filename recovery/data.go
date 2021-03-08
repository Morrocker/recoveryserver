package recovery

import (
	"github.com/morrocker/errors"
	"github.com/morrocker/logger"
	tracker "github.com/morrocker/progress-tracker"
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
	// VeryLow just a priority
	VeryLow = iota
	// Low just a priority
	Low
	// Medium just a priority
	Medium
	// High just a priority
	High
	// VeryHigh just a priority
	VeryHigh
	// Urgent just a priority
	Urgent
)

// Recovery stores a single recovery data
type Recovery struct {
	ID           string
	Info         Data
	Destination  string
	Status       int
	Priority     int
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
	Exclusions []string
	Server     string
	ClonerKey  string
}

// Multiple stores multiple recoveries
type Multiple struct {
	Recoveries []Data
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
	return nil
}
