package files

import (
	"fmt"
	"os"
	"sync"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/recovery2/remote"
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
		positionArray := []*fileData{}
		blocksArray := []string{}
		for _, fd := range fda {
			blocksArray = append(blocksArray, fd.blocksList...)
			for x := 0; x < len(fd.blocksList); x++ {
				positionArray = append(positionArray, fd)
			}
		}
		log.Notice("Blockslists: #%d.", len(blocksArray))
		bytesArrays, err := rbs.GetBlocks(blocksArray, user)
		if err != nil {
			log.Errorln(errors.Extend(op, err))
			continue
		}
		bytesArray := []byte{}
		log.Task("Writting small files. Original list: #%d. Positional list: #%d. BlocksArray: #%d", len(fda), len(positionArray), len(blocksArray))
		for i, content := range bytesArrays {
			log.Bench("Index: %d", i)
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

func startBigFilesWorkers(data Data, rbs remote.RBS) (chan *fileData, *sync.WaitGroup) {
	log.Task("Starting %d big files workers", data.Workers)
	wg := &sync.WaitGroup{}
	fdc := make(chan *fileData)
	wg.Add(data.Workers)
	for x := 0; x < data.Workers; x++ {
		go bigFilesWorker(fdc, data.User, wg, rbs)
	}
	return fdc, wg
}

func bigFilesWorker(fc chan *fileData, user string, wg *sync.WaitGroup, rbs remote.RBS /*, bc chan bData*/) {
	op := "recovery.bigFilesWorker()"
	log.Taskln("Starting big files workers")
Outer:
	for fd := range fc {
		log.Taskln("Writting big file %s", fd.OutputPath)
		f, err := os.Create(norm.NFC.String(fd.OutputPath))
		if err != nil {
			// r.increaseErrors()
			// r.tracker.ChangeCurr("completedSize", mt.mf.Size)
			log.Errorln(errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", fd.OutputPath, err)))
		}
		for _, block := range fd.blocksList {
			// bytes := []byte{}
			subList := []string{}
			for x := 0; x < 100; x++ {
				subList = append(subList, block)
			}
			bytesArray, err := rbs.GetBlocks(subList, user)
			if err != nil {
				log.Errorln(errors.Extend(op, err))
				f.Close()
				continue Outer
			}
			for _, bytes := range bytesArray {
				if bytes == nil {
					bytes = zeroedBuffer
				}
				if _, err := f.Write(bytes); err != nil {
					// r.increaseErrors()
					// r.log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", blocks[x], path, err)))
					// r.tracker.ChangeCurr("completedSize", len(content))
					log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for file '%s': %v\n", fd.OutputPath, err)))
				}
			}
		}
		f.Close()
	}
	wg.Done()
}

func writeSmallFile(fd *fileData, content []byte) error {
	// Creating recovery file
	op := "recovery.writeFile()"
	log.Task("Writting small file %s", fd.OutputPath)
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
