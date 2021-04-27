package files

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/recovery2/remote"
	track "github.com/morrocker/recoveryserver/recovery2/tracker"
	"github.com/morrocker/utils"
	"golang.org/x/text/unicode/norm"
)

func startSmallFilesWorkers(data Data, rbs remote.RBS, tr *tracker.SuperTracker) (chan []*fileData, *sync.WaitGroup) {
	log.Task("Starting %d big files workers", data.Workers)
	wg := &sync.WaitGroup{}
	fdc := make(chan []*fileData)
	wg.Add(data.Workers)
	for x := 0; x < data.Workers; x++ {
		go smallFilesWorker(fdc, data.User, wg, rbs, tr)
	}
	return fdc, wg
}

func smallFilesWorker(fc chan []*fileData, user string, wg *sync.WaitGroup, rbs remote.RBS, tr *tracker.SuperTracker /*, bc chan bData*/) {
	op := "recovery.smallFilesWorker()"
	log.Taskln("Starting small files workers")
	for fda := range fc {
		positionArray := []*fileData{}
		blocksArray := []string{}
		for _, fd := range fda {
			log.Info("Recovering file %s [%s]", fd.OutputPath, utils.B2H(int64(fd.Mt.Mf.Size)))
			blocksArray = append(blocksArray, fd.blocksList...)
			for x := 0; x < len(fd.blocksList); x++ {
				positionArray = append(positionArray, fd)
			}
		}
		bytesArrays, err := rbs.GetBlocks(blocksArray, user)
		if err != nil {
			log.Errorln(errors.Extend(op, err))
			track.FailedFiles(len(bytesArrays), tr)
			continue
		}
		bytesArray := []byte{}

		for i, content := range bytesArrays {
			if i == 0 {
				bytesArray = appendContent(bytesArray, content, tr)
			} else if positionArray[i-1].Mt.Mf.Hash == positionArray[i].Mt.Mf.Hash {
				bytesArray = appendContent(bytesArray, content, tr)
			} else {
				if err := writeSmallFile(positionArray[i-1], bytesArray); err != nil {
					log.Errorln(errors.Extend(op, err))
					track.FailedFiles(1, tr)
				}
				bytesArray = []byte{}
				bytesArray = appendContent(bytesArray, content, tr)
			}
		}

		if err := writeSmallFile(positionArray[len(positionArray)-1], bytesArray); err != nil {
			log.Errorln(errors.Extend(op, err))
			track.FailedFiles(1, tr)
		} // writting the last file
	}
	wg.Done()
}

func appendContent(content []byte, newContent []byte, tr *tracker.SuperTracker) []byte {
	if newContent != nil {
		content = append(content, newContent...)
	} else {
		tr.ChangeCurr("errors", 1)
		tr.ChangeTotal("errors", 1)
		content = append(content, zeroedBuffer...)
	}
	return content
}

// Creating recovery file
func writeSmallFile(fd *fileData, content []byte) error {
	op := "recovery.writeFile()"
	// log.Task("Writting small file %s [%s]", fd.OutputPath, utils.B2H(int64(fd.Mt.Mf.Size)))
	f, err := os.Create(norm.NFC.String(fd.OutputPath))
	if err != nil {
		return errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", fd.OutputPath, err))
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		return errors.New(op, fmt.Sprintf("error could not write content for file '%s': %v\n", fd.OutputPath, err))
	}
	return nil
}

func startBigFilesWorkers(data Data, rbs remote.RBS, tr *tracker.SuperTracker) (chan *fileData, chan bigData, *sync.WaitGroup, *sync.WaitGroup) {
	log.Task("Starting %d big files workers", data.Workers)
	wg := &sync.WaitGroup{}
	wg2 := &sync.WaitGroup{}
	bfc := make(chan *fileData)
	bdc := make(chan bigData)
	wg.Add(data.Workers)
	for x := 0; x < data.Workers; x++ {
		go blocksWorker(bdc, data.User, wg, rbs, tr)
	}
	wg2.Add(data.Workers)
	for x := 0; x < data.Workers; x++ {
		go recoverBigFile(bfc, bdc, data.User, wg2, rbs, tr)
	}
	return bfc, bdc, wg, wg2
}

type returnBlock struct {
	idx     int
	content []byte
	err     error
}

type bigData struct {
	idx   int
	hashs []string
	ret   chan returnBlock
}

// func bigFilesWorker(fc chan *fileData, user string, wg *sync.WaitGroup, rbs remote.RBS /*, bc chan bData*/) {
func recoverBigFile(bfc chan *fileData, fdc chan bigData, user string, wg *sync.WaitGroup, rbs remote.RBS, tr *tracker.SuperTracker) {
	op := "files.recoverBigFile()"
	for fd := range bfc {
		log.Info("Recovering file %s [%s]", fd.OutputPath, utils.B2H(int64(fd.Mt.Mf.Size)))

		// Creating recovery file
		f, err := os.Create(norm.NFC.String(fd.OutputPath))
		if err != nil {
			track.FailedFiles(1, tr)
			log.Errorln(errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", fd.OutputPath, err)))
		}

		ret := make(chan returnBlock)
		blocksBuffer := make(map[int][]byte)
		// Sending blocks to the blocks worker
		idx := 0
		go func() {
			hashs := []string{}
			for i, hash := range fd.blocksList {
				if i%100 == 0 && i != 0 {
					fdc <- bigData{idx: idx, hashs: hashs, ret: ret}
					hashs = []string{}
					hashs = append(hashs, hash)
					idx++
				} else {
					hashs = append(hashs, hash)
				}
				// if r.flowGate() {
				// 	return
				// }
			}
			if len(hashs) != 0 {
				fdc <- bigData{idx: idx, hashs: hashs, ret: ret}
			}
		}()

		// Receiving blocks from blocksworkers and writting into file
		for x := 0; x < idx+1; x++ {
			// if r.flowGate() {
			// 	f.Close()
			// 	break Outer
			// }
			content, ok := blocksBuffer[x]
			if ok {
				// log.Info("Block #%d is being written from buffer for %s", x, utils.Trimmer(fd.OutputPath, 0, 50))
				if _, err := f.Write(content); err != nil {
					track.FailedFiles(1, tr)
					log.Errorln(errors.Extend(op, err))
				}
				tr.ChangeCurr("buffer", -1)
				delete(blocksBuffer, x)
				continue
			}
			for data := range ret {
				if data.err != nil {
					f.Close()
					log.Errorln(errors.Extend(op, err))
				}
				if data.idx == x {
					// log.Info("Block #%d is being written directly for %s", x, utils.Trimmer(fd.OutputPath, 0, 50))
					if _, err := f.Write(data.content); err != nil {
						track.FailedFiles(1, tr)
						log.Errorln(errors.Extend(op, err))
					}
					break
				}
				checkBuffer(tr)
				blocksBuffer[data.idx] = data.content
				tr.ChangeCurr("blocksBuffer", 1)
			}
		}
		tr.ChangeCurr("files", 1)
		f.Close()
	}
	wg.Done()
}

func blocksWorker(bfc chan bigData, user string, wg *sync.WaitGroup, rbs remote.RBS, tr *tracker.SuperTracker) {
	op := "recovery.files.blocksWorker()"
Outer:
	for data := range bfc {
		ret := returnBlock{idx: data.idx}
		bytesArray, err := rbs.GetBlocks(data.hashs, user)
		if err != nil {
			ret.err = errors.Extend(op, err)
			data.ret <- ret
			continue Outer
		}
		for _, bytes := range bytesArray {
			if bytes == nil {
				ret.content = append(ret.content, zeroedBuffer...)
				tr.ChangeCurr("totalsize", len(bytes))
				tr.ChangeCurr("errors", 1)
			} else {
				tr.ChangeCurr("size", len(bytes))
				tr.ChangeCurr("totalsize", len(bytes))
				ret.content = append(ret.content, bytes...)
			}
		}
		data.ret <- ret
	}
	wg.Done()
}

func checkBuffer(tr *tracker.SuperTracker) {
	for {
		c, t, err := tr.RawValues("buffer")
		if err != nil {
			log.Errorln(errors.New("files.recoveries.checkBuffer()", err))
		}
		if c < t {
			break
		}
		time.Sleep(time.Millisecond)
	}
}
