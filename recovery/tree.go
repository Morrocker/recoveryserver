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
	"github.com/morrocker/recoveryserver/config"
)

// MetaTree stores the information about a single node on a metafile tree. It indicates its own metafile data and all children it contains
type MetaTree struct {
	mf       *reposerver.Metafile
	children []*MetaTree
}

// GetRecoveryTree takes in a recovery data and returns a metafileTree
func GetRecoveryTree(r *Recovery) (*MetaTree, error) {
	errPath := "recovery.GetRecoveryTree()"

	metaChan := make(chan *MetaTree)
	var wg sync.WaitGroup
	logger.TaskV("Starting %d block workers", config.Data.MetafileWorkers)
	for i := 0; i < config.Data.MetafileWorkers; i++ {
		go getTree(metaChan, &r.Info, &wg)
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

func obsoletePrevious() error {
	// THIS WILL BE DONE VIA THE PROGRESS TRACKER !!!!!!!!! ALSO TAKE INTO ACCOUNT ALREADY DOWNLOADED FILES
	// checkSize += uint64(f.Size)
	// checkFiles++
	// checkBlocks++                             // fileblock
	// checkBlocks += uint64(f.Size) / (1024000) // 1 MB blocks
	// remainder := f.Size % 1024000
	// if remainder != 0 {
	// 	checkBlocks++
	// }

	// if checkFiles%100 == 0 {
	// 	fmt.Printf("Checking %d / ???. Checked %s already.\n", checkFiles, b2h(checkSize))
	// }
	return nil
}

func getLatestChildrenMaybe(id string) ([]*reposerver.Metafile, error) {
	// for retries := 0; retries < 5; retries++ {
	// 	req, err := http.NewRequest("GET", *server+"/api/latestChildren?id="+id+"&repo_id="+*repoID, nil)
	// 	if err != nil {
	// 		panic(fmt.Sprintf("could not create req for latest children of '%s': %v\n", id, err))
	// 	}

	// 	req.Header.Add("Cloner_key", config.ClonerKey)
	// 	resp, err := http.DefaultClient.Do(req)
	// 	if err != nil {
	// 		fmt.Printf("request for latest children of '%s' failed: %v\n", id, err)
	// 		continue
	// 	}

	// 	body, err := ioutil.ReadAll(resp.Body)
	// 	if err != nil {
	// 		fmt.Printf("could not read response for latest children of '%s': %v\n", id, err)
	// 		continue
	// 	}
	// 	resp.Body.Close()

	// 	var ret []*reposerver.Metafile
	// 	if err := json.Unmarshal(body, &ret); err != nil {
	// 		fmt.Printf("could not unmarshal response for latest children of '%s': %v\n%s\n", id, err, string(body))
	// 		continue
	// 	}

	// 	return ret, nil
	// }

	// return nil, fmt.Errorf("too many retries")
	return nil, fmt.Errorf("too many retries")
}

func getChildren(id string, version int64) ([]*reposerver.Metafile, error) {
	// for retries := 0; retries < 5; retries++ {
	// 	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/children?id=%s&version=%d&repo_id=%s", *server, id, version, *repoID), nil)
	// 	if err != nil {
	// 		panic(fmt.Sprintf("could not create req for children of '%s' version %d: %v\n", id, version, err))
	// 	}

	// 	req.Header.Add("Cloner_key", config.ClonerKey)
	// 	resp, err := http.DefaultClient.Do(req)
	// 	if err != nil {
	// 		fmt.Printf("request for children of '%s' version %d failed: %v\n", id, version, err)
	// 		continue
	// 	}

	// 	body, err := ioutil.ReadAll(resp.Body)
	// 	if err != nil {
	// 		fmt.Printf("could not read response for children of '%s' version %d: %v\n", id, version, err)
	// 		continue
	// 	}
	// 	resp.Body.Close()

	// 	var ret []*reposerver.Metafile
	// 	if err := json.Unmarshal(body, &ret); err != nil {
	// 		fmt.Printf("could not unmarshal response for children of '%s' version %d: %v\n", id, version, err)
	// 		continue
	// 	}

	// 	return ret, nil
	// }
	// return nil, fmt.Errorf("too many retries")
	return nil, fmt.Errorf("too many retries")
}

func getTree(mc chan *MetaTree, d *Data, wg *sync.WaitGroup) {
	errPath := "recovery.getTree()"
	for mt := range mc {
		wg.Add(1)
		if mt.mf.Type == reposerver.FolderType {
			if err := getLatestChildren(mt, d); err != nil {
				err := errors.Extend(errPath, err)
				logger.Error("%s", err)
				// TENEMOS QUE VER COMO MEJORAR ESTO
			}
			for i := range mt.children {
				mc <- mt.children[i]
			}
		}
		wg.Done()
	}
}

func getLatestChildren(t *MetaTree, d *Data) error {
	// errPath := "getLatestChildren()"
	// var errOut error
	for retries := 0; retries < 5; retries++ {
		newQuery := fmt.Sprintf("%sapi/latestsChilden?id=%s&repo_id=%s", d.Server, t.mf.Hash, d.Repository)
		req, err := http.NewRequest("GET", newQuery, nil)
		if err != nil {
			// spPath := fmt.Sprintf("%s Failed to obtain metafile %s", errPath, t.mf.Hash)
			// errOut = errors.New(spPath, err)
			continue
		}

		req.Header.Add("Cloner_key", "6f9536809658ceda12ae071783417d656e997409c83254dc85d391516abb0d7d66069bc77f68a0d30d77e261332a7e09429d96cd7909fbc00e46fef005d8737f")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			// spPath := fmt.Sprintf("%s Failed to obtain metafile %s", errPath, t.mf.Hash)
			// errOut = errors.New(spPath, err)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			// fmt.Printf("could not read response for latest children of '%s': %v\n", id, err)
			continue
		}
		resp.Body.Close()

		if err := json.Unmarshal(body, &t.children); err != nil {
			// fmt.Printf("could not unmarshal response for latest children of '%s': %v\n%s\n", id, err, string(body))
			continue
		}

		return nil
	}
	return fmt.Errorf("too many retries")
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

		req.Header.Add("Cloner_key", "MUSTFILLTHISURGENT")
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

		ret := &reposerver.Metafile{}
		if err := json.Unmarshal(body, ret); err != nil {
			errOut = errors.New(errPath+"Failed to obtain root metafile", err)
			continue
		}
		return ret, nil
	}

	return nil, errOut
}
