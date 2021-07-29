package files

import (
	"fmt"
	"os"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/morrocker/broadcast"
	"github.com/morrocker/errors"
	"github.com/morrocker/flow"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/recovery2/remote"
	"github.com/morrocker/utils"
	"golang.org/x/text/unicode/norm"

	// "github.com/morrocker/tracker"

	"github.com/morrocker/recoveryserver/recovery2/tracker"
)

// func startSmallFilesWorkers(data Data, rbs remote.RBS, tr *tracker.RecoveryTracker, ctrl *flow.Controller) (chan []*fileData, *sync.WaitGroup) {
// 	// log.Task("Starting %d small files workers", data.Workers)
// 	wg := &sync.WaitGroup{}
// 	fdc := make(chan []*fileData)
// 	wg.Add(data.Workers)
// 	for x := 0; x < data.Workers; x++ {
// 		go smallFilesWorker(fdc, data.User, wg, rbs, tr, ctrl)
// 	}
// 	return fdc, wg
// }

// func smallFilesWorker(fc chan []*fileData, user string, wg *sync.WaitGroup, rbs remote.RBS, rt *tracker.RecoveryTracker, ctrl *flow.Controller) {
// 	op := "recovery.smallFilesWorker()"
// 	// log.Taskln("Starting small files workers")
// Outer:
// 	for fda := range fc {
// 		if ctrl.Checkpoint() != 0 {
// 			break
// 		}
// 		positionArray := []*fileData{}
// 		blocksArray := []string{}
// 		for _, fd := range fda {
// 			if ctrl.Checkpoint() != 0 {
// 				break
// 			}
// 			// log.Info("Recovering file %s [%s]", fd.OutputPath, utils.B2H(int64(fd.Mt.Mf.Size)))
// 			blocksArray = append(blocksArray, fd.blocksList...)
// 			for x := 0; x < len(fd.blocksList); x++ {
// 				positionArray = append(positionArray, fd)
// 			}
// 		}
// 		// log.Infoln("Sending small files hashs")
// 		bytesArrays, err := rbs.GetBlocks(blocksArray, user)
// 		if err != nil {
// 			log.Errorln(errors.Extend(op, err))
// 			rt.FailedFiles(len(bytesArrays))
// 			continue
// 		}
// 		bytesArray := []byte{}

// 		// log.Infoln("Writting small files hashs")
// 		for i, content := range bytesArrays {
// 			if ctrl.Checkpoint() != 0 {
// 				break Outer
// 			}
// 			if i == 0 {
// 				bytesArray = appendContent(bytesArray, content, rt)
// 			} else if positionArray[i-1].Mt.Mf.Hash == positionArray[i].Mt.Mf.Hash {
// 				bytesArray = appendContent(bytesArray, content, rt)
// 			} else {
// 				if err := writeSmallFile(positionArray[i-1], bytesArray); err != nil {
// 					log.Errorln(errors.Extend(op, err))
// 					rt.FailedFiles(1)
// 				}
// 				rt.CompleteFile(int64(len(bytesArray)))
// 				bytesArray = []byte{}
// 				bytesArray = appendContent(bytesArray, content, rt)
// 			}
// 		}

// 		if err := writeSmallFile(positionArray[len(positionArray)-1], bytesArray); err != nil {
// 			log.Errorln(errors.Extend(op, err))
// 			rt.FailedFiles(1)
// 		} // writting the last file
// 		rt.CompleteFile(int64(len(bytesArray)))
// 		// log.Infoln("Completed Writting small files hashs")
// 	}
// 	wg.Done()
// }

// func appendContent(content []byte, newContent []byte, rt *tracker.RecoveryTracker) []byte {
// 	if newContent != nil {
// 		content = append(content, newContent...)
// 	} else {
// 		rt.Counters["errors"].Current(1)
// 		rt.Counters["fileErrors"].Current(1)
// 		content = append(content, zeroedBuffer...)
// 	}
// 	return content
// }

// // Creating recovery file
// func writeSmallFile(fd *fileData, content []byte) error {
// 	op := "recovery.writeFile()"
// 	// log.Task("Writting small file %s [%s]", fd.OutputPath, utils.B2H(int64(fd.Mt.Mf.Size)))
// 	f, err := os.Create(norm.NFC.String(fd.OutputPath))
// 	if err != nil {
// 		return errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", fd.OutputPath, err))
// 	}
// 	defer f.Close()
// 	if _, err := f.Write(content); err != nil {
// 		return errors.New(op, fmt.Sprintf("error could not write content for file '%s': %v\n", fd.OutputPath, err))
// 	}
// 	return nil
// }

// func startBigFilesWorkers(data Data, rbs remote.RBS, rt *tracker.RecoveryTracker, ctrl *flow.Controller) (chan *fileData, chan bigData, *sync.WaitGroup, *sync.WaitGroup) {
// 	log.Task("Starting %d big files workers", data.Workers)
// 	wg := &sync.WaitGroup{}
// 	wg2 := &sync.WaitGroup{}
// 	bfc := make(chan *fileData)
// 	bdc := make(chan bigData)
// 	wg.Add(data.Workers)
// 	for x := 0; x < data.Workers; x++ {
// 		go recoverBigFile(bfc, bdc, data.User, wg, rbs, rt, ctrl)
// 	}
// 	wg2.Add(data.Workers)
// 	for x := 0; x < data.Workers; x++ {
// 		go blocksWorker(bdc, data.User, wg2, rbs, rt, ctrl)
// 	}
// 	return bfc, bdc, wg, wg2
// }

// type returnBlock struct {
// 	idx     int
// 	content []byte
// 	err     error
// }

// type bigData struct {
// 	idx   int
// 	hashs []string
// 	ret   chan returnBlock
// }

// // func bigFilesWorker(fc chan *fileData, user string, wg *sync.WaitGroup, rbs remote.RBS /*, bc chan bData*/) {
// func recoverBigFile(bfc chan *fileData, fdc chan bigData, user string, wg *sync.WaitGroup, rbs remote.RBS, rt *tracker.RecoveryTracker, ctrl *flow.Controller) {
// 	op := "files.recoverBigFile()"

// Outer:
// 	for fd := range bfc {
// 		if ctrl.Checkpoint() != 0 {
// 			break
// 		}
// 		// log.Info("Recovering file %s [%s]", fd.OutputPath, utils.B2H(int64(fd.Mt.Mf.Size)))

// 		// Creating recovery file
// 		f, err := os.Create(norm.NFC.String(fd.OutputPath))
// 		if err != nil {
// 			rt.FailedFiles(1)
// 			log.Errorln(errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", fd.OutputPath, err)))
// 		}

// 		ret := make(chan returnBlock)
// 		blocksBuffer := make(map[int][]byte)
// 		// Sending blocks to the blocks worker
// 		idx := 0
// 		go func() {
// 			hashs := []string{}
// 			for i, hash := range fd.blocksList {
// 				if ctrl.Checkpoint() != 0 {
// 					break
// 				}
// 				if i%100 == 0 && i != 0 {
// 					fdc <- bigData{idx: idx, hashs: hashs, ret: ret}
// 					hashs = []string{}
// 					hashs = append(hashs, hash)
// 					idx++
// 				} else {
// 					hashs = append(hashs, hash)
// 				}
// 				// if r.flowGate() {
// 				// 	return
// 				// }
// 			}
// 			if len(hashs) != 0 {
// 				fdc <- bigData{idx: idx, hashs: hashs, ret: ret}
// 			}
// 		}()

// 		// Receiving blocks from blocksworkers and writting into file
// 		for x := 0; x < idx+1; x++ {
// 			if ctrl.Checkpoint() != 0 {
// 				f.Close()
// 				break Outer
// 			}
// 			content, ok := blocksBuffer[x]
// 			if ok {
// 				// log.Info("Block #%d is being written from buffer for %s", x, utils.Trimmer(fd.OutputPath, 0, 50))
// 				if _, err := f.Write(content); err != nil {
// 					rt.FailedFiles(1)
// 					log.Errorln(errors.Extend(op, err))
// 				}
// 				rt.Gauges["buffer"].Current(-1)
// 				delete(blocksBuffer, x)
// 				continue
// 			}
// 			for data := range ret {
// 				if ctrl.Checkpoint() != 0 {
// 					f.Close()
// 					break Outer
// 				}
// 				if data.err != nil {
// 					f.Close()
// 					log.Errorln(errors.Extend(op, err))
// 				}
// 				if data.idx == x {
// 					// log.Info("Block #%d is being written directly for %s", x, utils.Trimmer(fd.OutputPath, 0, 50))
// 					if _, err := f.Write(data.content); err != nil {
// 						rt.FailedFiles(1)
// 						log.Errorln(errors.Extend(op, err))
// 					}
// 					break
// 				}
// 				checkBuffer(rt)
// 				blocksBuffer[data.idx] = data.content
// 				rt.Gauges["blocksBuffer"].Current(1)
// 			}
// 		}
// 		rt.CompleteFile(fd.Mt.Mf.Size)
// 		f.Close()
// 	}
// 	wg.Done()
// }

// func blocksWorker(bfc chan bigData, user string, wg *sync.WaitGroup, rbs remote.RBS, rt *tracker.RecoveryTracker, ctrl *flow.Controller) {
// 	op := "recovery.files.blocksWorker()"
// Outer:
// 	for data := range bfc {
// 		if ctrl.Checkpoint() != 0 {
// 			break Outer
// 		}
// 		ret := returnBlock{idx: data.idx}
// 		bytesArray, err := rbs.GetBlocks(data.hashs, user)
// 		if err != nil {
// 			ret.err = errors.Extend(op, err)
// 			data.ret <- ret
// 			continue Outer
// 		}
// 		for _, bytes := range bytesArray {
// 			if ctrl.Checkpoint() != 0 {
// 				break Outer
// 			}
// 			if bytes == nil {
// 				ret.content = append(ret.content, zeroedBuffer...)
// 				rt.Gauges["totalsize"].Current(int64(len(bytes)))
// 				rt.Gauges["errors"].Current(1)
// 			} else {
// 				rt.Gauges["size"].Current(int64(len(bytes)))
// 				rt.Gauges["totalsize"].Current(int64(len(bytes)))
// 				ret.content = append(ret.content, bytes...)
// 			}
// 		}
// 		data.ret <- ret
// 	}
// 	wg.Done()
// }

// func checkBuffer(rt *tracker.RecoveryTracker) {
// 	for {
// 		c, t := rt.Gauges["buffer"].RawValues()
// 		if c < t {
// 			break
// 		}
// 		time.Sleep(time.Millisecond)
// 	}
// }

func blockListWorker(
	blockChan chan string,
	destMap map[string]*fileData, user string,
	wg *sync.WaitGroup,
	rbs remote.RBS,
	rt *tracker.RecoveryTracker,
	ctrl *flow.Controller) {
	for block := range blockChan {
		if ctrl.Checkpoint() != 0 {
			break
		}
		blockList, err := rbs.GetBlocksList(block, user)
		if err != nil && len(blockList) == 0 {
			log.Errorln(err)
			rt.Counters["fileErrors"].Current(1)
			continue
		}
		destMap[block].blocksList = blockList
		rt.Gauges["files"].Total(1)
	}
}

func fileWorker(
	fdc chan *fileData, bdc chan blockData,
	bufferMap map[string]map[string][]byte, user string,
	ls *broadcast.Listener,
	wg *sync.WaitGroup,
	rbs remote.RBS,
	rt *tracker.RecoveryTracker,
	ctrl *flow.Controller) {
	op := "recovery.fileWorker()"

	for fd := range fdc {
		wg.Add(1)
		log.Info("Recovering file %s\t[%s]", fd.OutputPath, utils.B2H(fd.Mt.Mf.Size))
		bufferMap[fd.Mt.Mf.Hash] = make(map[string][]byte)
		log.Infoln("Buffermap")
		// spew.Dump(bufferMap)
		go func() {
			for _, block := range fd.blocksList {
				newBlockData := blockData{
					user:     user,
					hash:     block,
					fileHash: fd.Mt.Mf.Hash,
				}
				bdc <- newBlockData
			}
		}()
		f, err := os.Create(norm.NFC.String(fd.OutputPath))
		if err != nil {
			log.Errorln(errors.New(op, err))
		}
		for _, block := range fd.blocksList {
			for {
				val, ok := bufferMap[fd.Mt.Mf.Hash][block]
				if ok {
					if _, err := f.Write(val); err != nil {
						log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for file '%s': %v\n", fd.OutputPath, err)))
					}
					delete(bufferMap, fd.Mt.Mf.Hash)
					break
				}
				<-ls.C
			}
		}
		f.Close()
		delete(bufferMap, fd.Mt.Mf.Hash)
		wg.Done()
	}
}

type blockData struct {
	user     string
	hash     string
	fileHash string
}

func filesBlockWorker(
	bdc chan blockData,
	bufferMap map[string]map[string][]byte,
	bc *broadcast.Broadcaster,
	wg *sync.WaitGroup,
	rbs remote.RBS,
	rt *tracker.RecoveryTracker,
	ctrl *flow.Controller) {
	op := "files.filesBlockWorker()"

	for bd := range bdc {
		wg.Add(1)
		bytes, err := rbs.GetBlock(bd.hash, bd.user)
		if err != nil {
			log.Errorln(errors.Extend(op, err))
			bytes = zeroedBuffer
		}
		spew.Dump(bufferMap)
		bufferMap[bd.fileHash][bd.hash] = bytes
		bc.Broadcast()
		wg.Done()
	}
}
