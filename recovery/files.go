package recovery

import (
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/clonercl/reposerver"
	"github.com/morrocker/broadcast"
	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/utils"
	"golang.org/x/text/unicode/norm"
)

type fileQueue struct {
	ToDo []*MetaTree
	lock sync.Mutex
}

type bData struct {
	id   int
	hash string
	ret  chan returnBlock
}

type returnBlock struct {
	id      int
	content []byte
}

var fq fileQueue = fileQueue{}

func (r *Recovery) getFiles(mt *MetaTree) error {
	op := "recovery.getFiles()"
	fc := make(chan *MetaTree)
	bc := make(chan bData)
	wg := sync.WaitGroup{}
	wg2 := sync.WaitGroup{}
	broadcaster := broadcast.New()

	r.log.Notice("Starting %d File workers", config.Data.FileWorkers)
	for i := 0; i < config.Data.FileWorkers; i++ {
		wg.Add(1)
		go r.fileWorker(fc, &wg, broadcaster, bc)
	}

	r.log.Notice("Starting %d Block workers", config.Data.FileWorkers)
	for i := 0; i < config.Data.BlockWorkers; i++ {
		wg2.Add(1)
		go r.blockWorker(bc, &wg)
	}

	dst := path.Join(r.OutputTo, r.Data.Org, r.Data.User, r.Data.Machine, r.Data.Disk)
	r.log.Notice("Creating root directory " + dst)
	if err := os.MkdirAll(dst, 0700); err != nil {
		return errors.New(op, errors.Extend(op, err))
	}
	log.Info("Writting files to " + dst)
	r.createFileQueue(dst, mt)

	time.Sleep(5 * time.Second)

	for _, tree := range fq.ToDo {
		if r.flowGate() {
			break
		}
		fc <- tree
	}

	time.Sleep(time.Second)
	close(fc)
	wg.Wait()
	fq = fileQueue{}
	r.log.Noticeln("Files retrieval completed")
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

func (r *Recovery) fileWorker(fc chan *MetaTree, wg *sync.WaitGroup, b *broadcast.Broadcaster, bc chan bData) {
	op := "recovery.fileWorker()"
	for mt := range fc {
		if r.flowGate() {
			break
		}
		if err := r.recoverFile(mt.path, mt.mf.Hash, uint64(mt.mf.Size), b, bc); err != nil {
			r.log.Errorln(errors.Extend(op, err))
		}
	}
	wg.Done()
}

func (r *Recovery) blockWorker(dc chan bData, wg2 *sync.WaitGroup) {
	for data := range dc {
		if r.flowGate() {
			break
		}
		// log.Info("Recovering Block %s", data.hash)
		b, err := r.RBS.GetBlock(data.hash, r.Data.User)
		if err != nil {
			var zeroedBuffer = make([]byte, 1024*1000)
			// log.Errorln(errors.Extend("recovery.fileWorker()", err))
			data.ret <- returnBlock{data.id, zeroedBuffer}
		}
		// log.Info("Returning Block %s", data.hash)
		data.ret <- returnBlock{data.id, b}
	}
	wg2.Done()
}

func (r *Recovery) recoverFile(p, hash string, size uint64, b *broadcast.Broadcaster, bc chan bData) error {
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
	blist, err := r.RBS.GetBlocksList(hash, r.Data.User)
	if err != nil {
		r.increaseErrors()
		r.log.ErrorlnV(errors.New(op, fmt.Sprintf("error could not create file '%s' because fileblock is unavailable", p)))
		return errors.New(op, err)
	}
	r.tracker.IncreaseCurr("blocks")

	r.fileWriter(p, blist.Blocks, b, bc)

	return nil
}

func (r *Recovery) fileWriter(path string, blocks []string, b *broadcast.Broadcaster, bc chan bData) {
	op := "recovery.fileWriter()"
	f, err := os.Create(norm.NFC.String(path))
	if err != nil {
		r.increaseErrors()
		log.Errorln(errors.New(op, fmt.Sprintf("error could not create file '%s' : %v\n", path, err)))
	}
	defer f.Close()

	ret := make(chan returnBlock)
	blocksBuffer := make(map[int][]byte)
	n := len(blocks)
	go func() {
		for i, hash := range blocks {
			bc <- bData{id: i, hash: hash, ret: ret}
		}
	}()

	for x := 0; x < n; x++ {
		content, ok := blocksBuffer[x]
		if ok {
			// log.Info("Block #%d is being written from buffer", x)
			if _, err := f.Write(content); err != nil {
				r.increaseErrors()
				r.log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", blocks[x], path, err)))
			}
			r.tracker.ChangeCurr("size", len(content))
			r.tracker.IncreaseCurr("blocks")
			r.tracker.ChangeCurr("blocksBuffer", -1)
			delete(blocksBuffer, x)
			b.Broadcast()
			continue
		}
		for d := range ret {
			if d.id == x {
				// log.Info("Block #%d is being written directly", x)
				if _, err := f.Write(d.content); err != nil {
					r.increaseErrors()
					r.log.Errorln(errors.New(op, fmt.Sprintf("error could not write content for block '%s' for file '%s': %v\n", blocks[x], path, err)))
				}
				r.tracker.ChangeCurr("size", len(d.content))
				r.tracker.IncreaseCurr("blocks")
				break
			}
			r.checkBuffer(b.Listen())
			blocksBuffer[d.id] = d.content
			r.tracker.IncreaseCurr("blocksBuffer")
		}
	}
	r.tracker.IncreaseCurr("files")
}

func (r *Recovery) checkBuffer(l *broadcast.Listener) {
	c, t, err := r.tracker.RawValues("blocksBuffer")
	if err != nil {
		log.Errorln(errors.New("recoveries.checkBuffer()", err))
	}
	for {
		if c < t {
			break
		}
		<-l.C
	}
	l.Close()
}
