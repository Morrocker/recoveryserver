package recovery

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/broadcast"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/utils"
)

// New returns a new Recovery object from the given recovery data
func New(id int, data *Data, bc *broadcast.Broadcaster) *Recovery {
	newRecovery := &Recovery{
		Data:        data,
		Priority:    MediumPr,
		broadcaster: bc,
	}
	return newRecovery
}

// Pause stops a recovery execution
func (r *Recovery) Pause() {
	log.Task("Pausing recovery %d", r.Data.ID)
	r.Status = Paused
	r.notify()
}

// Start starts (or resumes) a recovery execution
func (r *Recovery) Start() {
	log.Task("Running recovery #%d", r.Data.ID)
	r.Status = Running
	r.notify()
}

// Done sets a recovery status as Done
func (r *Recovery) Done(finish time.Duration) error {
	op := "recovery.Done()"
	rate, err := r.tracker.TrueProgressRate("size")
	if err != nil {
		return errors.Extend(op, err)
	}
	log.Info("Recovery #%d finished in %s with an average download rate of %sps", r.Data.ID, finish, rate)
	r.Status = Done
	r.notify()
	return nil
}

// Done sets a recovery status as Done
func (r *Recovery) PreDone() {
	// op := "recovery.Done()"
	_, st, _ := r.tracker.RawValues("size")
	_, ft, _ := r.tracker.RawValues("files")
	r.Data.TotalSize = st
	r.Data.TotalFiles = ft
	r.Status = Entry
	r.notify()
	time.Sleep(6 * time.Second)
	log.Info("Recovery #%d precalculation finished. Total size: %s, Total Files: %d", r.Data.ID, utils.B2H(st), ft)
}

// Cancel sets a recovery status as Cancel
func (r *Recovery) Cancel() {
	log.TaskV("Canceling recovery #%d", r.Data.ID)
	r.Status = Canceled
	r.notify()
}

// Queue sets a recovery status as Done
func (r *Recovery) Queue() {
	log.TaskV("Queueing recovery #%d", r.Data.ID)
	r.Status = Queued
	r.notify()
}

// Queue sets a recovery status as Done
func (r *Recovery) Unqueue() {
	log.TaskV("Unqueueing recovery #%d", r.Data.ID)
	r.Status = Entry
	r.notify()
}

func (r *Recovery) SetCloud(rc config.Cloud) {
	r.LoginServer = rc.FilesAddress
	r.Data.ClonerKey = rc.ClonerKey
	r.RBS = NewRBS(rc)
}

func (r *Recovery) SetOutput(dst string) {
	r.outputTo = dst
	r.notify()
}
func (r *Recovery) GetOutput() string {
	return r.outputTo
}

func (r *Recovery) SetPriority(p int) error {
	if p > VeryHighPr || p < VeryLowPr {
		return errors.New("recovery.SetPriority()", "Priority value outside allowed parameters")
	}
	r.Priority = p
	return nil
}

func (r *Recovery) Step(s int) {
	r.step = s
	r.broadcaster.Broadcast()
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
	// GIVEN CHANGES TO THE TRACKER & LOGGER MAYBE CHANGES ARE NEEDED
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
