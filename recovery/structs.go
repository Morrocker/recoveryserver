package recovery

import (
	"github.com/morrocker/log"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/broadcast"
)

// Recovery stores a single recovery data
type Recovery struct {
	Data        *Data
	LoginServer string
	Status      int
	Priority    int

	outputTo string
	step     int
	// cloud       config.Cloud
	RBS         *RBS
	broadcaster *broadcast.Broadcaster
	tracker     *tracker.SuperTracker
	log         *log.Logger
}

// Data stores the data needed to execute a recovery
type Data struct {
	ID         int
	TotalSize  int64
	TotalFiles int64
	User       string
	Machine    string
	Metafile   string
	Repository string
	Disk       string
	Org        string
	Deleted    bool
	Version    int
	Exclusions map[string]bool
	ClonerKey  string
}

const (
	// Entry default entry status for a recovery
	Entry = iota
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
