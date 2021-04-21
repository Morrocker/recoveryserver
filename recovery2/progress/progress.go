package progress

// import (
// 	"github.com/morrocker/errors"
// 	"github.com/morrocker/log"
// 	tracker "github.com/morrocker/progress-tracker"
// )

// func initTracker() *tracker.SuperTracker {
// 	tr := tracker.New()
// 	tr.AddGauge("files", "files", 0)
// 	tr.AddGauge("size", "size", 0)
// 	tr.AddGauge("csize", "csize", 0)
// 	tr.AddGauge("errors", "errors", 0)
// 	tr.InitSpdRate("size", 40)
// 	tr.InitSpdRate("csize", 40)
// 	printFunction := func() {
// 		op := "recovery.printFunction()"
// 		fc, ft, _ := tr.RawValues("files")
// 		_, ef, _ := tr.RawValues("errors")
// 		sc, st, err := tr.Values("size")
// 		if err != nil {
// 			log.Errorln(errors.New(op, err))
// 		}
// 		rt, err := tr.ProgressRate("size")
// 		if err != nil {
// 			log.Errorln(errors.New(op, err))
// 		}
// 		eta, err := tr.ETA("completedSize")
// 		if err != nil {
// 			log.Errorln(errors.New(op, err))
// 		}
// 		// tr.State("default")
// 		// if r.Step == Metafiles {
// 		// 	log.Notice("[ Building Filetree ] Files: %d | Blocks: %d | Size: %s",
// 		// 		ft, bt, st)
// 		// } else if r.Step == Files {
// 		// 	log.Notice("[ Downloading Files ] Files: %d / %d | Size: %s / %s | Errors: %d [ %sps | %s ]",
// 		// 		fc, ft, sc, st, ef, rt, eta /*, bfc, bft*/)
// 		// }

// 	}
// 	tr.PrintFunc(printFunction)

// 	return tr
// }
