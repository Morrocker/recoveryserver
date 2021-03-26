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

type Priority int
type State int
type Step int

const (
	// Entry default entry status for a recovery
	Entry State = iota
	// Queue recovery in queue to initilize recovery worker
	Queued
	// Start Recovery running currently
	Running
	// Pause Recovery temporarily stopped
	Paused
	// Done Recovery finished
	Done
	// Cancel Recovery to be removed
	Canceled
)

const (
	// VeryLowPr just a priority
	VeryLowPr Priority = iota
	// LowPr just a priority
	LowPr
	// MediumPr just a priority
	MediumPr
	// HighPr just a priority
	HighPr
	// VeryHighPr just a priority
	VeryHighPr
	// UrgentPr just a priority
	UrgentPr
)

const (
	Metafiles Step = iota
	Files
)

// New returns a new Recovery object from the given recovery data
func New(id int, data *Data, bc *broadcast.Broadcaster, cl config.Cloud) *Recovery {
	newRecovery := &Recovery{
		Data:        data,
		Priority:    MediumPr,
		broadcaster: bc,
	}
	newRecovery.SetCloud(cl)
	return newRecovery
}

// Pause stops a recovery execution
func (r *Recovery) Pause() error {
	log.Task("Pausing recovery %d", r.Data.ID)
	op := "recovery.Pause()"
	switch r.Status {
	case Running:
		r.changeState(Paused)
		return nil
	case Paused:
		return errors.New(op, fmt.Sprintf("Recovery #%d is already paused", r.Data.ID))
	default:
		return errors.New(op, fmt.Sprintf("Recovery #%d is not running", r.Data.ID))
	}
}

// Start starts (or resumes) a recovery execution
func (r *Recovery) Start() error {
	log.Task("Running recovery #%d", r.Data.ID)
	switch r.Status {
	case Paused, Running:
		r.changeState(Running)
		return nil
	default:
		return errors.New("recovery.Start()", fmt.Sprintf("Recovery #%d must be queued or paused to start", r.Data.ID))
	}
}

// Start starts (or resumes) a recovery execution
func (r *Recovery) done() error {
	log.TaskD("Setting recovery #%d as done.", r.Data.ID)
	switch r.Status {
	case Running:
		r.changeState(Done)
		return nil
	default:
		return errors.New("recovery.Start()", fmt.Sprintf("Recovery #%d must be running to be set as Done", r.Data.ID))
	}
}

// Done sets a recovery status as Done
func (r *Recovery) doDone(finish time.Duration) error {
	op := "recovery.Done()"
	rate, err := r.tracker.TrueProgressRate("size")
	if err != nil {
		return errors.Extend(op, err)
	}
	log.Info("Recovery #%d finished in %s with an average download rate of %sps", r.Data.ID, finish, rate)
	return r.done()
}

// Done sets a recovery status as Done
func (r *Recovery) PreDone() {
	// op := "recovery.Done()"
	_, st, _ := r.tracker.RawValues("size")
	_, ft, _ := r.tracker.RawValues("files")
	r.Data.TotalSize = st
	r.Data.TotalFiles = ft
	r.changeState(Entry)
	time.Sleep(6 * time.Second)
	log.Info("Recovery #%d precalculation finished. Total size: %s, Total Files: %d", r.Data.ID, utils.B2H(st), ft)
}

// Cancel sets a recovery status as Cancel
func (r *Recovery) Cancel() error {
	log.TaskV("Canceling recovery #%d", r.Data.ID)
	op := "recovery.Cancel()"
	switch r.Status {
	case Entry:
		return errors.New(op, fmt.Sprintf("Recovery #%d is on Entry state Cancelling is irrelevant. Remove it if necesary", r.Data.ID))
	case Done:
		return errors.New(op, fmt.Sprintf("Recovery #%d Recovery is Done. Remove it first", r.Data.ID))
	default:
		r.changeState(Canceled)
		return nil
	}
}

// Queue sets a recovery status as Done
func (r *Recovery) Queue() error {
	log.TaskV("Queueing recovery #%d", r.Data.ID)
	op := "recovery.Queue()"
	switch r.Status {
	case Running:
		return errors.New(op, fmt.Sprintf("Recovery #%d is Running. Cancel it first", r.Data.ID))
	case Done:
		return errors.New(op, fmt.Sprintf("Recovery #%d is Done. Remove it first", r.Data.ID))
	default:
		r.changeState(Queued)
		return nil
	}
}

// Queue sets a recovery status as Done
func (r *Recovery) Unqueue() error {
	log.TaskV("Unqueueing recovery #%d", r.Data.ID)
	switch r.Status {
	case Queued:
		r.changeState(Entry)
		return nil
	default:
		return errors.New("recovery.Unqueue()", fmt.Sprintf("Recovery #%d must be Queued to be Unqueued", r.Data.ID))
	}
}

func (r *Recovery) SetCloud(rc config.Cloud) {
	r.LoginServer = rc.FilesAddress
	r.Data.ClonerKey = rc.ClonerKey
	r.cloud = rc
	r.RBS = NewRBS(rc)
}

func (r *Recovery) SetOutput(dst string) {
	r.OutputTo = dst
	r.notify()
}
func (r *Recovery) GetOutput() string {
	return r.OutputTo
}

func (r *Recovery) SetPriority(n int) error {
	p := Priority(n)
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
