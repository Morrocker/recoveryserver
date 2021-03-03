package server

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/recoveryserver/recovery"
)

const recoveriesFile string = "currentRecoveries.json"

// AddRecoveries adds a new set of recoveries to the server
// func (s *Server) AddRecoveries(r *recovery.Group) {
// s.Lock.Lock()
// defer s.Lock.Unlock()
// hash := utils.RandString(8)
// s.recoveries[hash] = r
// s.WriteRecoveryJSON()
// }

// AddRecovery adds a new recovery to a given set
func (s *Server) AddRecovery(h string, r recovery.Recovery) {
	// s.Lock.Lock()
	// defer s.Lock.Unlock()
	// hash := s.recoveries[h].AddRecovery(r)
	// logger.Info("we must return the hash! %s", hash)
	// s.WriteRecoveryJSON()
}

// WriteRecoveryJSON writes the recoveries data into a JSON
func (s *Server) WriteRecoveryJSON() {

	json, err := json.Marshal(s.service.recoveries)
	if err != nil {
		// SEE error later
	}
	if err := ioutil.WriteFile(recoveriesFile, json, 0644); err != nil {
		// SEE error later
	}
}

// ReadRecoveryJSON reads in the recoveries data JSON file
func (s *Server) ReadRecoveryJSON() {

}

func (s *Server) StartWorkers() {
	for {
		s.Lock.Lock()
		for key, recover := range s.service.recoveries {
			if recover.Status == recovery.Queue {
				go s.service.recoveries[key].RunRecovery()
			}
		}
		s.Lock.Unlock()
		time.Sleep(10 * time.Second)
	}
}
