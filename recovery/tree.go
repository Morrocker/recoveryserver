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
	"github.com/morrocker/log"
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
	wg.Add(config.Data.MetafileWorkers)
	for x := 0; x < config.Data.MetafileWorkers; x++ {
		go r.getChildMetaTree(tc, &wg)
	}

	mf, err := r.getMetafile()
	if err != nil {
		return nil, errors.New("recovery.GetRecoveryTree()", err)
	}

	recoveryTree := newMetaTree(mf)
	tc <- recoveryTree

	wg.Wait()
	if _, ok := <-tc; ok {
		log.Taskln("Shutting down lingering channel")
		close(tc)
	}

	r.log.Task("Metafile tree completed")
	return recoveryTree, nil
}

func (r *Recovery) getChildMetaTree(tc chan *MetaTree, wg *sync.WaitGroup) {
Outer:
	for mt := range tc {
		if r.flowGate() {
			break
		}

		if mt.mf.Type == reposerver.FolderType {
			children, err := r.getChildren(mt.mf.ID)
			if err != nil {
				r.log.Error("Couldnt retrieve metafile: %s", errors.Extend("recoveries.getChildMetaTree()", err))
				r.tracker.IncreaseCurr("metafiles")
				continue
			}

			for _, child := range children {
				if r.flowGate() {
					break Outer
				}
				childTree := newMetaTree(child)
				mt.addChildren(childTree)
				tc <- childTree
			}
			r.tracker.IncreaseCurr("metafiles")
			r.isDone(tc)
			continue
		}
		r.updateTrackerTotals(mt.mf.Size)
		r.tracker.IncreaseCurr("metafiles")
		r.isDone(tc)
	}
	wg.Done()
}

func (r *Recovery) getChildren(id string) ([]*reposerver.Metafile, error) {
	op := "recovery.getChildren()"
	r.log.Task("Getting children from " + id)
	var errOut error
	for retries := 0; retries < 5; retries++ {
		errOut = nil
		var newQuery string
		if r.Data.Deleted {
			newQuery = fmt.Sprintf("%sapi/latestChildren?id=%s&repo_id=%s", r.LoginServer, id, r.Data.Repository)
		} else {
			var version int
			if r.Data.Version == 0 {
				version = 999999999999
			} else {
				version = r.Data.Version
			}
			newQuery = fmt.Sprintf("%sapi/children?id=%s&version=%d&repo_id=%s", r.LoginServer, id, version, r.Data.Repository)
		}
		req, err := http.NewRequest("GET", newQuery, nil)
		if err != nil {
			errOut = errors.Extend(op, err)
			continue
		}

		req.Header.Add("Cloner_key", r.Data.ClonerKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errOut = errors.Extend(op, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errOut = errors.NewSimple("Status not ok")
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errOut = errors.Extend(op, err)
			continue
		}
		resp.Body.Close()

		var children []*reposerver.Metafile
		if err := json.Unmarshal(body, &children); err != nil {
			errOut = errors.Extend(op, err)
			continue
		}
		r.tracker.ChangeTotal("metafiles", len(children))
		return children, nil
	}
	errOut = errors.New(op, fmt.Sprintf("Failed to obtain metafile %s", errOut))
	return nil, errOut
}

func (r *Recovery) getMetafile() (*reposerver.Metafile, error) {
	op := "recovery.getMetafile()"
	r.log.Task("Getting root metafile")
	var errOut error
	for retries := 0; retries < 5; retries++ {
		errOut = nil
		newQuery := fmt.Sprintf("%sapi/metafile?id=%s&repo_id=%s", r.LoginServer, r.Data.Metafile, r.Data.Repository)
		req, err := http.NewRequest("GET", newQuery, nil)
		if err != nil {
			errOut = errors.Extend(op, err)
			continue
		}

		req.Header.Add("Cloner_key", r.Data.ClonerKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errOut = errors.Extend(op, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errOut = errors.NewSimple("Status not ok")
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errOut = errors.Extend(op, err)
			continue
		}
		resp.Body.Close()

		var ret reposerver.Metafile
		if err := json.Unmarshal(body, &ret); err != nil {
			errOut = errors.Extend(op, err)
			continue
		}
		r.tracker.ChangeTotal("metafiles", 1)
		return &ret, nil
	}
	errOut = errors.New(op, fmt.Sprintf("Failed to obtain metafile: %s", errOut))
	return nil, errOut
}

func (m *MetaTree) addChildren(mt *MetaTree) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.children = append(m.children, mt)
}

func (r *Recovery) isDone(tc chan *MetaTree) {
	c, t, err := r.tracker.RawValues("metafiles")
	if err != nil {
		log.Errorln("ERROR while getting metafiles tracker values")
	}
	if c == t {
		time.Sleep(5 * time.Second)
		close(tc)
	}
}
