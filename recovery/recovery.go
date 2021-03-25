package recovery

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
)

// Run starts a recovery execution
func (r *Recovery) Run() {
	op := "recovery.Run()"
	log.Info("Starting recovery %d", r.Data.ID)
	r.initLogger()
	r.startTracker()
	r.log.Task("Starting recovery %d", r.Data.ID)
	go r.autoTrack()

	// CHECK THIS POINT OR THE END. IT IS IMPORTANT TO CONSIDER DATA DUPLICATION IF RECOVEEY IS STOPPED > STARTED

	r.Step(Metafiles)
	r.Start()
	if r.flowGate() {
		return
	}
	start := time.Now()
	tree, err := r.GetRecoveryTree()
	if err != nil {
		log.Errorln(errors.Extend(op, err))
		r.Cancel()
		return
	}
	r.tracker.StartAutoPrint(5 * time.Second)
	r.tracker.Print()
	r.Step(Files)
	if r.flowGate() {
		return
	}

	if err := r.getFiles(tree); err != nil {
		log.Errorln(errors.Extend(op, err))
		r.Cancel()
		return
	}
	r.tracker.Print()
	if r.flowGate() {
		return
	}
	if err := r.Done(time.Since(start).Truncate(time.Second)); err != nil {
		log.Error(op, err)
	}
}

func (r *Recovery) PreCalculate() {
	op := "recovery.Run()"
	log.Info("Starting recovery %d size calculation", r.Data.ID)
	r.initLogger()
	r.startTracker()
	r.log.Task("Starting recovery %d size calculation", r.Data.ID)
	go r.autoTrack()

	r.Step(Metafiles)
	r.Start()
	if r.flowGate() {
		return
	}
	_, err := r.GetRecoveryTree()
	if err != nil {
		log.Errorln(errors.Extend(op, err))
		r.Cancel()
		return
	}
	r.tracker.StartAutoPrint(5 * time.Second)
	r.tracker.Print()
	r.PreDone()
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
