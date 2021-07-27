package tree

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/clonercl/reposerver"
	"github.com/morrocker/errors"
	"github.com/morrocker/flow"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/recovery2/tracker"
	"github.com/morrocker/utils"
)

type Data struct {
	RootId     string
	Repository string
	Server     string
	ClonerKey  string
	Version    int
	Deleted    bool
	Exclusions map[string]bool
}

type Throttling struct {
	BuffSize int
	Workers  int
}

// MetaTree stores the information about a single node on a metafile tree. It indicates its own metafile data and all children it contains
type MetaTree struct {
	Mf       *reposerver.Metafile
	Children []*MetaTree
}

// NewMetaTree returns a metatree with the given reposerver.Metafile
func newMetaTree(mf *reposerver.Metafile) *MetaTree {
	tree := &MetaTree{Mf: mf}
	return tree
}

// GetRecoveryTree takes in a recovery data and returns a metafileTree
func GetRecoveryTree(data Data, tt Throttling, rt *tracker.RecoveryTracker, ctrl *flow.Controller) (*MetaTree, error) {
	log.Task("Starting metafile tree retrieval")

	if ctrl.Checkpoint() != 0 {
		return nil, nil
	}

	if len(data.Exclusions) > 0 {
		log.Info("List of metafiles (and their children) that will be excluded")
		for hash := range data.Exclusions {
			log.Infoln("ID: " + hash) // Temporary
		}
	}

	mt, err := getRootMetaTree(data, rt)
	if err != nil {
		return nil, errors.New("recovery.GetRecoveryTree()", err)
	}

	if ctrl.Checkpoint() != 0 {
		return nil, nil
	}

	tc, wg := startWorkers(data, tt, rt, ctrl)
	tc <- mt

	wg.Wait()

	if _, ok := <-tc; ok {
		close(tc)
	}
	if ctrl.Checkpoint() != 0 {
		return nil, nil
	}

	log.Info("Metafile tree completed")
	return mt, nil
}

func metaTreeWorker(data Data, tc chan *MetaTree, wg *sync.WaitGroup, tr *tracker.RecoveryTracker, ctrl *flow.Controller) {
	log.Task("Starting metaTreeWorker")
Outer:
	for mt := range tc {
		if ctrl.Checkpoint() != 0 {
			break
		}

		if mt.Mf.Type == reposerver.FolderType {
			childrenTrees, err := getChildren(mt.Mf.ID, data, tr)
			if err != nil {
				log.Error("Couldnt retrieve metafile: %s", errors.Extend("recoveries.getChildMetaTree()", err))
				isDone(1, 0, tc, tr)
				continue
			}

			for _, childTree := range childrenTrees {
				if ctrl.Checkpoint() != 0 {
					break Outer
				}
				mt.Children = append(mt.Children, childTree)

				tc <- childTree
			}
			isDone(1, 0, tc, tr)
			continue
		}
		isDone(1, 0, tc, tr)
	}
	wg.Done()
}

func getChildren(id string, data Data, rt *tracker.RecoveryTracker) ([]*MetaTree, error) {
	op := "recovery.getChildren()"
	log.Task("Getting children from " + utils.Trimmer(id))

	var errOut error
	for retries := 0; retries < 5; retries++ {
		errOut = nil
		var newQuery string
		if data.Deleted {
			newQuery = fmt.Sprintf("%sapi/latestChildren?id=%s&repo_id=%s", data.Server, id, data.Repository)
		} else {
			var version int
			if data.Version == 0 {
				version = 999999999999
			} else {
				version = data.Version
			}
			newQuery = fmt.Sprintf("%sapi/children?id=%s&version=%d&repo_id=%s", data.Server, id, version, data.Repository)
		}
		req, err := http.NewRequest("GET", newQuery, nil)
		if err != nil {
			errOut = errors.Extend(op, err)
			continue
		}

		req.Header.Add("Cloner_key", data.ClonerKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errOut = errors.Extend(op, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errOut = errors.Single("Status not ok")
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
		rt.Gauges["metafiles"].Total(int64(len(children)))
		trees := make([]*MetaTree, 0)
		for _, child := range children {
			childTree := newMetaTree(child)
			trees = append(trees, childTree)
		}
		return trees, nil
	}
	errOut = errors.New(op, fmt.Sprintf("Failed to obtain metafile %s", errOut))
	return nil, errOut
}

func startWorkers(data Data, tt Throttling, rt *tracker.RecoveryTracker, ctrl *flow.Controller) (chan *MetaTree, *sync.WaitGroup) {
	log.Task("Starting %d metaTree workers", tt.Workers)

	wg := &sync.WaitGroup{}
	tc := make(chan *MetaTree, tt.BuffSize)
	wg.Add(tt.Workers)
	for x := 0; x < tt.Workers; x++ {
		go metaTreeWorker(data, tc, wg, rt, ctrl)
	}

	return tc, wg
}

func getRootMetaTree(data Data, rt *tracker.RecoveryTracker) (mt *MetaTree, errOut error) {
	op := "recovery.getMetafile()"
	log.Task("Getting root metafile %s", data.RootId)

	for retries := 0; retries < 3; retries++ {
		errOut = nil
		req, err := http.NewRequest(
			"GET",
			fmt.Sprintf("%sapi/metafile?id=%s&repo_id=%s", data.Server, data.RootId, data.Repository),
			nil,
		)
		if err != nil {
			errOut = errors.Extend(op, err)
			continue
		}

		req.Header.Add("Cloner_key", data.ClonerKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errOut = errors.Extend(op, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errOut = errors.Single("Status not ok")
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errOut = errors.Extend(op, err)
			continue
		}
		resp.Body.Close()

		mf := &reposerver.Metafile{}
		if err := json.Unmarshal(body, &mf); err != nil {
			errOut = errors.Extend(op, err)
			continue
		}
		rt.Gauges["metafiles"].Total(1)
		return newMetaTree(mf), nil
	}
	errOut = errors.New(op, fmt.Sprintf("Failed to obtain metafile: %s", errOut))
	return nil, errOut
}

func isDone(curr, tot int, tc chan *MetaTree, rt *tracker.RecoveryTracker) {
	t, err := rt.Gauges["metafiles"].Total(int64(tot))
	if err != nil {
		log.Error("Error updating metafiles total")
	}
	c, err := rt.Gauges["metafiles"].Current(int64(tot))
	if err != nil {
		log.Error("Error updating current total")
	}
	if c == t {
		time.Sleep(time.Second)
		close(tc)
	}
}
