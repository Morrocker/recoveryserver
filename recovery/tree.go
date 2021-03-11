package recovery

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/clonercl/reposerver"
	"github.com/morrocker/errors"
	"github.com/morrocker/logger"
	"github.com/morrocker/recoveryserver/config"
)

// MetaTree stores the information about a single node on a metafile tree. It indicates its own metafile data and all children it contains
type MetaTree struct {
	mf       *reposerver.Metafile
	children []*MetaTree
	path     string
	lock     sync.Mutex
}

// Queues asdf as
type metaQueue struct {
	lock          sync.Mutex
	ToDoQueue     []*MetaTree
	NextToDoQueue []*MetaTree
}

var mq metaQueue = metaQueue{}

// NewMetaTree asfas
func NewMetaTree(mf *reposerver.Metafile) *MetaTree {
	tree := &MetaTree{mf: mf}
	return tree
}

// GetRecoveryTree takes in a recovery data and returns a metafileTree
func (r *Recovery) GetRecoveryTree() (*MetaTree, error) {
	errPath := "recovery.GetRecoveryTree()"
	logger.Notice("Starting metafile tree retrieval")

	var wg sync.WaitGroup
	tc := make(chan *MetaTree)
	if len(r.Data.Exclusions) > 0 {
		logger.Info("List of metafiles (and their children) that will be excluded")
		for hash := range r.Data.Exclusions {
			logger.Info("ID: %s", hash)
		}
	}

	for x := 0; x < config.Data.MetafileWorkers; x++ {
		go r.getChildMetaTree(tc, &wg)
	}

	logger.Task("Getting root metafile")
	mf, err := r.getRootMetafile()
	if err != nil {
		err := errors.New(errPath, err)
		return nil, err
	}

	recoveryTree := NewMetaTree(mf)
	mq.addTree(recoveryTree)

	for {
		mq.restart()
		for _, tree := range mq.ToDoQueue {
			tc <- tree
		}
		time.Sleep(1 * time.Second)
		wg.Wait()
		if len(mq.NextToDoQueue) <= 0 {
			break
		}
	}

	return recoveryTree, nil
}

func (r *Recovery) getChildMetaTree(tc chan *MetaTree, wg *sync.WaitGroup) {
	errPath := "recoveries.getChildMetaTree()"
	for mt := range tc {
		r.stopGate()
		wg.Add(1)

		if r.Data.Exclusions[mt.mf.ID] {
			fmt.Printf("%s excluded\n", mt.mf.Name)
			wg.Done()
			continue
		}

		if mt.mf.Type == reposerver.FolderType {
			children, err := r.getChildren(mt.mf.ID)
			if err != nil {
				err = errors.Extend(errPath, err)
				logger.Error("Couldnt retrieve metafile: %s", err)
			}

			for _, child := range children {
				childTree := NewMetaTree(child)
				mt.AddChildren(childTree)
				mq.addTree(childTree)
			}
			wg.Done()
			continue
		}
		r.updateTrackerTotals(mt.mf.Size)
		wg.Done()
	}
}

func (r *Recovery) getChildren(id string) ([]*reposerver.Metafile, error) {
	errPath := "getLatestChildren()"
	var errOut error
	for retries := 0; retries < 5; retries++ {
		errOut = nil
		var newQuery string
		if r.Data.Deleted {
			newQuery = fmt.Sprintf("%sapi/latestsChilden?id=%s&repo_id=%s", r.Data.Server, id, r.Data.Repository)
		} else {
			var version int
			if r.Data.Version == 0 {
				version = 999999999
			} else {
				version = r.Data.Version
			}
			newQuery = fmt.Sprintf("%sapi/children?id=%s&version=%d&repo_id=%s", r.Data.Server, id, version, r.Data.Repository)
		}
		req, err := http.NewRequest("GET", newQuery, nil)
		if err != nil {
			spPath := fmt.Sprintf("%s Failed to obtain metafile %s", errPath, id)
			errOut = errors.New(spPath, err)
			continue
		}

		req.Header.Add("Cloner_key", r.Data.ClonerKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			spPath := fmt.Sprintf("%s Failed to obtain metafile %s", errPath, id)
			errOut = errors.New(spPath, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errOut = errors.New(errPath+"Failed to obtain root metafile", "Response status code not OK")
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			spPath := fmt.Sprintf("%s Failed to obtain metafile %s", errPath, id)
			errOut = errors.New(spPath, err)
			continue
		}
		resp.Body.Close()

		var children []*reposerver.Metafile
		if err := json.Unmarshal(body, &children); err != nil {
			spPath := fmt.Sprintf("%s Failed to obtain metafile %s", errPath, id)
			errOut = errors.New(spPath, err)
			continue
		}
		return children, nil
	}
	return nil, errOut
}

func (r *Recovery) getRootMetafile() (*reposerver.Metafile, error) {
	errPath := "recovery.getMetafile()"
	var errOut error
	for retries := 0; retries < 5; retries++ {
		newQuery := fmt.Sprintf("%sapi/metafile?id=%s&repo_id=%s", r.Data.Server, r.Data.Metafile, r.Data.Repository)
		req, err := http.NewRequest("GET", newQuery, nil)
		if err != nil {
			errOut = errors.New(errPath+"Failed to obtain root metafile", err)
			continue
		}

		req.Header.Add("Cloner_key", r.Data.ClonerKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errOut = errors.New(errPath+"Failed to obtain root metafile", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errOut = errors.New(errPath+"Failed to obtain root metafile", "Response status code not OK")
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errOut = errors.New(errPath+"Failed to obtain root metafile", err)
			continue
		}
		resp.Body.Close()

		var ret reposerver.Metafile
		if err := json.Unmarshal(body, &ret); err != nil {
			errOut = errors.New(errPath+"Failed to obtain root metafile", err)
			continue
		}
		return &ret, nil
	}

	return nil, errOut
}

// AddChildren adfa fa
func (m *MetaTree) AddChildren(mt *MetaTree) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.children = append(m.children, mt)
}

func (q *metaQueue) addTree(mt *MetaTree) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.NextToDoQueue = append(q.NextToDoQueue, mt)
}

func (q *metaQueue) restart() {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.ToDoQueue = q.NextToDoQueue
	q.NextToDoQueue = []*MetaTree{}
}
