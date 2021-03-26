package recovery

import (
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/utils"
)

func (r *Recovery) autoTrack() {
	l := r.broadcaster.Listen()
	for {
		tick := 5 * time.Second
		if r.Status == Running {
			r.tracker.StartAutoPrint(tick)
			if r.Step == Files {
				r.tracker.StartAutoMeasure("size", tick)
			}
		} else {
			r.tracker.StopAutoMeasure("size")
			r.tracker.StopAutoPrint()
			if r.Status == Done || r.Status == Canceled {
				break
			}
		}
		<-l.C
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
	r.tracker.UnitsFunc("size", utils.B2H)
	r.tracker.PrintFunc(r.printFunction)
	return nil
}

func (r *Recovery) printFunction() {
	op := "recovery.printFunction()"
	fc, ft, err := r.tracker.RawValues("files")
	if err != nil {
		log.Errorln(errors.New(op, err))
	}
	bc, bt, err := r.tracker.RawValues("blocks")
	if err != nil {
		log.Errorln(errors.New(op, err))
	}
	sc, st, err := r.tracker.Values("size")
	if err != nil {
		log.Errorln(errors.New(op, err))
	}
	ec, _, err := r.tracker.RawValues("errors")
	if err != nil {
		log.Errorln(errors.New(op, err))
	}
	rt, err := r.tracker.ProgressRate("size")
	if err != nil {
		log.Errorln(errors.New(op, err))
	}
	eta, err := r.tracker.ETA("size")
	if err != nil {
		log.Errorln(errors.New(op, err))
	}
	if r.Step == Metafiles {
		log.Notice("[ Building Filetree ] Files: %d | Blocks: %d | Size: %s",
			ft, bt, st)
	} else if r.Step == Files {
		log.Notice("[ Downloading Files ] Files: %d / %d | Blocks: %d / %d | Size: %s / %s | Errors: %d [ %sps | %s ]",
			fc, ft, bc, bt, sc, st, ec, rt, eta)
	}
}
