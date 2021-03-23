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
)

// Run starts a recovery execution
func (r *Recovery) Run(lock *sync.Mutex) {
	op := "recovery.Run()"
	r.initLogger()
	r.startTracker()
	// CHECK THIS POINT OR THE END. IT IS IMPORTANT TO CONSIDER DATA DUPLICATION IF RECOVEEY IS STOPPED > STARTED

	// r.Status = Paused
	// log.Info("Recovery %d worker is waiting to start!", r.Data.ID)

	r.step = Metafiles
	if r.flowGate() {
		return
	}
	r.autoTrack()
	start := time.Now()
	log.Info("Starting recovery %d", r.Data.ID)
	r.log.Task("Starting recovery %d", r.Data.ID)
	tree, err := r.GetRecoveryTree()
	if err != nil {
		log.Errorln(errors.Extend(op, err))
		return
	}
	r.step = Files
	if r.flowGate() {
		return
	}

	if err := r.getFiles(tree); err != nil {
		log.Errorln(errors.Extend(op, err))
		return
	}
	if r.flowGate() {
		return
	}
	if err := r.Done(time.Since(start).Truncate(time.Second)); err != nil {
		log.Error(op, err)
	}
}

func (r *Recovery) PreCalculate() {
	// op := "recovery.PreCalculate()"
	// r.Status = Paused
	// log.Info("Recovery %d precalculation worker is waiting to start!", r.Data.ID)

	// r.startTracker()
	// r.step = Metafiles
	// if exit := r.stopGate(); exit != 0 {
	// 	return
	// }
	// r.initLogger()
	// log.Info("Starting precalculation %d", r.Data.ID)
	// r.log.Task("Starting precalculation %d", r.Data.ID)
	// tree, err := r.GetRecoveryTree()
	// if err != nil {
	// 	err = errors.Extend(op, err)
	// 	log.Errorln(err)
	// }
	// if exit := r.stopGate(); exit != 0 {
	// 	return
	// }

	// if err := r.Done(time.Since(start).Truncate(time.Second)); err != nil {
	// 	log.Error(op, err)
	// }
}

// GetLogin finds the server that the users belongs to
func GetLogin(addr, login string) (string, error) {
	op := "recovery.GetLogin()"

	uLogin := url.QueryEscape(login)
	query := fmt.Sprintf("%s?login=%s", addr, uLogin)
	req, err := http.NewRequest("GET", query, nil)
	if err != nil {
		return "", errors.New(op, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.New(op, err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorln(*resp)
		return "", errors.New(op, "Response status not OK")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New(op, err)
	}
	resp.Body.Close()

	s := string(body)
	out := strings.Trim(s, "\"")
	return out, nil
}
