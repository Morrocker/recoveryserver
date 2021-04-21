package files

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/clonercl/reposerver"
	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/recovery2/remote"
	"github.com/morrocker/recoveryserver/recovery2/tree"
	"golang.org/x/text/unicode/norm"
)

type Data struct {
	User    string
	Legacy  bool
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

func GetFiles(mt *tree.MetaTree, OutputPath string, data Data, rbs remote.RBS, tr *tracker.SuperTracker) error {
	log.Taskln("Starting files recovery")
	op := "recovery.getFiles()"

	fd := &fileData{
		Mt:         mt,
		OutputPath: mt.Mf.Name,
	}
	fl := &filesList{
		ToDo: make(map[string]*fileData),
	}

	sfc, sfWg := startSmallFilesWorkers(data, rbs)
	bfc, bfWg := startBigFilesWorkers(data, rbs)

	if err := os.MkdirAll(OutputPath, 0700); err != nil {
		return errors.New(op, errors.Extend(op, err))
	}

	log.Taskln("Filling files list")
	fillFilesList(OutputPath, fd, fl)

	fl.ToDo = filterDoneFiles(fl.ToDo)

	time.Sleep(5 * time.Second)

	preProcessFQ(fl, data, rbs)

	bigFiles, smallFiles := sortFiles(fl)

	var size int64
	var subFl []*fileData

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

	for _, fd := range bigFiles {
		recoverBigFile(fd, data.User, bfc, bfWg, rbs)
	}
	time.Sleep(time.Second)
	// close(bfc)
	bfWg.Wait()

	log.Noticeln("Files retrieval completed")
	for _, fd := range fl.ToDo {
		log.Info("File: %s", fd.OutputPath)
	}
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

func preProcessFQ(fl *filesList, data Data, rbs remote.RBS) error {
	log.Taskln("Pre-processing FQ (getting blocklists)")
	subHl := []string{}
	var size int64
	for hash, fd := range fl.ToDo {
		fileSize := fd.Mt.Mf.Size
		if size+fileSize > 10737418240 && size != 0 { // 10000 BLOCKS
			log.Info("going to get BlockLists")
			if err := getBlockLists(subHl, fl, data, rbs); err != nil {
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
		log.Info("going to get BlockLists")
		if err := getBlockLists(subHl, fl, data, rbs); err != nil {
			log.Errorln(errors.Extend("recovery.files.preProcessFQ()", err))
		}
	}
	return nil
}

func getBlockLists(hl []string, fl *filesList, data Data, rbs remote.RBS) error {
	contents, err := rbs.GetBlocksLists(hl, data.User)
	if err != nil {
		return errors.Extend("recovery.getBlockList()", err)
	}
	for i, content := range contents {
		if len(content) == 0 {
			delete(fl.ToDo, hl[i])
			continue
		}
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

func filterDoneFiles(fda map[string]*fileData) map[string]*fileData {
	log.Taskln("Filtering Done Files")
	subFDA := make(map[string]*fileData)
	for key, fd := range fda {
		size := fd.Mt.Mf.Size
		path := fd.OutputPath
		if fi, err := os.Stat(path); err == nil {
			if fi.Size() == int64(size) {
				// r.updateTrackerCurrent(int64(size))
				log.NoticeV("skipping file '%s'", path) // Temporal
				continue
			}
		}
		subFDA[key] = fd
	}
	return subFDA
}

// func (r *Recovery) fileWorker(fc chan *MetaTree, wg *sync.WaitGroup, bc chan bData) {
// 	op := "recovery.fileWorker()"
// Outer:
// 	for mt := range fc {
// 		if r.flowGate() {
// 			break
// 		}
// 		// Checking if file exists and is already done
// 		size := mt.mf.Size
// 		path := mt.path
// 		if fi, err := os.Stat(path); err == nil {
// 			if fi.Size() == int64(size) {
// 				r.updateTrackerCurrent(int64(size))
// 				r.log.NoticeV("skipping file '%s'", path)
// 				continue
// 			}
// 		}

// 		r.log.Info("Recovering file %s [%s]", mt.path, utils.B2H(int64(size)))
// 		// Getting file blocklist
// 		blist, err := r.RBS.GetBlocksList(mt.mf.Hash, r.Data.User)
// 		if err != nil {
// 			r.increaseErrors()
// 			r.log.ErrorlnV(errors.New(op, fmt.Sprintf("error could not create file '%s' because fileblock is unavailable", path)))
// 			r.tracker.ChangeCurr("completedSize", mt.mf.Size)
// 			continue
// 		}
// 		r.tracker.IncreaseCurr("blocks") // This is the fileblock

// 		// Creating recovery file
// 		op := "recovery.fileWriter()"
// 		f, err := os.Create(norm.NFC.String(path))
// 		if err != nil {
// 			r.increaseErrors()
// 			log.Errorln(errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", path, err)))
// 			r.tracker.ChangeCurr("completedSize", mt.mf.Size)
// 			continue
// 		}

// 		ret := make(chan returnBlock)
// 		blocksBuffer := make(map[int][]byte)
// 		blocks := blist.Blocks
// 		// Sending blocks to the blocks worker
// 		go func() {
// 			for i, hash := range blocks {
// 				if r.flowGate() {
// 					return
// 				}
// 				bc <- bData{id: i, hash: hash, ret: ret}
// 			}
// 		}()

// 		// Receiving blocks from blocksworkers and writting into file
// 		for x := 0; x < len(blocks); x++ {
// 			if r.flowGate() {
// 				f.Close()
// 				break Outer
// 			}
// 			content, ok := blocksBuffer[x]
// 			if ok {
// 				// log.Info("Block #%d is being written from buffer for %s", x, path[len(path)-20:])
// 				if _, err := f.Write(content); err != nil {
// 					r.increaseErrors()
// 					r.log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", blocks[x], path, err)))
// 					r.tracker.ChangeCurr("completedSize", len(content))
// 					continue Outer
// 				}
// 				r.tracker.ChangeCurr("completedSize", len(content))
// 				r.tracker.ChangeCurr("size", len(content))
// 				r.tracker.IncreaseCurr("blocks")
// 				r.tracker.ChangeCurr("blocksBuffer", -1)
// 				delete(blocksBuffer, x)
// 				continue
// 			}
// 			for d := range ret {
// 				if d.id == x {
// 					// log.Info("Block #%d is being written directly for %s", x, path[len(path)-20:])
// 					if _, err := f.Write(d.content); err != nil {
// 						r.increaseErrors()
// 						r.log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", blocks[x], path[len(path)-20:], err)))
// 						r.tracker.ChangeCurr("completedSize", len(d.content))
// 						continue Outer
// 					}
// 					r.tracker.ChangeCurr("completedSize", len(d.content))
// 					r.tracker.ChangeCurr("size", len(d.content))
// 					r.tracker.IncreaseCurr("blocks")
// 					break
// 				}
// 				r.checkBuffer()
// 				blocksBuffer[d.id] = d.content
// 				r.tracker.IncreaseCurr("blocksBuffer")
// 			}
// 		}
// 		r.tracker.IncreaseCurr("files")
// 		// log.Info("Finishing file %s", path[len(path)-20:])
// 		f.Close()
// 	}
// 	wg.Done()
// }

// func (r *Recovery) blockWorker(dc chan bData, wg2 *sync.WaitGroup) {
// 	for data := range dc {
// 		if r.flowGate() {
// 			break
// 		}
// 		b, err := r.RBS.GetBlock(data.hash, r.Data.User)
// 		if err != nil {
// 			var zeroedBuffer = make([]byte, 1024*1000)
// 			data.ret <- returnBlock{data.id, zeroedBuffer}
// 			continue
// 		}
// 		data.ret <- returnBlock{data.id, b}
// 	}
// 	wg2.Done()
// }

// func (r *Recovery) checkBuffer() {
// 	for {
// 		c, t, err := r.tracker.RawValues("blocksBuffer")
// 		if err != nil {
// 			log.Errorln(errors.New("recoveries.checkBuffer()", err))
// 		}
// 		if c < t {
// 			break
// 		}
// 		time.Sleep(time.Millisecond)
// 	}
// }
