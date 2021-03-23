package recovery

import (
	"time"
)

func (r *Recovery) flowGate() bool {
	for {
		switch r.Status {
		case Running:
			return false
		case Paused, Canceled:
			if r.Status == Canceled {
				return true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (r *Recovery) notify() {
	r.statusMonitor <- ""
}
