package recovery

import (
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
)

func (r *Recovery) stopGate() int {
	op := "recoveries.stopGate()"
	for {
		switch r.Status {
		case Running:
			if r.step == Files {
				if err := r.tracker.StartAutoMeasure("size", 6); err != nil {
					log.Errorln(errors.New(op, err))
				}
			}
			r.tracker.StartAutoPrint()
			return 0
		case Paused, Canceled:
			if r.step == Files {
				if err := r.tracker.StopAutoMeasure("size"); err != nil {
					log.Errorln(errors.New(op, err))
				}
			}
			r.tracker.StopAutoPrint()
			if r.Status == Canceled {
				return Canceled
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}
