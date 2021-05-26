package recovery

import (
	"github.com/morrocker/flow"
	tracker "github.com/morrocker/progress-tracker"
	track "github.com/morrocker/recoveryserver/recovery2/tracker"
)

type Recovery struct {
	data       Data
	resources  Resources
	status     State
	priority   Prty
	outputPath string

	progress   *tracker.SuperTracker
	controller *flow.Controller
}

type Data struct {
	Metafile   string
	Repository string
	User       string
	Version    string
	Deleted    bool
	Exclusions map[string]bool
	Key        string
}

type Resources struct {
	FileWorkers int
	MetaWorkers int
	MetaBuffer  int
}

func New(d Data, r Resources) *Recovery {
	newRecovery := &Recovery{
		data:       d,
		resources:  r,
		status:     Entry,
		priority:   MediumPr,
		progress:   track.New(),
		controller: flow.New(),
	}
	return newRecovery
}

func (r *Recovery) Run() {
	r.controller.Go()
}

func (r *Recovery) Stop() {
	r.controller.Stop()
}

func (r *Recovery) Cancel() {
	r.controller.Exit(int(Canceled))
}

func (r *Recovery) Progress() {}

func (r *Recovery) Output(o ...string) string {
	if len(o) != 0 {
		r.outputPath = o[0]
	}
	return r.outputPath
}

func (r *Recovery) Prty(p ...Prty) Prty {
	if len(p) != 0 {
		r.priority = p[0]
	}
	return r.priority
}
