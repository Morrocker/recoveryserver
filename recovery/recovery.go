package recovery

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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
	r.Status = Stop
	exitCode := 0
	for {
		logger.Info("Recovery %s worker is waiting to start!. BTW login is %s", r.ID, r.Info.Server)
		switch r.Status {
		case Start:
			goto Metafiles
		case Cancel:
			exitCode = exitAlone
			goto EndPoint
		}
		time.Sleep(10 * time.Second)
	}
Metafiles:
	for x := 0; x < 3; x++ {
		logger.Info("Finding metafiles for recovery %s", r.ID)
		time.Sleep(10 * time.Second)
	}
	for x := 0; x < 5; x++ {
		for {
			switch r.Status {
			case Start:
				goto Next
			case Cancel:
				exitCode = exitDelete
				goto EndPoint
			}
			time.Sleep(5 * time.Second)
		}
	Next:
		logger.Info("Recovering file #%d from recovery %s", x, r.ID)
		time.Sleep(5 * time.Second)
	}
EndPoint:
	if exitCode == exitDelete {
		r.RemoveFiles()
	}
	r.Status = Done
}

// LegacyRecovery recovers files using legacy blockserver remote
func (r *Recovery) LegacyRecovery() {}

// Recovery recovers files using current blockserver remote
func (r *Recovery) Recovery() {}

// RemoveFiles removes any recovered file from the destination location
func (r *Recovery) RemoveFiles() {
	logger.Task("We are happily removing these files")
}

// GetLogin finds the server that the users belongs to
func (r *Recovery) GetLogin() error {
	errPath := "recovery.GetLogin()"

	user := url.QueryEscape(r.Info.User)
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

	r.Info.Server = string(body)
	return nil
}
