package recovery

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/logger"
	"github.com/morrocker/recoveryserver/config"
)

const (
	exitNoCode = iota
	exitAlone
	exitDelete
)

// Run starts a recovery execution
func (r *Recovery) Run(lock *sync.Mutex) {
	errPath := "recovery.Run()"
	r.Status = Stop
	logger.Info("Recovery %s worker is waiting to start!. BTW login is %s and cloner key is %s", r.ID, r.Data.Server, r.Data.ClonerKey)
	for {
		switch r.Status {
		case Start:
			goto Metafiles
			// case Cancel:
			// 	exitCode = exitAlone
			// 	goto EndPoint
		}
		time.Sleep(10 * time.Second)
	}
Metafiles:
	// go func() {
	// 	for {
	// 		cf, tf, _ := r.SuperTracker.GetValues("files")
	// 		cb, tb, _ := r.SuperTracker.GetValues("blocks")
	// 		cs, ts, _ := r.SuperTracker.GetValues("size")
	// 		logger.Info("Files: %d/%d | Blocks: %d/%d | Size: %d/%d", cf, tf, cb, tb, cs, ts)
	// 		time.Sleep(5 * time.Second)
	// 	}
	// }()
	tree, err := GetRecoveryTree(r.Data, r.SuperTracker)
	if err != nil {
		err = errors.Extend(errPath, err)
		logger.Error("%s", err)
	}
	// REMOVE LATER
	logger.Notice("Metafiles Done")
	for {
		switch r.Status {
		case Start:
			break
		}
		time.Sleep(5 * time.Second)
		break
	}

	if err := r.getFiles(tree); err != nil {
		err = errors.Extend(errPath, err)
	}
	logger.Info("Recovery finished")
}

// RemoveFiles removes any recovered file from the destination location
func (r *Recovery) RemoveFiles() {
	logger.Task("We are happily removing these files")
}

// GetLogin finds the server that the users belongs to
func (r *Recovery) GetLogin() error {
	errPath := "recovery.GetLogin()"

	user := url.QueryEscape(r.Data.User)
	query := fmt.Sprintf("%s?login=%s", config.Data.LoginAddr, user)
	req, err := http.NewRequest("GET", query, nil)
	if err != nil {
		return errors.New(errPath, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New(errPath, err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Error("%s", *resp)
		return errors.New(errPath, "Response status not OK")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New(errPath, err)
	}
	resp.Body.Close()

	s := string(body)
	out := strings.Trim(s, "\"")
	r.Data.Server = out
	return nil
}
