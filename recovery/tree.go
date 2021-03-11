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
	tracker "github.com/morrocker/progress-tracker"
	"github.com/morrocker/recoveryserver/config"
)

// MetaTree stores the information about a single node on a metafile tree. It indicates its own metafile data and all children it contains
type MetaTree struct {
	mf       *reposerver.Metafile
	children []*MetaTree
	lock     sync.Mutex
}

// Queues asdf as
type Queues struct {
	lock          sync.Mutex
	ToDoQueue     []*MetaTree
	NextToDoQueue []*MetaTree
}

var queue Queues

// GetRecoveryTree takes in a recovery data and returns a metafileTree
func GetRecoveryTree(d *Data, t *tracker.SuperTracker) (*MetaTree, error) {
	errPath := "recovery.GetRecoveryTree()"
	logger.Notice("Starting metafile tree retrieval")

	var wg sync.WaitGroup
	tc := make(chan *MetaTree)
	if len(d.Exclusions) > 0 {
		logger.Info("List of metafiles (and their children) that will be excluded")
		for hash := range d.Exclusions {
			logger.Info("ID: %s", hash)
		}
	}

	for x := 0; x < config.Data.MetafileWorkers; x++ {
		go getChildMetaTree(tc, d, &wg, t)
	}

	logger.Task("Getting root metafile")
	mf, err := getMetafile(d)
	if err != nil {
		err := errors.New(errPath, err)
		return nil, err
	}

	recoveryTree := NewMetaTree(mf)
	queue.addTree(recoveryTree)

	for {
		queue.restart()
		for _, tree := range queue.ToDoQueue {
			tc <- tree
		}
		time.Sleep(1 * time.Second)
		wg.Wait()
		if len(queue.NextToDoQueue) <= 0 {
			break
		}
	}

	return recoveryTree, nil
}

func getChildMetaTree(tc chan *MetaTree, d *Data, wg *sync.WaitGroup, t *tracker.SuperTracker) {
	errPath := "recoveries.getChildMetaTree()"
	for mt := range tc {
		wg.Add(1)

		if d.Exclusions[mt.mf.ID] {
			fmt.Printf("%s excluded\n", mt.mf.Name)
			wg.Done()
			continue
		}

		if mt.mf.Type == reposerver.FolderType {
			children, err := getChildren(mt.mf.ID, d)
			if err != nil {
				err = errors.Extend(errPath, err)
				logger.Error("Couldnt retrieve metafile: %s", err)
			}

			for _, child := range children {
				childTree := NewMetaTree(child)
				mt.AddChildren(childTree)
				queue.addTree(childTree)
			}
			wg.Done()
			continue
		}
		updateTracker(mt.mf.Size, t)
		wg.Done()
	}
}

func getChildren(id string, d *Data) ([]*reposerver.Metafile, error) {
	errPath := "getLatestChildren()"
	var errOut error
	for retries := 0; retries < 5; retries++ {
		errOut = nil
		var newQuery string
		if d.Deleted {
			newQuery = fmt.Sprintf("%sapi/latestsChilden?id=%s&repo_id=%s", d.Server, id, d.Repository)
		} else {
			var version int
			if d.Version == 0 {
				version = 999999999
			} else {
				version = d.Version
			}
			newQuery = fmt.Sprintf("%sapi/children?id=%s&version=%d&repo_id=%s", d.Server, id, version, d.Repository)
		}
		req, err := http.NewRequest("GET", newQuery, nil)
		if err != nil {
			spPath := fmt.Sprintf("%s Failed to obtain metafile %s", errPath, id)
			errOut = errors.New(spPath, err)
			continue
		}

		req.Header.Add("Cloner_key", d.ClonerKey)
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

func getMetafile(d *Data) (*reposerver.Metafile, error) {
	errPath := "recovery.getMetafile()"
	var errOut error
	for retries := 0; retries < 5; retries++ {
		newQuery := fmt.Sprintf("%sapi/metafile?id=%s&repo_id=%s", d.Server, d.Metafile, d.Repository)
		req, err := http.NewRequest("GET", newQuery, nil)
		if err != nil {
			errOut = errors.New(errPath+"Failed to obtain root metafile", err)
			continue
		}

		req.Header.Add("Cloner_key", d.ClonerKey)
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

func updateTracker(size int64, t *tracker.SuperTracker) {
	blocks := int64(1)                // fileblock
	blocks += (int64(size) / 1024000) // 1 MB blocks
	remainder := size % 1024000
	if remainder != 0 {
		blocks++
	}
	t.ChangeTotal("size", size)
	t.ChangeTotal("files", 1)
	t.ChangeTotal("blocks", blocks)
}

// NewMetaTree asfas
func NewMetaTree(mf *reposerver.Metafile) *MetaTree {
	tree := &MetaTree{mf: mf}
	return tree

}

// AddChildren adfa fa
func (m *MetaTree) AddChildren(mt *MetaTree) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.children = append(m.children, mt)
}

func (q *Queues) addTree(mt *MetaTree) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.NextToDoQueue = append(q.NextToDoQueue, mt)
}

func (q *Queues) restart() {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.ToDoQueue = q.NextToDoQueue
	q.NextToDoQueue = []*MetaTree{}
}
