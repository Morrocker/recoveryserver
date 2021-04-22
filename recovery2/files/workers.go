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
	"github.com/morrocker/utils"
	"golang.org/x/text/unicode/norm"
)

func startSmallFilesWorkers(data Data, rbs remote.RBS) (chan []*fileData, *sync.WaitGroup) {
	log.Task("Starting %d big files workers", data.Workers)
	wg := &sync.WaitGroup{}
	fdc := make(chan []*fileData)
	wg.Add(data.Workers)
	for x := 0; x < data.Workers; x++ {
		go smallFilesWorker(fdc, data.User, wg, rbs)
	}
	return fdc, wg
}

func smallFilesWorker(fc chan []*fileData, user string, wg *sync.WaitGroup, rbs remote.RBS /*, bc chan bData*/) {
	op := "recovery.smallFilesWorker()"
	log.Taskln("Starting small files workers")
	for fda := range fc {
		// spew.Dump(fda[0])
		// log.Info("Small sublist #%d", len(fda))
		positionArray := []*fileData{}
		blocksArray := []string{}
		for _, fd := range fda {
			log.Info("Recovering file %s [%s]", fd.OutputPath, utils.B2H(int64(fd.Mt.Mf.Size)))
			blocksArray = append(blocksArray, fd.blocksList...)
			for x := 0; x < len(fd.blocksList); x++ {
				positionArray = append(positionArray, fd)
			}
		}
		// log.Notice("Blockslists: #%d.", len(blocksArray))
		bytesArrays, err := rbs.GetBlocks(blocksArray, user)
		if err != nil {
			log.Errorln(errors.Extend(op, err))
			continue
		}
		bytesArray := []byte{}
		// log.Task("Writting small files. Original list: #%d. Positional list: #%d. BlocksArray: #%d. BytesArrays: #%d", len(fda), len(positionArray), len(blocksArray), len(bytesArrays))
		// log.Notice("Analyzing bytes arrays total:%d", len(bytesArrays))
		// for i, btArray := range bytesArrays {
		// log.Bench("Index: %d. Length: %d", i, len(btArray))
		// }

		for i, content := range bytesArrays {
			// log.Bench("Index: %d", i)
			if i == 0 {
				bytesArray = appendContent(bytesArray, content)
			} else if positionArray[i-1].Mt.Mf.Hash == positionArray[i].Mt.Mf.Hash {
				bytesArray = appendContent(bytesArray, content)
			} else {
				if err := writeSmallFile(positionArray[i-1], bytesArray); err != nil {
					log.Errorln(errors.Extend(op, err))
				}
				bytesArray = []byte{}
				bytesArray = appendContent(bytesArray, content)
			}
		}

		writeSmallFile(positionArray[len(positionArray)-1], bytesArray) // writting the last file
	}
	wg.Done()
}

func appendContent(content []byte, newContent []byte) []byte {
	if newContent != nil {
		content = append(content, newContent...)
	} else {
		content = append(content, zeroedBuffer...)
	}
	return content
}

func startBigFilesWorkers(data Data, rbs remote.RBS) (chan bigData, *sync.WaitGroup) {
	log.Task("Starting %d big files workers", data.Workers)
	wg := &sync.WaitGroup{}
	bfc := make(chan bigData)
	wg.Add(data.Workers)
	for x := 0; x < data.Workers; x++ {
		go blocksWorker(bfc, data.User, wg, rbs)
	}
	return bfc, wg
}

// func bigFilesWorker(fc chan *fileData, user string, wg *sync.WaitGroup, rbs remote.RBS /*, bc chan bData*/) {
// 	op := "recovery.bigFilesWorker()"
// 	log.Taskln("Starting big files workers")
// Outer:
// 	for fd := range fc {
// 		log.Task("Writting big file %s", fd.OutputPath)
// 		f, err := os.Create(norm.NFC.String(fd.OutputPath))
// 		if err != nil {
// 			// r.increaseErrors()
// 			// r.tracker.ChangeCurr("completedSize", mt.mf.Size)
// 			log.Errorln(errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", fd.OutputPath, err)))
// 		}
// 		for _, block := range fd.blocksList {
// 			// bytes := []byte{}
// 			subList := []string{}
// 			for x := 0; x < 100; x++ {
// 				subList = append(subList, block)
// 			}
// 			bytesArray, err := rbs.GetBlocks(subList, user)
// 			if err != nil {
// 				log.Errorln(errors.Extend(op, err))
// 				f.Close()
// 				continue Outer
// 			}
// 			for _, bytes := range bytesArray {
// 				if bytes == nil {
// 					bytes = zeroedBuffer
// 				}
// 				if _, err := f.Write(bytes); err != nil {
// 					// r.increaseErrors()
// 					// r.log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", blocks[x], path, err)))
// 					// r.tracker.ChangeCurr("completedSize", len(content))
// 					log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for file '%s': %v\n", fd.OutputPath, err)))
// 				}
// 			}
// 		}
// 		f.Close()
// 	}
// 	wg.Done()
// }

func writeSmallFile(fd *fileData, content []byte) error {
	// Creating recovery file
	op := "recovery.writeFile()"
	// log.Task("Writting small file %s [%s]", fd.OutputPath, utils.B2H(int64(fd.Mt.Mf.Size)))
	f, err := os.Create(norm.NFC.String(fd.OutputPath))
	if err != nil {
		// r.increaseErrors()
		// log.Errorln(errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", path, err)))
		// r.tracker.ChangeCurr("completedSize", mt.mf.Size)
		return errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", fd.OutputPath, err))
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		// r.increaseErrors()
		// r.log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", blocks[x], path, err)))
		// r.tracker.ChangeCurr("completedSize", len(content))
		return errors.New(op, fmt.Sprintf("error could not write content for file '%s': %v\n", fd.OutputPath, err))
	}

	return nil
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
func recoverBigFile(fd *fileData, user string, bfc chan bigData, wg *sync.WaitGroup, rbs remote.RBS) error {
	op := "files.recoverBigFile()"
	// if r.flowGate() {
	// 	break
	// }
	log.Info("Recovering file %s [%s]", fd.OutputPath, utils.B2H(int64(fd.Mt.Mf.Size)))

	// Creating recovery file
	f, err := os.Create(norm.NFC.String(fd.OutputPath))
	if err != nil {
		// r.increaseErrors()
		log.Errorln(errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", fd.OutputPath, err)))
		// r.tracker.ChangeCurr("completedSize", mt.mf.Size)
		return errors.New(op, err)
	}

	ret := make(chan returnBlock)
	blocksBuffer := make(map[int][]byte)
	// Sending blocks to the blocks worker
	idx := 0
	go func() {
		hashs := []string{}
		for i, hash := range fd.blocksList {
			if i%100 == 0 && i != 0 {
				bfc <- bigData{idx: idx, hashs: hashs, ret: ret}
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
			bfc <- bigData{idx: idx, hashs: hashs, ret: ret}
		}
	}()

	// Receiving blocks from blocksworkers and writting into file
	tr := tracker.New()
	tr.AddGauge("buffer", "buffer", 10)
	for x := 0; x < idx+1; x++ {
		// if r.flowGate() {
		// 	f.Close()
		// 	break Outer
		// }
		content, ok := blocksBuffer[x]
		if ok {
			// log.Info("Block #%d is being written from buffer for %s", x, utils.Trimmer(fd.OutputPath, 0, 50))
			if _, err := f.Write(content); err != nil {
				// r.increaseErrors()
				// r.log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", blocks[x], path, err)))
				// r.tracker.ChangeCurr("completedSize", len(content))
				return errors.New(op, err)
			}
			// r.tracker.ChangeCurr("completedSize", len(content))
			// r.tracker.ChangeCurr("size", len(content))
			// r.tracker.ChangeCurr("blocks")
			tr.ChangeCurr("buffer", -1)
			delete(blocksBuffer, x)
			continue
		}
		for data := range ret {
			if data.err != nil {
				f.Close()
				return errors.Extend(op, data.err)
			}
			if data.idx == x {
				// log.Info("Block #%d is being written directly for %s", x, utils.Trimmer(fd.OutputPath, 0, 50))
				if _, err := f.Write(data.content); err != nil {
					// r.increaseErrors()
					// r.log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", blocks[x], path[len(path)-20:], err)))
					// r.tracker.ChangeCurr("completedSize", len(d.content))
					return errors.New(op, err)
				}
				// r.tracker.ChangeCurr("completedSize", len(d.content))
				// r.tracker.ChangeCurr("size", len(d.content))
				// r.tracker.ChangeCurr("blocks")
				break
			}
			checkBuffer(tr)
			blocksBuffer[data.idx] = data.content
			tr.ChangeCurr("blocksBuffer", 1)
		}
	}
	// r.tracker.ChangeCurr("files")
	// log.Info("Finishing file %s", path[len(path)-20:])
	f.Close()
	return nil
}

func blocksWorker(bfc chan bigData, user string, wg *sync.WaitGroup, rbs remote.RBS) {
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
			} else {
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
