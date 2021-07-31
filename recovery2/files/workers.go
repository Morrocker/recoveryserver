package files

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/flow"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/recovery2/remote"
	"github.com/morrocker/utils"
	"golang.org/x/text/unicode/norm"

	"github.com/morrocker/recoveryserver/recovery2/tracker"
)

func blockListWorker(
	blockChan chan string,
	destMap map[string]*fileData, user string,
	wg *sync.WaitGroup,
	rbs remote.RBS,
	rt *tracker.RecoveryTracker,
	ctrl flow.Controller) {
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
	wg.Done()
}

func fileWorker(
	fdc chan *fileData, bdc chan blockData,
	user string,
	bufferMap *sync.Map,
	wg *sync.WaitGroup,
	rbs remote.RBS,
	rt *tracker.RecoveryTracker,
	ctrl flow.Controller) {
	op := "recovery.fileWorker()"

	for fd := range fdc {
		wg.Add(1)
		log.Info("Recovering file %s\t[%s]", fd.OutputPath, utils.B2H(fd.Mt.Mf.Size))
		time.Sleep(1 * time.Second)
		go func() {
			for _, block := range fd.blocksList {
				for {
					if ctrl.Checkpoint() != 0 {
						return
					}
					c, t := rt.Gauges["membuff"].RawValues()
					if c >= t {
						continue
					}
					newBlockData := blockData{
						user:     user,
						hash:     block,
						fileHash: fd.Mt.Mf.Hash,
					}
					bdc <- newBlockData
					break
				}
			}
		}()
		f, err := os.Create(norm.NFC.String(fd.OutputPath))
		if err != nil {
			log.Errorln(errors.New(op, err))
		}
		for _, block := range fd.blocksList {
			if ctrl.Checkpoint() != 0 {
				break
			}
			for {
				subMapIf, ok := bufferMap.Load(fd.Mt.Mf.Hash)
				if ok {
					subMap := subMapIf.(*sync.Map)
					val, ok := subMap.LoadAndDelete(block)
					if ok {
						val := val.(mapBlockData)
						if _, err := f.Write(val.bytes); err != nil {
							log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for file '%s': %v\n", fd.OutputPath, err)))
						}
						if val.ctr == 1 {
							rt.Gauges["membuff"].Current(-1)
							break
						} else if val.ctr > 1 {
							val.ctr--
							subMap.Store(block, val)
						} else {
							log.Errorln("Somehow counter is going lower than 1")
						}
					}
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
		f.Close()

		bufferMap.Delete(fd.Mt.Mf.Hash)
		wg.Done()
	}
}

type blockData struct {
	user     string
	hash     string
	fileHash string
}

type mapBlockData struct {
	bytes []byte
	ctr   int
}

func filesBlockWorker(
	bdc chan blockData,
	bufferMap *sync.Map,
	wg *sync.WaitGroup,
	rbs remote.RBS,
	rt *tracker.RecoveryTracker,
	ctrl flow.Controller) {
	op := "files.filesBlockWorker()"

	for bd := range bdc {
		wg.Add(1)
		if ctrl.Checkpoint() != 0 {
			break
		}
		bytes, err := rbs.GetBlock(bd.hash, bd.user)
		if err != nil {
			log.Errorln(errors.Extend(op, err))
			bytes = zeroedBuffer
		}
		newData := mapBlockData{
			bytes: bytes,
			ctr:   1,
		}
		subMapIf, ok := bufferMap.Load(bd.fileHash)
		if ok {
			subMap := subMapIf.(*sync.Map)
			valIf, ok := subMap.Load(bd.hash)
			if !ok {
				subMap.Store(bd.hash, newData)
			} else {
				val := valIf.(mapBlockData)
				newData.ctr = val.ctr + 1
				subMap.Store(bd.hash, val)
			}
			rt.Gauges["membuff"].Current(1)
		}
		wg.Done()
	}
}
