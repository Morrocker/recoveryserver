package recovery

import (
	"github.com/morrocker/log"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/remotes"
)

// Recovery stores a single recovery data
type Recovery struct {
	Data     *Data
	Status   int
	Priority int

	destination string
	step        int
	cloud       *remotes.Cloud
	tracker     *tracker.SuperTracker
	log         *log.Logger
}

// Data stores the data needed to execute a recovery
type Data struct {
	ID         int
	TotalSize  int64
	User       string
	Machine    string
	Metafile   string
	Repository string
	Disk       string
	Org        string
	Deleted    bool
	Version    int
	Exclusions map[string]bool
	Server     string
	ClonerKey  string
}

// Multiple stores multiple recoveries
type Multiple struct {
	Recoveries []*Data
}

const (
	// Entry default entry status for a recovery
	Entry = iota
	// Queue recovery in queue to initilize recovery worker
	Queued
	// Stop recovery queued and waiting to start running
	Stopped
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
	VeryLowPr = iota
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
	Metafiles = iota
	Files
)
