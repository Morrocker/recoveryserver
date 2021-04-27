package track

import (
	"github.com/morrocker/log"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/utils"
)

func New() *tracker.SuperTracker {
	tr := tracker.New()
	tr.AddGauge("buffer", 10)
	tr.AddGauge("files", 0)
	tr.AddGauge("size", 0)
	tr.AddGauge("totalsize", 0)
	tr.AddGauge("errors", 0)
	tr.InitSpdRate("size", 40)
	tr.UnitsFunc("totalsize", utils.B2H)
	fn := func() {
		fe, et, _ := tr.RawValues("errors")
		cf, tf, _ := tr.RawValues("files")
		cs, ts, _ := tr.Values("totalsize")
		rt, _ := tr.TrueProgressRate("size")
		eta, _ := tr.ETA("size")
		log.Notice("Files: %d/%d | Size: %s/%s\t\t | Error Files: %d | Error Total:%d [ %s | ETA: %d ]", cf, tf, cs, ts, fe, et, rt, eta)
	}
	tr.PrintFunc(fn)

	return tr
}

func AlreadyDone(size int64, tr *tracker.SuperTracker) {
	tr.ChangeCurr("files", 1)
	tr.ChangeTotal("files", 1)
	tr.ChangeCurr("size", size)
	tr.ChangeTotal("size", size)
	tr.ChangeCurr("totalsize", size)
	tr.ChangeTotal("totalsize", size)
}

func FailedBlocklist(size int64, tr *tracker.SuperTracker) {
	tr.ChangeTotal("files", 1)
	tr.ChangeTotal("totalsize", size)
	tr.ChangeCurr("errors", 1)
}

func AddFile(size int64, tr *tracker.SuperTracker) {
	tr.ChangeTotal("files", 1)
	tr.ChangeTotal("size", size)
	tr.ChangeTotal("totalsize", size)
}

func CompleteFile(size int64, tr *tracker.SuperTracker) {
	tr.ChangeCurr("files", 1)
	tr.ChangeCurr("size", size)
	tr.ChangeCurr("totalsize", size)
}

func FailedFiles(n int, tr *tracker.SuperTracker) {
	tr.ChangeCurr("errors", n)
	tr.ChangeTotal("errors", n)
}
