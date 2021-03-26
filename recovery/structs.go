package recovery

import (
	"github.com/morrocker/broadcast"
	"github.com/morrocker/log"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/config"
)

// Recovery stores a single recovery data
type Recovery struct {
	Data        *Data    `json:"data"`
	LoginServer string   `json:"login"`
	Status      State    `json:"status"`
	Priority    Priority `json:"-"`

	OutputTo    string                 `json:"outputTo"`
	Step        Step                   `json:"step"`
	cloud       config.Cloud           `json:"-"`
	RBS         *RBS                   `json:"-"`
	broadcaster *broadcast.Broadcaster `json:"-"`
	tracker     *tracker.SuperTracker  `json:"-"`
	log         *log.Logger            `json:"-"`
}

// Data stores the data needed to execute a recovery
type Data struct {
	ID         int             `json:"id"`
	TotalSize  int64           `json:"totalSize"`
	TotalFiles int64           `json:"totalFiles"`
	User       string          `json:"user"`
	Machine    string          `json:"machine"`
	Metafile   string          `json:"metafile"`
	Repository string          `json:"repository"`
	Disk       string          `json:"disk"`
	Org        string          `json:"org"`
	Deleted    bool            `json:"deleted"`
	Version    int             `json:"version"`
	Exclusions map[string]bool `json:"exclusions"`
	ClonerKey  string          `json:"-"`
}
