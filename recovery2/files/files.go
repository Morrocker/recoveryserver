package files

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/clonercl/reposerver"
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

type filesList struct {
	ToDo map[string]*fileData
}

type fileData struct {
	Mt         *tree.MetaTree
	OutputPath string
	blocksList []string
}

var zeroedBuffer = make([]byte, 1024*1000)

func GetFiles(mt *tree.MetaTree, OutputPath string, data Data, rbs remote.RBS, tr *tracker.SuperTracker, ctrl *flow.Controller) error {
	log.Taskln("Starting files recovery")
	op := "recovery.getFiles()"

	fd := &fileData{
		Mt:         mt,
		OutputPath: mt.Mf.Name,
	}
	fl := &filesList{
		ToDo: make(map[string]*fileData),
	}

	sfc, sfWg := startSmallFilesWorkers(data, rbs, tr, ctrl)
	bfc, bdc, bfWg, bdWg := startBigFilesWorkers(data, rbs, tr, ctrl)

	if err := os.MkdirAll(OutputPath, 0700); err != nil {
		return errors.New(op, errors.Extend(op, err))
	}

	log.Taskln("Filling files list")
	fillFilesList(OutputPath, fd, fl)

	fl.ToDo = filterDoneFiles(fl.ToDo, tr)

	time.Sleep(5 * time.Second)

	preProcessFQ(fl, data, rbs, tr)

	bigFiles, smallFiles := sortFiles(fl)

	var size int64
	var subFl []*fileData

	tr.StartAutoPrint(6 * time.Second)
	tr.StartAutoMeasure("size", 20*time.Second)
	log.Notice("Sending small files lists. #%d", len(smallFiles))
	for _, fd := range smallFiles {
		fileSize := fd.Mt.Mf.Size
		if size+fileSize > 104857600 && size != 0 { // 10000 BLOCKS
			sfc <- subFl
			size = 0
			subFl = []*fileData{}
		}
		subFl = append(subFl, fd)
		size += fileSize
	}

	if len(subFl) > 0 {
		sfc <- subFl
	}
	time.Sleep(time.Second)
	close(sfc)
	sfWg.Wait()

	log.Notice("Sending big files lists. #%d", len(bigFiles))
	for _, fd := range bigFiles {
		bfc <- fd
	}
	// log.Info("Finished recovering big files")
	time.Sleep(time.Second)
	close(bfc)
	bfWg.Wait()
	close(bdc)
	bdWg.Wait()

	tr.StopAutoPrint()
	tr.StopAutoMeasure("size")
	log.Noticeln("Files retrieval completed")
	return nil
}

func fillFilesList(output string, fd *fileData, fl *filesList) {
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
	fl.ToDo[mf.Hash] = fd
}

func filterDoneFiles(fda map[string]*fileData, ot *tracker.OmniTracker) map[string]*fileData {
	log.Taskln("Filtering Done Files")
	subFDA := make(map[string]*fileData)
	for key, fd := range fda {
		size := fd.Mt.Mf.Size
		path := fd.OutputPath
		if fi, err := os.Stat(path); err == nil {
			if fi.Size() == int64(size) {
				// r.updateTrackerCurrent(int64(size))
				log.NoticeV("skipping done file '%s'", path) // Temporal
				ot.AlreadyDone(size, tr)
				continue
			}
		}
		subFDA[key] = fd
	}
	return subFDA
}

func preProcessFQ(fl *filesList, data Data, rbs remote.RBS, tr *tracker.SuperTracker) error {
	log.Taskln("Pre-processing FQ (getting blocklists)")
	subHl := []string{}
	var size int64
	for hash, fd := range fl.ToDo {
		fileSize := fd.Mt.Mf.Size
		if size+fileSize > 10737418240 && size != 0 { // 10000 BLOCKS
			if err := getBlockLists(subHl, fl, data, rbs, tr); err != nil {
				log.Errorln(errors.Extend("recovery.files.preProcessFQ()", err))
			}
			size = 0
			subHl = []string{}
		}
		subHl = append(subHl, hash)
		size += fileSize
		// log.Info("Size: #%d. subHl len:%d", size, len(subHl))
	}

	if len(subHl) != 0 {
		if err := getBlockLists(subHl, fl, data, rbs, tr); err != nil {
			log.Errorln(errors.Extend("recovery.files.preProcessFQ()", err))
		}
	}
	return nil
}

func getBlockLists(hl []string, fl *filesList, data Data, rbs remote.RBS, tr *tracker.SuperTracker) error {
	contents, err := rbs.GetBlocksLists(hl, data.User)
	if err != nil {
		return errors.Extend("recovery.getBlockList()", err)
	}
	for i, content := range contents {
		size := fl.ToDo[hl[i]].Mt.Mf.Size
		if len(content) == 0 {
			track.FailedBlocklist(size, tr)
			delete(fl.ToDo, hl[i])
			continue
		}
		track.AddFile(size, tr)
		fl.ToDo[hl[i]].blocksList = content
	}

	return nil
}

func sortFiles(fl *filesList) (bigFiles []*fileData, smallFiles []*fileData) {
	log.Taskln("Sorting files")
	for _, fd := range fl.ToDo {
		if fd.Mt.Mf.Size > 104857600 {
			bigFiles = append(bigFiles, fd)
		} else {
			smallFiles = append(smallFiles, fd)
		}
	}
	log.Task("Small files: %d, Big files: %d", len(smallFiles), len(bigFiles))
	return
}
