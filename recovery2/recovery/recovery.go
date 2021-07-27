package recovery

import (
	"github.com/morrocker/errors"
	"github.com/morrocker/flow"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/recovery2/files"
	"github.com/morrocker/recoveryserver/recovery2/remote"
	track "github.com/morrocker/recoveryserver/recovery2/tracker"
	"github.com/morrocker/recoveryserver/recovery2/tree"
)

type Recovery struct {
	data       Data
	resources  Resources
	status     State
	priority   Prty
	outputPath string
	tree       *tree.MetaTree

	rbs        remote.RBS
	progress   *tracker.SuperTracker
	controller *flow.Controller
}

type Data struct {
	Metafile   string
	Repository string
	User       string
	Version    int
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

func (r *Recovery) GetTree() error {
	op := "recovery.GetTree()"
	newTree, err := tree.GetRecoveryTree(
		tree.Data{
			RootId:     r.data.Metafile,
			Repository: r.data.Repository,
			// Server: ,
			ClonerKey:  r.data.Key,
			Version:    r.data.Version,
			Deleted:    r.data.Deleted,
			Exclusions: r.data.Exclusions,
		}, tree.Throttling{
			BuffSize: r.resources.MetaWorkers,
			Workers:  r.resources.MetaBuffer,
		},
		r.progress,
		r.controller,
	)
	if err != nil {
		return errors.Extend(op, err)
	}
	r.tree = newTree
	return nil
}

func (r *Recovery) GetFiles() error {
	op := "recovery.GetFiles()"
	err := files.GetFiles(
		r.tree,
		r.outputPath,
		files.Data{
			User:    r.data.User,
			Workers: r.resources.FileWorkers,
		},
		r.rbs,
		r.progress,
		r.controller,
	)
	if err != nil {
		return errors.Extend(op, err)
	}
	return nil
}
