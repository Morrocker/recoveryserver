package recovery

import (
	"github.com/morrocker/errors"
	"github.com/morrocker/flow"
	"github.com/morrocker/log"

	"github.com/morrocker/recoveryserver/recovery2/files"
	"github.com/morrocker/recoveryserver/recovery2/remote"
	"github.com/morrocker/recoveryserver/recovery2/tracker"
	"github.com/morrocker/recoveryserver/recovery2/tree"
)

type Recovery interface {
	GetTree() error
	GetFiles() error
	Priority(...Prty) string //subject to change
	Status(...State) string  //subject to change
	Progress()
	Output(...string) string

	Run()
	Stop()
	Cancel()
}

type recovery struct {
	data       Data
	resources  Resources
	status     State
	priority   Prty
	outputPath string
	tree       *tree.MetaTree

	rbs        remote.RBS
	tracker    *tracker.RecoveryTracker
	controller flow.Controller
}

type Data struct {
	Metafile   string
	Repository string
	User       string
	Version    int
	Deleted    bool
	Exclusions map[string]bool
	Server     string
	Key        string
}

type Resources struct {
	FileWorkers int
	MetaWorkers int
	MemBuffer   int
}

type RemotesData struct {
	address string
	magic   string
}

func New(d Data, r Resources, rd []remotesData) Recovery {
	var rbs remote.RBS
	if len(rd) <= 0 {
		log.Errorln(errors.New("recovery.New()", "no remotes set for the recovery"))
		return nil
	} else if len(rd) == 1 {
		rbs = remote.NewRBS(rd[0].address, rd[0].magic)
	} else {
		for i, data := range rd {
			if i == 0 {
				rbs = remote.NewRBS(rd[0].address, rd[0].magic)
				continue
			}
			rbs.SetBkp(data.address, data.magic)
		}
	}
	newRecovery := &recovery{
		data:       d,
		resources:  r,
		status:     Entry,
		priority:   MediumPr,
		rbs:        rbs,
		tracker:    tracker.New(),
		controller: flow.New(),
	}
	newRecovery.tracker.Gauges["membuff"].SetTotal(int64(newRecovery.resources.MemBuffer))

	return newRecovery
}

func (r *recovery) Run() {
	r.controller.Go()
}

func (r *recovery) Stop() {
	r.controller.Stop()
}

func (r *recovery) Cancel() {
	r.controller.Exit(int(Canceled))
}

func (r *recovery) Progress() {}

func (r *recovery) Status(s ...State) string {
	if len(s) != 0 {
		r.status = s[0]
	}
	return ParseState(r.status)
}

func (r *recovery) Priority(p ...Prty) string {
	if len(p) != 0 {
		r.priority = p[0]
	}
	return ParsePrty(r.priority)
}

func (r *recovery) Output(o ...string) string {
	if len(o) != 0 {
		r.outputPath = o[0]
	}
	return r.outputPath
}

func (r *recovery) GetTree() error {
	op := "recovery.GetTree()"

	data := tree.Data{
		RootId:     r.data.Metafile,
		Repository: r.data.Repository,
		Server:     r.data.Server,
		ClonerKey:  r.data.Key,
		Version:    r.data.Version,
		Deleted:    r.data.Deleted,
		Exclusions: r.data.Exclusions,
	}

	tt := tree.Throttling{
		Workers: r.resources.MetaWorkers,
	}

	newTree, err := tree.GetRecoveryTree(data, tt, r.tracker, r.controller)
	if err != nil {
		return errors.Extend(op, err)
	}
	r.tree = newTree
	return nil
}

func (r *recovery) GetFiles() error {
	op := "recovery.GetFiles()"

	data := files.Data{
		User:    r.data.User,
		Workers: r.resources.FileWorkers,
	}

	err := files.GetFiles(r.tree, r.outputPath, data, r.rbs, r.tracker, r.controller)
	if err != nil {
		return errors.Extend(op, err)
	}
	return nil
}
