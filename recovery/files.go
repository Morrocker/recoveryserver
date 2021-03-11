package recovery

import (
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/clonercl/reposerver"
	"github.com/morrocker/errors"
	"github.com/morrocker/logger"
	"github.com/morrocker/recoveryserver/config"
	"golang.org/x/text/unicode/norm"
)

type fileDescriptor struct {
	path string
	hash string
	size uint64
}

func (r *Recovery) getFiles(mt *MetaTree) error {
	errPath := "recovery.getFiles()"
	fileQueue := make(chan *fileDescriptor)
	wg := sync.WaitGroup{}

	logger.Notice("Starting File workers")
	for i := 0; i < config.Data.FileWorkers; i++ {
		wg.Add(1)
		go r.fileWorker(fileQueue, &wg)
	}

	startingTime := time.Now()
	logger.Notice("Creating root directory %s", r.Destination)
	if err := os.MkdirAll(r.Destination, 0700); err != nil {
		fmt.Printf("could not create output path: %v\n", err)
		return errors.New(errPath, err)
	}
	logger.Notice("Starting file recovery")
	r.createRec(r.Destination, mt, fileQueue)
	close(fileQueue)
	wg.Wait()

	totalTime := time.Duration(int64(time.Since(startingTime)) / int64(time.Second) * int64(time.Second))
	// CHANGE THIS
	logger.Info("Total time %s", totalTime)
	return nil
}

func (r *Recovery) createRec(base string, tree *MetaTree, fq chan *fileDescriptor) {
	f := tree.mf
	p := path.Join(base, f.Name)

	logger.Notice("Recovering file %s path %s", tree.mf.Name, p)
	if f.Type == reposerver.FolderType {
		if f.Parent == "" {
			p = base
		} else {
			if err := os.MkdirAll(norm.NFC.String(p), 0700); err != nil { // maybe dont clean names
				panic(fmt.Sprintf("could not create path '%s': %v\n", p, err))
			}
		}

		for _, child := range tree.children {
			r.createRec(p, child, fq)
		}

		return
	}

	// fmt.Printf("recovering %s [%s]\n", p, b2h(uint64(f.Size)))
	fq <- &fileDescriptor{
		path: p,
		hash: f.Hash,
		size: uint64(f.Size),
	}
}

func (r *Recovery) fileWorker(q chan *fileDescriptor, wg *sync.WaitGroup) {
	for {
		fd := <-q
		if fd == nil {
			wg.Done()
			return
		}

		r.recoverFile(fd.path, fd.hash, fd.size)
	}
}

func (r *Recovery) recoverFile(p, hash string, size uint64) {
	if fi, err := os.Stat(p); err == nil {
		if fi.Size() == int64(size) {
			fmt.Printf("skipping file '%s'\n", p)
			return
		}
	}

	blist := r.Cloud.GetBlocksList(hash, r.Data.User)
	if blist == nil {
		fmt.Printf("error could not create file '%s' because fileblock is unavailable. Deleting\n", p)
		return
	}

	f, err := os.Create(norm.NFC.String(p))
	if err != nil {
		panic(fmt.Sprintf("error could not create file '%s' : %v", p, err))
	}

	var zeroedBuffer = make([]byte, 1024*1000)
	for _, hash := range blist.Blocks {
		b, err := r.Cloud.GetBlock(hash, r.Data.User)

		if err != nil {
			fmt.Printf("error failed to retrieve block '%s' for '%s'.\n", hash, p)
			if _, err2 := f.Write(zeroedBuffer); err2 != nil {
				panic(fmt.Sprintf("error could not write zeroed content for block '%s' for file '%s': %v\n", hash, p, err))
			}
		} else {
			if _, err2 := f.Write(b); err2 != nil {
				panic(fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", hash, p, err2))
			}

			// atomic.AddUint64(&totalBytes, uint64(len(b)))
		}
	}

	f.Close()
	// atomic.AddUint64(&nFiles, 1)
}

type blocksList struct { // TODO(br): deprecate
	Blocks []string `json:"blocks"`
}
