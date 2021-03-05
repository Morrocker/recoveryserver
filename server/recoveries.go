package server

import (
	"encoding/json"
	"io/ioutil"
)

const recoveriesFile string = "currentRecoveries.json"

// WriteRecoveryJSON writes the recoveries data into a JSON
func (s *Server) WriteRecoveryJSON() {

	json, err := json.Marshal(s.Service)
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

// // StartWorkers asdfasd
// func (s *Server) StartWorkers() {
// 	s.Service.Director.StartWorkers()
// }
