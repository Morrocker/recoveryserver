package recovery

import (
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/clonercl/reposerver"
	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/utils"
	"golang.org/x/text/unicode/norm"
)

type fileQueue struct {
	ToDo []*MetaTree
	lock sync.Mutex
}

var fq fileQueue = fileQueue{}

func (r *Recovery) getFiles(mt *MetaTree) error {
	op := "recovery.getFiles()"
	fc := make(chan *MetaTree)
	wg := sync.WaitGroup{}

	r.log.Notice("Starting %d File workers", config.Data.FileWorkers)
	for i := 0; i < config.Data.FileWorkers; i++ {
		go r.fileWorker(fc, &wg)
	}

	r.log.Notice("Creating root directory " + r.outputTo)
	if err := os.MkdirAll(r.outputTo, 0700); err != nil {
		return errors.New(op, errors.Extend(op, err))
	}
	dst := path.Join(r.outputTo, r.Data.Org, r.Data.User, r.Data.Machine, r.Data.Disk)
	log.Info("Writting files to " + dst)
	r.createFileQueue(dst, mt)

	for _, tree := range fq.ToDo {
		if r.flowGate() {
			break
		}
		fc <- tree
	}

	wg.Wait()
	close(fc)
	fq = fileQueue{}
	return nil
}

func (r *Recovery) createFileQueue(filepath string, mt *MetaTree) {
	f := mt.mf
	p := path.Join(filepath, f.Name)
	if f.Type == reposerver.FolderType {
		if f.Parent == "" {
			p = filepath
		} else {
			if err := os.MkdirAll(norm.NFC.String(p), 0700); err != nil {
				panic(fmt.Sprintf("could not create path '%s': %v\n", p, err))
			}
		}
		for _, child := range mt.children {
			if r.flowGate() {
				break
			}
			r.createFileQueue(p, child)
		}
		return
	}
	mt.path = p
	fq.addFile(mt)
}

func (f *fileQueue) addFile(mt *MetaTree) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.ToDo = append(f.ToDo, mt)
}

func (r *Recovery) fileWorker(fc chan *MetaTree, wg *sync.WaitGroup) {
	op := "recovery.fileWorker()"
	RBS := NewRBS(r.cloud)
	for mt := range fc {
		if r.flowGate() {
			break
		}
		wg.Add(1)
		if err := r.recoverFile(mt.path, mt.mf.Hash, uint64(mt.mf.Size), RBS); err != nil {
			r.log.Errorln(errors.Extend(op, err))
		}
		wg.Done()
	}
}

func (r *Recovery) recoverFile(p, hash string, size uint64, RBS *RBS) error {
	op := "recovery.recoverFile()"
	if r.flowGate() {
		return nil
	}
	if fi, err := os.Stat(p); err == nil {
		if fi.Size() == int64(size) {
			r.updateTrackerCurrent(int64(size))
			r.log.NoticeV("skipping file '%s'", p)
			return nil
		}
	}

	r.log.Info("Recovering file %s [%s]", p, utils.B2H(int64(size)))
	blist, err := RBS.GetBlocksList(hash, r.Data.User)
	if err != nil {
		r.increaseErrors()
		r.log.ErrorlnV(errors.New(op, "error could not create file '%s' because fileblock is unavailable"))
		return errors.New(op, err)
	}
	r.tracker.IncreaseCurr("blocks")

	f, err := os.Create(norm.NFC.String(p))
	if err != nil {
		r.increaseErrors()
		return errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", p, err))
	}
	defer f.Close()

	var zeroedBuffer = make([]byte, 1024*1000)
	for _, hash := range blist.Blocks {
		if r.flowGate() {
			break
		}
		b, err := RBS.GetBlock(hash, r.Data.User)

		if err != nil {
			if _, err2 := f.Write(zeroedBuffer); err2 != nil {
				r.increaseErrors()
				return errors.New(op, fmt.Sprintf("error could not write zeroed content for block '%s' for file '%s': %v\n", hash, p, err))
			}
		} else {
			if _, err2 := f.Write(b); err2 != nil {
				r.increaseErrors()
				return errors.New(op, fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", hash, p, err2))
			}
		}

		r.tracker.ChangeCurr("size", len(b))
		r.tracker.IncreaseCurr("blocks")
	}

	r.tracker.IncreaseCurr("files")
	return nil
}
