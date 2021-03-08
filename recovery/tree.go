package recovery

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

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
}

// GetRecoveryTree takes in a recovery data and returns a metafileTree
func GetRecoveryTree(r *Recovery, t *tracker.SuperTracker) (*MetaTree, error) {
	errPath := "recovery.GetRecoveryTree()"

	metaChan := make(chan *MetaTree)
	var wg sync.WaitGroup
	logger.TaskV("Starting %d block workers", config.Data.MetafileWorkers)
	for i := 0; i < config.Data.MetafileWorkers; i++ {
		go getTree(metaChan, &r.Info, t, &wg)
	}
	if len(r.Info.Exclusions) > 0 {
		logger.Info("List of metafiles (and their children) that will be excluded")
		for _, hash := range r.Info.Exclusions {
			logger.Info("ID: %s", hash)
		}
	}

	var recoveryTree *MetaTree

	mf, err := getMetafile(r.Info)
	if err != nil {
		err := errors.New(errPath, err)
		return recoveryTree, err
	}
	recoveryTree.mf = mf
	metaChan <- recoveryTree
	wg.Wait()
	close(metaChan)
	return recoveryTree, nil
}

func getTree(mc chan *MetaTree, d *Data, t *tracker.SuperTracker, wg *sync.WaitGroup) {
	errPath := "recovery.getTree()"
	for mt := range mc {
		wg.Add(1)
		if mt.mf.Type == reposerver.FolderType {
			children, err := getChildren(mt, d)
			if err != nil {
				err := errors.Extend(errPath, err)
				logger.Error("%s", err)
				// TENEMOS QUE VER COMO MEJORAR ESTO
			} else {
				for _, child := range children {
					childTree := &MetaTree{mf: child}
					mt.children = append(mt.children, childTree)
				}
				for i := range mt.children {
					mc <- mt.children[i]
				}
			}
		}

		updateTracker(mt.mf.Size, t)
		wg.Done()
	}
}

func getChildren(t *MetaTree, d *Data) ([]*reposerver.Metafile, error) {
	errPath := "getLatestChildren()"
	var errOut error
	for retries := 0; retries < 5; retries++ {
		var newQuery string
		if d.Deleted {
			newQuery = fmt.Sprintf("%sapi/latestsChilden?id=%s&repo_id=%s", d.Server, t.mf.Hash, d.Repository)
		} else {
			newQuery = fmt.Sprintf("%sapi/children?id=%s&version=%d&repo_id=%s", d.Server, t.mf.Hash, d.Version, d.Repository)
		}
		req, err := http.NewRequest("GET", newQuery, nil)
		if err != nil {
			spPath := fmt.Sprintf("%s Failed to obtain metafile %s", errPath, t.mf.Hash)
			errOut = errors.New(spPath, err)
			continue
		}

		req.Header.Add("Cloner_key", d.ClonerKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			spPath := fmt.Sprintf("%s Failed to obtain metafile %s", errPath, t.mf.Hash)
			errOut = errors.New(spPath, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errOut = errors.New(errPath+"Failed to obtain root metafile", "Response status code not OK")
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			spPath := fmt.Sprintf("%s Failed to obtain metafile %s", errPath, t.mf.Hash)
			errOut = errors.New(spPath, err)
			continue
		}
		resp.Body.Close()

		var ret []*reposerver.Metafile
		if err := json.Unmarshal(body, &ret); err != nil {
			spPath := fmt.Sprintf("%s Failed to obtain metafile %s", errPath, t.mf.Hash)
			errOut = errors.New(spPath, err)
			continue
		}

		return ret, nil
	}
	return nil, errOut
}

func getMetafile(d Data) (*reposerver.Metafile, error) {
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

		var ret *reposerver.Metafile
		if err := json.Unmarshal(body, ret); err != nil {
			errOut = errors.New(errPath+"Failed to obtain root metafile", err)
			continue
		}
		return ret, nil
	}

	return nil, errOut
}

func updateTracker(size int64, t *tracker.SuperTracker) {
	// checkSize += uint64(f.Size)
	// checkFiles++
	blocks := int64(1)                // fileblock
	blocks += int64(size) / (1024000) // 1 MB blocks
	remainder := size % 1024000
	if remainder != 0 {
		blocks++
	}
	t.ChangeTotal("size", size)
	t.ChangeTotal("files", 1)
	t.ChangeTotal("blocks", blocks)
	// if checkFiles%100 == 0 {
	// 	fmt.Printf("Checking %d / ???. Checked %s already.\n", checkFiles, b2h(checkSize))
	// }
}
