package recovery

import "github.com/morrocker/errors"

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
	ID          string
	Info        Data
	Destination string
	Status      int
	Priority    int
}

// Data stores the data needed to execute a recovery
type Data struct {
	User         string
	Machine      string
	Metafile     string
	Repository   string
	Disk         string
	Organization int
	Deleted      bool
	Date         string
}

// Multiple stores multiple recoveries
type Multiple struct {
	Recoveries []Data
}

var e = errors.Error{Path: "recovery"}

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
