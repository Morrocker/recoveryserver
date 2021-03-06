package recovery

import (
	"time"

	"github.com/morrocker/logger"
)

const (
	exitNoCode = iota
	exitAlone
	exitDelete
)

// Run starts a recovery execution
func (r *Recovery) Run() {
	r.Status = Stop
	exitCode := 0
	logger.Info("Recovery %s worker is waiting to start!", r.ID)
	for {
		switch r.Status {
		case Start:
			goto Metafiles
		case Cancel:
			exitCode = exitAlone
			goto EndPoint
		}
	}
Metafiles:
	for x := 0; x < 3; x++ {
		logger.Info("Finding metafiles for recovery %s", r.ID)
		time.Sleep(10 * time.Second)
	}
	for x := 0; x < 5; x++ {
		for {
			switch r.Status {
			case Start:
				goto Next
			case Cancel:
				exitCode = exitDelete
				goto EndPoint
			}
			time.Sleep(5 * time.Second)
		}
	Next:
		logger.Info("Recovering file #%d from recovery %s", x, r.ID)
		time.Sleep(5 * time.Second)
	}
EndPoint:
	if exitCode == exitDelete {
		r.RemoveFiles()
	}
	r.Status = Done
}

// LegacyRecovery recovers files using legacy blockserver remote
func (r *Recovery) LegacyRecovery() {}

// Recovery recovers files using current blockserver remote
func (r *Recovery) Recovery() {}

// RemoveFiles removes any recovered file from the destination location
func (r *Recovery) RemoveFiles() {
	logger.Task("We are happily removing these files")
}
