package recovery

const (
	Queue = iota
	Stop
	Start
	Done
	Cancel
)

const (
	Low = iota
	Medium
	High
	VeryHigh
	Urgent
)

// Recovery stores a single recovery data
type Recovery struct {
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

// Pause stops a recovery execution
func (r *Recovery) Pause() {
	r.Status = Stop
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

//
