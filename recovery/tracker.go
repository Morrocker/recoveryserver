package recovery

import (
	"time"

	tracker "github.com/morrocker/progress-tracker"
)

func (r *Recovery) autoTrack() {
	tick := 5 * time.Second
	for {
		if r.Status == Running {
			r.tracker.StartAutoPrint(tick)
			if r.step == Files {
				r.tracker.StartAutoMeasure("size", tick)
			}
		} else {
			r.tracker.StopAutoMeasure("size")
			r.tracker.StopAutoPrint()
			if r.Status == Done || r.Status == Canceled {
				break
			}
		}
	}
}

// StartTracker starts a new tracker for a Recovery
func (r *Recovery) startTracker() error {
	st := tracker.New()
	r.tracker = st
	r.tracker.AddGauge("files", "Files", 0)
	r.tracker.Reset("files")
	r.tracker.AddGauge("blocks", "Blocks", 0)
	r.tracker.Reset("blocks")
	r.tracker.AddGauge("size", "Size", 0)
	r.tracker.Reset("size")
	r.tracker.AddGauge("errors", "Errors", 0)
	r.tracker.Reset("errors")
	r.tracker.InitSpdRate("size", 40)
	// r.tracker.EtaTracker("size")
	// r.tracker.ProgressFunction("size", utils.B2H)
	return nil
}
