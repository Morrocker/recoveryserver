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
	"github.com/morrocker/recoveryserver/config"
)

// MetaTree stores the information about a single node on a metafile tree. It indicates its own metafile data and all children it contains
type MetaTree struct {
	mf       *reposerver.Metafile
	children []*MetaTree
	path     string
	lock     sync.Mutex
}

// NewMetaTree asfas
func newMetaTree(mf *reposerver.Metafile) *MetaTree {
	tree := &MetaTree{mf: mf}
	return tree
}

// GetRecoveryTree takes in a recovery data and returns a metafileTree
func (r *Recovery) GetRecoveryTree() (*MetaTree, error) {
	op := "recovery.GetRecoveryTree()"
	r.log.Task("Starting metafile tree retrieval")

	var wg sync.WaitGroup

	if len(r.Data.Exclusions) > 0 {
		r.log.InfoV("List of metafiles (and their children) that will be excluded")
		for hash := range r.Data.Exclusions {
			r.log.InfoV("ID: " + hash)
		}
	}

	r.log.TaskV("Opening workers channel with buffer %d", config.Data.MetafilesBuffSize)
	tc := make(chan *MetaTree, config.Data.MetafilesBuffSize)
	r.log.TaskV("Starting %d metafile workers", config.Data.MetafileWorkers)
	for x := 0; x < config.Data.MetafileWorkers; x++ {
		go r.getChildMetaTree(tc, &wg)
	}

	mf, err := r.getMetafile()
	if err != nil {
		return nil, errors.New(op, err)
	}

	recoveryTree := newMetaTree(mf)
	tc <- recoveryTree

	time.Sleep(time.Second)
	wg.Wait()
	close(tc)

	return recoveryTree, nil
}

func (r *Recovery) getChildMetaTree(tc chan *MetaTree, wg *sync.WaitGroup) {
	op := "recoveries.getChildMetaTree()"
	for mt := range tc {
		if r.flowGate() {
			wg.Done()
			return
		}
		wg.Add(1)

		if mt.mf.Type == reposerver.FolderType {
			children, err := r.getChildren(mt.mf.ID)
			if err != nil {
				r.log.Error("Couldnt retrieve metafile: %s", errors.Extend(op, err))
			}

			for _, child := range children {
				if r.flowGate() {
					wg.Done()
					return
				}
				childTree := newMetaTree(child)
				mt.addChildren(childTree)
				tc <- childTree
			}
			wg.Done()
			continue
		}
		r.updateTrackerTotals(mt.mf.Size)
		wg.Done()
	}
}

func (r *Recovery) getChildren(id string) ([]*reposerver.Metafile, error) {
	r.log.Task("Getting children from " + id)
	var err error
	for retries := 0; retries < 5; retries++ {
		err = nil
		var newQuery string
		if r.Data.Deleted {
			newQuery = fmt.Sprintf("%sapi/latestsChilden?id=%s&repo_id=%s", r.Data.Server, id, r.Data.Repository)
		} else {
			var version int
			if r.Data.Version == 0 {
				version = 999999999999
			} else {
				version = r.Data.Version
			}
			newQuery = fmt.Sprintf("%sapi/children?id=%s&version=%d&repo_id=%s", r.Data.Server, id, version, r.Data.Repository)
		}
		req, err := http.NewRequest("GET", newQuery, nil)
		if err != nil {
			continue
		}

		req.Header.Add("Cloner_key", r.Data.ClonerKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			err = errors.NewSimple("Status not ok")
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		resp.Body.Close()

		var children []*reposerver.Metafile
		if err := json.Unmarshal(body, &children); err != nil {
			continue
		}
		return children, nil
	}
	err = errors.New("recovery.getLatestChildren()", fmt.Sprintf("Failed to obtain metafile %s", err))
	return nil, err
}

func (r *Recovery) getMetafile() (*reposerver.Metafile, error) {
	r.log.Task("Getting root metafile")
	var err error
	for retries := 0; retries < 5; retries++ {
		err = nil
		newQuery := fmt.Sprintf("%sapi/metafile?id=%s&repo_id=%s", r.Data.Server, r.Data.Metafile, r.Data.Repository)
		req, err := http.NewRequest("GET", newQuery, nil)
		if err != nil {
			continue
		}

		req.Header.Add("Cloner_key", r.Data.ClonerKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			err = errors.NewSimple("Status not ok")
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		resp.Body.Close()

		var ret reposerver.Metafile
		if err := json.Unmarshal(body, &ret); err != nil {
			continue
		}
		return &ret, nil
	}
	err = errors.New("recovery.getMetafile()", fmt.Sprintf("Failed to obtain metafile %s", err))
	return nil, err
}

func (m *MetaTree) addChildren(mt *MetaTree) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.children = append(m.children, mt)
}
