package recovery

import (
	"time"

	"github.com/morrocker/logger"
)

// Run starts a recovery execution
func (r *Recovery) Run() {
	r.Status = Stop
	for {
		logger.Info("Recovery: %s => Priority:%v, Status:%v", r.ID, r.Priority, r.Status)
		time.Sleep(10 * time.Second)
	}
}

// LegacyRecovery recovers files using legacy blockserver remote
func (r *Recovery) LegacyRecovery() {}

// Recovery recovers files using current blockserver remote
func (r *Recovery) Recovery() {}
