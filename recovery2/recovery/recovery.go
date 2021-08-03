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
	Progress() ProgressData

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

	logger     log.Logger
	rbs        remote.RBS
	tracker    *tracker.RecoveryTracker
	controller flow.Controller
}

type ProgressData struct {
	CurrFiles   int64
	TotFiles    int64
	CurrMetaf   int64
	TotMetaf    int64
	CurrSize    int64
	TotSize     int64
	CurrTotSize int64
	TotTotSize  int64
	Errors      int64
	FileErrors  int64
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
	Address string
	Magic   string
}

func New(d Data, r Resources, rd []RemotesData) Recovery {
	var rbs remote.RBS
	if len(rd) <= 0 {
		log.Errorln(errors.New("recovery.New()", "no remotes set for the recovery"))
		return nil
	} else if len(rd) == 1 {
		rbs = remote.NewRBS(rd[0].Address, rd[0].Magic)
	} else {
		for i, data := range rd {
			if i == 0 {
				rbs = remote.NewRBS(rd[0].Address, rd[0].Magic)
				continue
			}
			rbs.SetBkp(data.Address, data.Magic)
		}
	}
	lg := log.New()
	lg.SetScope(true, true, true, true, false, false)
	newRecovery := &recovery{
		data:       d,
		resources:  r,
		status:     Entry,
		priority:   MediumPr,
		rbs:        rbs,
		tracker:    tracker.New(),
		controller: flow.New(),
		logger:     lg,
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

func (r *recovery) Progress() ProgressData {
	cf, tf := r.tracker.Gauges["files"].RawValues()
	cmf, tmf := r.tracker.Gauges["metafiles"].RawValues()
	csz, tsz := r.tracker.Gauges["size"].RawValues()
	ctsz, ttsz := r.tracker.Gauges["totalSize"].RawValues()
	errs := r.tracker.Counters["errors"].RawValue()
	fErrs := r.tracker.Counters["fileErrors"].RawValue()
	pd := ProgressData{
		CurrFiles: cf, TotFiles: tf,
		CurrMetaf: cmf, TotMetaf: tmf,
		CurrSize: csz, TotSize: tsz,
		CurrTotSize: ctsz, TotTotSize: ttsz,
		Errors:     errs,
		FileErrors: fErrs,
	}
	return pd
}

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
		Workers:  r.resources.MetaWorkers,
		BuffSize: r.resources.MemBuffer * 100,
	}

	newTree, err := tree.GetRecoveryTree(data, tt, r.tracker, r.controller, r.logger)
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
