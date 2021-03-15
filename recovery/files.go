package recovery

import (
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/clonercl/reposerver"
	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"golang.org/x/text/unicode/norm"
)

type fileDescriptor struct {
	path string
	hash string
	size uint64
}

type fileQueue struct {
	ToDo []*MetaTree
	lock sync.Mutex
}

var fq fileQueue = fileQueue{}

func (r *Recovery) getFiles(mt *MetaTree) error {
	errPath := "recovery.getFiles()"
	fc := make(chan *MetaTree)
	wg := sync.WaitGroup{}

	log.Notice("Starting File workers")
	for i := 0; i < config.Data.FileWorkers; i++ {
		go r.fileWorker(fc, &wg)
	}

	startingTime := time.Now()
	log.Notice("Creating root directory %s", r.Destination)
	if err := os.MkdirAll(r.Destination, 0700); err != nil {
		fmt.Printf("could not create output path: %v\n", err)
		return errors.New(errPath, err)
	}

	r.createFileQueue(r.Destination, mt)

	for _, tree := range fq.ToDo {
		r.stopGate()
		fc <- tree
	}

	wg.Wait()
	close(fc)

	totalTime := time.Duration(int64(time.Since(startingTime)) / int64(time.Second) * int64(time.Second))
	// CHANGE THIS
	log.Info("Total time %s", totalTime)
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
			r.createFileQueue(p, child)
		}
		return
	}
	mt.path = p
	fq.addFile(mt)
}

func (r *Recovery) fileWorker(fc chan *MetaTree, wg *sync.WaitGroup) {
	errPath := "recovery.fileWorker()"
	for mt := range fc {
		wg.Add(1)
		if err := r.recoverFile(mt.path, mt.mf.Hash, uint64(mt.mf.Size)); err != nil {
			log.Error("%s", errors.Extend(errPath, err))
		}
		wg.Done()
	}
}

func (r *Recovery) recoverFile(p, hash string, size uint64) error {
	errPath := "recovery.recoverFile()"
	// log.Info("Recovering file %s [%s]", p, utils.B2H(size))
	if fi, err := os.Stat(p); err == nil {
		if fi.Size() == int64(size) {
			r.updateTrackerCurrent(int64(size))
			log.Notice("skipping file '%s'", p)
			return nil
		}
	}

	blist := r.Cloud.GetBlocksList(hash, r.Data.User)
	if blist == nil {
		r.increaseErrors()
		return errors.New(errPath, fmt.Sprintf("error could not create file '%s' because fileblock is unavailable.\n", p))
	}

	f, err := os.Create(norm.NFC.String(p))
	if err != nil {
		r.increaseErrors()
		return errors.New(errPath, fmt.Sprintf("error could not create file '%s' : %v\n", p, err))
	}

	var zeroedBuffer = make([]byte, 1024*1000)
	for _, hash := range blist.Blocks {
		r.stopGate()
		b, err := r.Cloud.GetBlock(hash, r.Data.User)

		if err != nil {
			if _, err2 := f.Write(zeroedBuffer); err2 != nil {
				r.increaseErrors()
				return errors.New(errPath, fmt.Sprintf("error could not write zeroed content for block '%s' for file '%s': %v\n", hash, p, err))
			}
		} else {
			if _, err2 := f.Write(b); err2 != nil {
				r.increaseErrors()
				return errors.New(errPath, fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", hash, p, err2))
			}
			// atomic.AddUint64(&totalBytes, uint64(len(b)))
		}
	}
	f.Close()
	r.updateTrackerCurrent(int64(size))
	// atomic.AddUint64(&nFiles, 1)
	return nil
}

type blocksList struct { // TODO(br): deprecate
	Blocks []string `json:"blocks"`
}

func (f *fileQueue) addFile(mt *MetaTree) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.ToDo = append(f.ToDo, mt)
}
