package recovery

import (
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
)

// Run starts a recovery execution
func (r *Recovery) Run() {
	op := "recovery.Run()"
	if err := r.Start(); err != nil {
		log.Errorln(errors.Extend(op, err))
		return
	}
	log.Info("Starting recovery %d", r.Data.ID)
	r.initLogger()
	r.startTracker()
	r.log.Task("Starting recovery %d", r.Data.ID)
	go r.autoTrack()

	// CHECK THIS POINT OR THE END. IT IS IMPORTANT TO CONSIDER DATA DUPLICATION IF RECOVEEY IS STOPPED > STARTED

	r.changeStep(Metafiles)
	if r.flowGate() {
		return
	}
	start := time.Now()
	tree, err := r.GetRecoveryTree()
	if err != nil {
		log.Errorln(errors.Extend(op, err))
		r.Cancel()
		return
	}
	r.changeStep(Files)
	if r.flowGate() {
		return
	}

	if err := r.getFiles(tree); err != nil {
		log.Errorln(errors.Extend(op, err))
		r.Cancel()
		return
	}
	r.tracker.Print()
	if r.flowGate() {
		return
	}
	if err := r.doDone(time.Since(start).Truncate(time.Second)); err != nil {
		log.Error(op, err)
	}
}

func (r *Recovery) PreCalculate() {
	log.TaskV("Precalculating recovery #%d size", r.Data.ID)
	op := "recovery.Run()"
	if err := r.Queue(); err != nil {
		log.Errorln(errors.Extend(op, err))
		return
	}

	if err := r.Start(); err != nil {
		log.Errorln(errors.Extend(op, err))
		return
	}
	log.Info("Starting recovery #%d size calculation", r.Data.ID)
	r.initLogger()
	r.startTracker()
	r.log.Task("Starting recovery #%d size calculation", r.Data.ID)
	go r.autoTrack()

	r.changeStep(Metafiles)
	if r.flowGate() {
		return
	}
	_, err := r.GetRecoveryTree()
	if err != nil {
		log.Errorln(errors.Extend(op, err))
		r.Cancel()
		return
	}
	if r.flowGate() {
		log.Infoln("Precalculations cancelled")
		return
	}
	r.PreDone()
}
