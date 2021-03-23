package recovery

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/remotes"
	"github.com/morrocker/recoveryserver/utils"
)

// New returns a new Recovery object from the given recovery data
func New(id int, data *Data) *Recovery {
	newRecovery := &Recovery{Data: data, Priority: MediumPr}
	return newRecovery
}

// Pause stops a recovery execution
func (r *Recovery) Pause() {
	log.Task("Pausing recovery %d", r.Data.ID)
	r.Status = Paused
}

// Start starts (or resumes) a recovery execution
func (r *Recovery) Start() {
	log.Task("Running recovery %d", r.Data.ID)
	r.Status = Running
}

// Done sets a recovery status as Done
func (r *Recovery) Done(finish time.Duration) error {
	op := "recovery.Done()"
	if err := r.tracker.StopAutoMeasure("size"); err != nil {
		log.Errorln(errors.New(op, err))
	}
	r.tracker.StopAutoPrint()
	rate, err := r.tracker.GetTrueProgressRate("size")
	if err != nil {
		return errors.Extend(op, err)
	}
	log.Info("Recovery finished in %s with an average download rate of %sps", finish, rate)
	r.Status = Done
	return nil
}

// Done sets a recovery status as Done
func (r *Recovery) PreDone() error {
	op := "recovery.Done()"
	if err := r.tracker.StopAutoMeasure("size"); err != nil {
		log.Errorln(errors.New(op, err))
	}
	r.tracker.StopAutoPrint()
	_, st, err := r.tracker.GetRawValues("size")
	if err != nil {
		return errors.Extend(op, err)
	}
	r.Data.TotalSize = st
	log.Info("Recovery precalculation finished. Total size %s", st)
	r.Status = Queued // FIX THIS
	return nil
}

// Cancel sets a recovery status as Cancel
func (r *Recovery) Cancel() {
	log.Task("Canceling recovery %d", r.Data.ID)
	r.Status = Canceled
}

// Queue sets a recovery status as Done
func (r *Recovery) Queue() {
	log.Task("Queueing recovery %d", r.Data.ID)
	r.Status = Queued
}

// Queue sets a recovery status as Done
func (r *Recovery) Unqueue() {
	log.Task("Unqueueing recovery %d", r.Data.ID)
	r.Status = Entry
}

// StartTracker starts a new tracker for a Recovery
func (r *Recovery) startTracker() error {
	st, err := tracker.New()
	r.tracker = st
	if err != nil {
		return errors.New("recovery.StartTracker()", err)
	}
	r.tracker.AddGauge("files", "Files", 0)
	r.tracker.Reset("files")
	r.tracker.AddGauge("blocks", "Blocks", 0)
	r.tracker.Reset("blocks")
	r.tracker.AddGauge("size", "Size", 0)
	r.tracker.Reset("size")
	r.tracker.AddGauge("errors", "Errors", 0)
	r.tracker.Reset("errors")
	r.tracker.InitSpdRate("size", 40)
	r.tracker.SetEtaTracker("size")
	r.tracker.SetProgressFunction("size", utils.B2H)
	return nil
}

func (r *Recovery) SetCloud(rc *remotes.Cloud) {
	r.cloud = rc
	r.Data.ClonerKey = rc.ClonerKey
}

func (r *Recovery) SetDstn(dst string) {
	r.destination = dst
}
func (r *Recovery) GetDstn() string {
	return r.destination
}

func (r *Recovery) SetPriority(p int) error {
	if p > VeryHighPr || p < VeryLowPr {
		return errors.New("recovery.SetPriority()", "Priority value outside allowed parameters")
	}
	r.Priority = p
	return nil
}

func (r *Recovery) updateTrackerTotals(size int64) {
	blocks := int64(1)                // fileblock
	blocks += (int64(size) / 1024000) // 1 MB blocks
	remainder := size % 1024000
	if remainder != 0 {
		blocks++
	}
	r.tracker.ChangeTotal("size", size)
	r.tracker.ChangeTotal("files", 1)
	r.tracker.ChangeTotal("blocks", blocks)
}

func (r *Recovery) updateTrackerCurrent(size int64) {
	blocks := int64(1)                // fileblock
	blocks += (int64(size) / 1024000) // 1 MB blocks
	remainder := size % 1024000
	if remainder != 0 {
		blocks++
	}
	r.tracker.ChangeCurr("size", size)
	r.tracker.ChangeCurr("files", 1)
	r.tracker.ChangeCurr("blocks", blocks)
}

func (r *Recovery) increaseErrors() {
	r.tracker.IncreaseCurr("errors")
}

func (r *Recovery) initLogger() {
	op := "recovery.initLogger()"
	Log := log.New()
	now := time.Now().Format("2006-01-02T15h04m")
	logName := fmt.Sprintf("%s.%s.%s.%s.log", r.Data.User, r.Data.Machine, r.Data.Disk, now)
	logPath := path.Join(config.Data.RcvrLogDir, r.Data.Org, logName)
	if err := os.MkdirAll(path.Join(config.Data.RcvrLogDir, r.Data.Org), 0700); err != nil {
		log.Error(op, err)
		os.Exit(1)
	}
	log.Info("Setting output file (for recovery) to %s", logPath)
	Log.OutputFile(logPath)
	Log.ToggleSilent()
	Log.StartWriter()
	Log.SetScope(true, true, true)
	Log.SetMode("verbose")
	r.log = Log
}
