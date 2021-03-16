package recovery

import (
	"time"
)

func (r *Recovery) stopGate() {
Loop:
	for {
		switch r.Status {
		case Start:
			break Loop
		}
		time.Sleep(2 * time.Second)
	}
}
