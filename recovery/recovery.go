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
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/utils"
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
	log.Info("Recovery %s worker is waiting to start!", r.ID)
	r.stopGate()
	go func() {
		for {
			fc, ft, _ := r.SuperTracker.GetValues("files")
			bc, bt, _ := r.SuperTracker.GetValues("blocks")
			sc, st, _ := r.SuperTracker.GetValues("size")
			erro, _, _ := r.SuperTracker.GetValues("errors")
			if erro != 0 {
				log.Info("Files: %d/%d | Blocks: %d/%d | Size: %s/%s (Errors:%d)", fc, ft, bc, bt, utils.B2H(sc), utils.B2H(st), erro)
			} else {
				log.Info("Files: %d/%d | Blocks: %d/%d | Size: %s/%s", fc, ft, bc, bt, utils.B2H(sc), utils.B2H(st))
			}
			time.Sleep(time.Second)
		}
	}()

	tree, err := r.GetRecoveryTree()
	if err != nil {
		err = errors.Extend(errPath, err)
		log.Error("%s", err)
	}
	log.Notice("Metafiles Done")
	r.stopGate()

	if err := r.getFiles(tree); err != nil {
		err = errors.Extend(errPath, err)
	}
	log.Info("Recovery finished")
}

// RemoveFiles removes any recovered file from the destination location
func (r *Recovery) RemoveFiles() {
	log.Task("We are happily removing these files")
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
		log.Error("%s", *resp)
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
