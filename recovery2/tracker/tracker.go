package tracker

import (
	"github.com/morrocker/tracker"
	"github.com/morrocker/utils"
)

type RecoveryTracker struct {
	Gauges   map[string]tracker.Gauge
	Counters map[string]tracker.Counter
}

func New() *RecoveryTracker {
	r := &RecoveryTracker{}
	r.Gauges = make(map[string]tracker.Gauge)
	r.Counters = make(map[string]tracker.Counter)
	r.Gauges["files"] = tracker.NewGauge()
	r.Gauges["metafiles"] = tracker.NewGauge()
	r.Gauges["size"] = tracker.NewGauge()
	r.Gauges["size"].UnitsFunc(utils.B2H)
	r.Gauges["totalsize"] = tracker.NewGauge()
	r.Gauges["totalsize"].UnitsFunc(utils.B2H)
	r.Counters["errors"] = tracker.NewCounter()
	r.Counters["fileErrors"] = tracker.NewCounter()
	return r

	// tr.AddGauge("buffer", 10)
	// tr.AddGauge("files", 0)
	// tr.AddGauge("metafiles", 0)
	// tr.AddGauge("size", 0)
	// tr.AddGauge("totalsize", 0)
	// tr.AddGauge("errors", 0)
	// tr.InitSpdRate("size", 40)
	// tr.UnitsFunc("totalsize", utils.B2H)
	// tr.UnitsFunc("size", utils.B2H)
	// fn := func() {
	// 	fe, et, _ := tr.RawValues("errors")
	// 	cf, tf, _ := tr.RawValues("files")
	// 	cs, ts, _ := tr.Values("totalsize")
	// 	rt, _ := tr.TrueProgressRate("size")
	// 	eta, _ := tr.ETA("size")
	// 	log.Notice("Files: %d/%d | Size: %s/%s | Error Files/Total: %d/%d \t\t[ %sps | ETA: %s ]", cf, tf, cs, ts, fe, et, rt, eta)
	// }
	// tr.PrintFunc(fn)

	// return tr
}

func (r *RecoveryTracker) AlreadyDone(size int64) {
	r.Gauges["files"].Total(1)
	r.Gauges["files"].Current(1)
	r.Gauges["size"].Total(size)
	r.Gauges["size"].Current(size)
	r.Gauges["totalsize"].Total(size)
	r.Gauges["totalsize"].Current(size)
}

func (r *RecoveryTracker) FailedBlocklist(size int64) {
	r.Gauges["files"].Total(1)
	r.Gauges["totalsize"].Total(size)
	r.Counters["errors"].Current(1)
}

func (r *RecoveryTracker) AddFile(size int64) {
	r.Gauges["files"].Total(1)
	r.Gauges["size"].Total(size)
	r.Gauges["totalsize"].Total(size)
}

func (r *RecoveryTracker) CompleteFile(size int64) {
	r.Gauges["files"].Current(1)
	r.Gauges["size"].Current(size)
	r.Gauges["totalsize"].Current(size)
}

func (r *RecoveryTracker) FailedFiles(n int) {
	r.Counters["errors"].Current(int64(n))
	r.Counters["fileErrors"].Current(int64(n))
}
