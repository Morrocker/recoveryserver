package recovery

import (
	"sync"
)

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
	info     *Data
	Status   int
	Priority int
}

type Data struct {
	User         string
	Machine      string
	Metafile     string
	Repository   string
	Disk         string
	Organization string
	Deleted      bool
	Date         string
}

// Group stores a group of recoveries to be ejecuted
type Organizer struct {
	Recoveries map[string]Recovery
	Lock       sync.Mutex
}

// AddRecovery adds a single recovery to a recovery group
// func (g *Group) AddRecovery(r Recovery) string {
// 	g.Lock.Lock()
// 	defer g.Lock.Unlock()
// 	hash := utils.RandString(8)
// 	g.Recoveries[hash] = r
// 	return hash
// }

// Pause stops a recovery execution
func (r *Recovery) Pause() {
	r.Status = Stop
}

// Start starts (or resumes) a recovery execution
func (r *Recovery) Start() {
	r.Status = Start
}

//
