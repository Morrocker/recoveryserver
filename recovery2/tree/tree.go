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
	"github.com/morrocker/log"
	tracker "github.com/morrocker/progress-tracker"
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
	// path     string
	// blockslist *BlocksList
}

// NewMetaTree asfas
func newMetaTree(mf *reposerver.Metafile) *MetaTree {
	tree := &MetaTree{Mf: mf}
	return tree
}

// GetRecoveryTree takes in a recovery data and returns a metafileTree
func GetRecoveryTree(data Data, t Throttling, tr *tracker.SuperTracker) (*MetaTree, error) {
	log.Task("Starting metafile tree retrieval")

	if len(data.Exclusions) > 0 {
		log.Info("List of metafiles (and their children) that will be excluded")
		for hash := range data.Exclusions {
			log.Infoln("ID: " + hash) // Temporary
		}
	}

	mt, err := getRootMetaTree(data, tr)
	if err != nil {
		return nil, errors.New("recovery.GetRecoveryTree()", err)
	}

	tc, wg := startWorkers(data, t, tr)
	tc <- mt

	wg.Wait()

	if _, ok := <-tc; ok {
		// log.Taskln("Shutting down lingering channel")
		close(tc)
	}

	log.Info("Metafile tree completed")
	return mt, nil
}

func metaTreeWorker(data Data, tc chan *MetaTree, wg *sync.WaitGroup, tr *tracker.SuperTracker) {
	log.Taskln("Starting metaTreeWorker")
	// Outer:
	for mt := range tc {
		// if r.flowGate() {
		// 	break
		// }

		if mt.Mf.Type == reposerver.FolderType {
			childrenTrees, err := getChildren(mt.Mf.ID, data, tr)
			if err != nil {
				log.Error("Couldnt retrieve metafile: %s", errors.Extend("recoveries.getChildMetaTree()", err))
				tr.IncreaseCurr("metafiles")
				continue
			}

			for _, childTree := range childrenTrees {
				// if r.flowGate() {
				// 	break Outer
				// }
				mt.Children = append(mt.Children, childTree)

				tc <- childTree
			}
			tr.IncreaseCurr("metafiles")
			isDone(tc, tr)
			continue
		}
		// r.updateTrackerTotals(mt.mf.Size)
		tr.IncreaseCurr("metafiles")
		isDone(tc, tr)
	}
	wg.Done()
}

func getChildren(id string, data Data, tr *tracker.SuperTracker) ([]*MetaTree, error) {
	op := "recovery.getChildren()"
	log.Task("Getting children from " + id)

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
		tr.ChangeTotal("metafiles", len(children))
		trees := make([]*MetaTree, len(children))
		for i, child := range children {
			trees[i] = newMetaTree(child)
		}
		return trees, nil
	}
	errOut = errors.New(op, fmt.Sprintf("Failed to obtain metafile %s", errOut))
	return nil, errOut
}

func startWorkers(data Data, t Throttling, tr *tracker.SuperTracker) (chan *MetaTree, *sync.WaitGroup) {
	log.Taskln("Starting metaTree workers")

	wg := &sync.WaitGroup{}
	// r.log.TaskV("Opening workers channel with buffer %d", config.Data.MetafilesBuffSize)
	tc := make(chan *MetaTree, t.BuffSize)
	// r.log.TaskV("Starting %d metafile workers", config.Data.MetafileWorkers)
	wg.Add(t.Workers)
	for x := 0; x < t.Workers; x++ {
		go metaTreeWorker(data, tc, wg, tr)
	}

	return tc, wg
}

func getRootMetaTree(data Data, tr *tracker.SuperTracker) (mt *MetaTree, errOut error) {
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
			errOut = errors.NewSimple("Status not ok")
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
		tr.ChangeTotal("metafiles", 1)
		return newMetaTree(mf), nil
	}
	errOut = errors.New(op, fmt.Sprintf("Failed to obtain metafile: %s", errOut))
	return nil, errOut
}

func isDone(tc chan *MetaTree, tr *tracker.SuperTracker) {
	curr, tot, err := tr.RawValues("metafiles")
	if err != nil {
		log.Errorln("ERROR while getting metafiles tracker values") // Temporary???
	}
	if curr == tot {
		time.Sleep(time.Second)
		close(tc)
	}
}
