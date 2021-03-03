package recovery

import "sync"

// Recovery stores a single recovery data
type Recovery struct {
	Machine    string
	Metafile   string
	Repository string
	User       string
	Disk       string
	Deleted    bool
	Date       string
	Done       bool
}

type RecoveryGroup struct {
	Recoveries   map[string]Recovery
	Organization string
	Priority     int
	Lock         sync.Mutex
}
