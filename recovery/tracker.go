package recovery

import (
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/utils"
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
	log.Notice("[ %s ] Files: %s / %s | Blocks: %s / %s | Size: %s / %s | Errors: %s [ %s | %s ]",
		r.Status, fc, ft, bc, bt, sc, st, ec, rt, eta)
}
