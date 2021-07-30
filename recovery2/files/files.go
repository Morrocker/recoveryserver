package files

import (
	"fmt"
	"os"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/clonercl/reposerver"
	"github.com/morrocker/broadcast"
	"github.com/morrocker/errors"
	"github.com/morrocker/flow"
	"github.com/morrocker/log"

	// tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/recovery2/remote"
	"github.com/morrocker/recoveryserver/recovery2/tracker"
	"github.com/morrocker/recoveryserver/recovery2/tree"
	"golang.org/x/text/unicode/norm"
)

type Data struct {
	User    string
	Workers int
}

// type filesList struct {
// 	ToDo map[string]*fileData
// }

type fileData struct {
	Mt         *tree.MetaTree
	OutputPath string
	blocksList []string
}

var zeroedBuffer = make([]byte, 1024*1000)

func GetFiles(mt *tree.MetaTree, OutputPath string, data Data, rbs remote.RBS, rt *tracker.RecoveryTracker, ctrl *flow.Controller) error {
	log.Taskln("Starting files recovery")
	op := "recovery.getFiles()"

	fd := &fileData{
		Mt:         mt,
		OutputPath: mt.Mf.Name,
	}
	filesList := make(map[string]*fileData)

	if err := os.MkdirAll(OutputPath, 0700); err != nil {
		return errors.New(op, errors.Extend(op, err))
	}

	log.Taskln("Filling files list")
	log.Info("Starting Size:%d", len(filesList))
	fillFilesList(OutputPath, fd, filesList)
	log.Info("Filled list Size:%d", len(filesList))
	filterDoneFiles(filesList, rt)
	log.Info("Filtered list Size:%d", len(filesList))

	time.Sleep(5 * time.Second)

	fetchBlockLists(filesList, data, rbs, rt, ctrl)

	fetchFiles(filesList, data, rbs, rt, ctrl)

	// // rt.StopAutoPrint()
	// // rt.StopAutoMeasure("size")
	log.Noticeln("Files retrieval completed")
	return nil
}

func fillFilesList(output string, fd *fileData, fl map[string]*fileData) {
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
			// if r.flowGate() {
			// 	break
			// }
			newFD := &fileData{
				Mt:         child,
				OutputPath: p,
			}
			fillFilesList(p, newFD, fl)
		}
		return
	}
	fd.OutputPath = p
	fl[mf.Hash] = fd
}

func filterDoneFiles(fda map[string]*fileData, rt *tracker.RecoveryTracker) {
	log.Taskln("Filtering Done Files")
	delList := []string{}
	for key, fd := range fda {
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
}

func fetchBlockLists(fl map[string]*fileData, data Data, rbs remote.RBS, rt *tracker.RecoveryTracker, ctrl *flow.Controller) {
	log.Taskln("Pre-processing FQ (getting blocklists)")

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
	for hash := range fl {
		if len(fl[hash].blocksList) <= 0 {
			delete(fl, hash)
		}
	}
}

func fetchFiles(fl map[string]*fileData, data Data, rbs remote.RBS, rt *tracker.RecoveryTracker, ctrl *flow.Controller) {
	orderedFiles := []*fileData{}
	for _, fd := range fl {
		orderedFiles = append(orderedFiles, fd)
	}
	sort.Slice(orderedFiles, func(i, j int) bool { return orderedFiles[i].Mt.Mf.Size > orderedFiles[j].Mt.Mf.Size })

	wg := &sync.WaitGroup{}
	wg2 := &sync.WaitGroup{}
	fdc := make(chan *fileData)
	bdc := make(chan blockData)
	// bufferMap := make(map[string]map[string][]byte)
	var bufferMap2 *sync.Map = &sync.Map{}
	bc := broadcast.New()
	for x := 0; x < data.Workers; x++ {
		go fileWorker(fdc, bdc, data.User, bufferMap2, bc.Listen(), wg, rbs, rt, ctrl)
	}
	for x := 0; x < data.Workers*2; x++ {
		go filesBlockWorker(bdc, bufferMap2, bc, wg2, rbs, rt, ctrl)
	}

	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			bc.Broadcast()
		}
	}()

	for _, fd := range orderedFiles {
		// log.Info("Sending file %s to recovery", fd.OutputPath)
		var fileBufferMap *sync.Map = &sync.Map{}
		bufferMap2.Store(fd.Mt.Mf.Hash, fileBufferMap)
		fdc <- fd
	}
	time.Sleep(5 * time.Second)
	close(fdc)
	wg.Wait()
	close(bdc)
	wg2.Wait()
}

// func sort(arr []*fileData, newFD *fileData) []*fileData {
// 	fsz := newFD.Mt.Mf.Size
// 	size := len(arr)
// 	pre := arr[:size/2]
// 	post := arr[size/2:]
// 	if fsz <= post[len(post)-1].Mt.Mf.Size {
// 		return append(arr, newFD)
// 	} else if fsz >= pre[0].Mt.Mf.Size {
// 		newArr := []*fileData{newFD}
// 		return append(newArr, arr...)
// 	} else if fsz > post[0].Mt.Mf.Size {
// 		pre = splitInsertSort(pre, newFD)
// 	} else {
// 		post = splitInsertSort(post, newFD)
// 	}
// 	return append(pre, post...)
// }
