package files

import (
	"fmt"
	"os"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/clonercl/reposerver"
	"github.com/morrocker/errors"
	"github.com/morrocker/flow"
	"github.com/morrocker/log"

	"github.com/morrocker/recoveryserver/recovery2/remote"
	"github.com/morrocker/recoveryserver/recovery2/tracker"
	"github.com/morrocker/recoveryserver/recovery2/tree"
	"golang.org/x/text/unicode/norm"
)

type Data struct {
	User    string
	Workers int
}

type fileData struct {
	Mt         *tree.MetaTree
	OutputPath string
	blocksList []string
}

var zeroedBuffer = make([]byte, 1024*1000)

func GetFiles(mt *tree.MetaTree, OutputPath string, data Data, rbs remote.RBS, rt *tracker.RecoveryTracker, ctrl flow.Controller) error {
	log.Taskln("Starting files recovery")
	op := "recovery.getFiles()"

	// log.Info("GET FILES DATA")
	// spew.Dump(mt)
	// spew.Dump(OutputPath)
	// spew.Dump(data)
	// spew.Dump(rbs)
	// spew.Dump(rt)
	// spew.Dump(ctrl)

	fd := &fileData{
		Mt:         mt,
		OutputPath: mt.Mf.Name,
	}
	filesList := make(map[string]*fileData)

	_, t := rt.Gauges["membuff"].RawValues()
	if t == 0 {
		return errors.New(op, "memory buffer parameter not set")
	}

	if err := os.MkdirAll(OutputPath, 0700); err != nil {
		return errors.New(op, errors.Extend(op, err))
	}

	log.Taskln("Filling files list")

	fillFilesList(OutputPath, fd, filesList, ctrl)
	if ctrl.Checkpoint() != 0 {
		return nil
	}

	filterDoneFiles(filesList, rt, ctrl)
	if ctrl.Checkpoint() != 0 {
		return nil
	}

	fetchBlockLists(filesList, data, rbs, rt, ctrl)
	if ctrl.Checkpoint() != 0 {
		return nil
	}

	fetchFiles(filesList, data, rbs, rt, ctrl)

	// // rt.StopAutoPrint()
	// // rt.StopAutoMeasure("size")
	log.Noticeln("Files retrieval completed")
	return nil
}

func fillFilesList(output string, fd *fileData, fl map[string]*fileData, ctrl flow.Controller) {
	mf := fd.Mt.Mf
	p := path.Join(output, mf.Name)
	if mf.Type == reposerver.FolderType {
		if mf.Parent == "" {
			p = output
		} else {
			if err := os.MkdirAll(norm.NFC.String(p), 0700); err != nil {
				panic(fmt.Sprintf("could not create path '%s': %v\n", p, err))
			}
		}
		for _, child := range fd.Mt.Children {
			if ctrl.Checkpoint() != 0 {
				break
			}
			newFD := &fileData{
				Mt:         child,
				OutputPath: p,
			}
			fillFilesList(p, newFD, fl, ctrl)
		}
		return
	}
	fd.OutputPath = p
	fl[mf.Hash] = fd
}

func filterDoneFiles(fda map[string]*fileData, rt *tracker.RecoveryTracker, ctrl flow.Controller) {
	log.Taskln("Filtering Done Files")
	delList := []string{}
	for key, fd := range fda {
		if ctrl.Checkpoint() != 0 {
			return
		}
		size := fd.Mt.Mf.Size
		path := fd.OutputPath
		if fi, err := os.Stat(path); err == nil {
			if fi.Size() == int64(size) {
				// r.updateTrackerCurrent(int64(size))
				log.NoticeV("skipping done file '%s'", path) // Temporal
				rt.AlreadyDone(size)
				delList = append(delList, key)
				continue
			}
		}
	}
	for _, key := range delList {
		delete(fda, key)
	}
	time.Sleep(1 * time.Second)
}

func fetchBlockLists(fl map[string]*fileData, data Data, rbs remote.RBS, rt *tracker.RecoveryTracker, ctrl flow.Controller) {
	log.Taskln("Fetching blocklists")

	wg := &sync.WaitGroup{}
	fdc := make(chan string)
	wg.Add(data.Workers)
	for x := 0; x < data.Workers; x++ {
		go blockListWorker(fdc, fl, data.User, wg, rbs, rt, ctrl)
	}

	for hash := range fl {
		if ctrl.Checkpoint() != 0 {
			break
		}
		fdc <- hash
	}
	time.Sleep(5 * time.Second)
	close(fdc)
	wg.Wait()
	for hash := range fl {
		if len(fl[hash].blocksList) <= 0 {
			delete(fl, hash)
		}
	}
}

func fetchFiles(fl map[string]*fileData, data Data, rbs remote.RBS, rt *tracker.RecoveryTracker, ctrl flow.Controller) {
	orderedFiles := []*fileData{}
	for _, fd := range fl {
		orderedFiles = append(orderedFiles, fd)
	}
	sort.Slice(orderedFiles, func(i, j int) bool { return orderedFiles[i].Mt.Mf.Size > orderedFiles[j].Mt.Mf.Size })

	wg := &sync.WaitGroup{}
	wg2 := &sync.WaitGroup{}
	fdc := make(chan *fileData)
	bdc := make(chan blockData)
	var bufferMap2 *sync.Map = &sync.Map{}
	for x := 0; x < data.Workers; x++ {
		go fileWorker(fdc, bdc, data.User, bufferMap2, wg, rbs, rt, ctrl)
	}
	for x := 0; x < data.Workers; x++ {
		go filesBlockWorker(bdc, bufferMap2, wg2, rbs, rt, ctrl)
	}

	for _, fd := range orderedFiles {
		var fileBufferMap *sync.Map = &sync.Map{}
		bufferMap2.Store(fd.Mt.Mf.Hash, fileBufferMap)
		fdc <- fd
	}
	time.Sleep(5 * time.Second)
	log.Taskln("Closing fileWorkers Channels")
	close(fdc)
	log.Taskln("Waiting for fileWorkers to finish")
	wg.Wait()
	log.Taskln("Closing blockWorkers Channels")
	close(bdc)
	log.Taskln("Waiting for blockWorkers to finish")
	wg2.Wait()
	log.Taskln("Ending Recovery")
}
