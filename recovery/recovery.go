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
	r.initLogger()
	r.initSpdTrack()
	r.Status = Stop
	log.Info("Recovery %s worker is waiting to start!", r.ID)
	r.stopGate()
	log.Info("Starting recovery %s", r.ID)
	r.Log.Task("Starting recovery %s", r.ID)
	start := time.Now()
	r.SuperTracker.InitSpdRate("size", 40)
	r.SuperTracker.SetEtaTracker("size")
	r.SuperTracker.SetProgressFunction("size", utils.B2H)

	r.SuperTracker.StartAutoPrint()
	tree, err := r.GetRecoveryTree()
	if err != nil {
		err = errors.Extend(errPath, err)
		log.Error("%s", err)
	}
	r.stopGate()

	r.SuperTracker.StartAutoMeasure("size", 5)
	if err := r.getFiles(tree); err != nil {
		err = errors.Extend(errPath, err)
	}
	r.SuperTracker.StopAutoMeasure("size")
	r.SuperTracker.StopAutoPrint()
	finish := time.Since(start).Truncate(time.Second)
	rate, err := r.SuperTracker.GetTrueProgressRate("size")
	if err != nil {
		log.Alert("%s", errors.Extend(errPath, err))
	}
	log.Info("Recovery finished in %s with an average download rate of %sps", finish, rate)
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
