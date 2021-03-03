package server

import (
	"encoding/json"
	"io/ioutil"

	"github.com/recoveryserver/recovery"
	"github.com/recoveryserver/utils"
)

const recoveriesFile string = "currentRecoveries.json"

func (s *Server) AddRecoveries(r recovery.RecoveryGroup) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	hash := utils.RandString(8)
	s.recoveries[hash] = r
	s.WriteRecoverJSON()
}

func (s *Server) WriteRecoverJSON() {

	json, err := json.Marshal(s.recoveries)
	if err != nil {
		// SEE error later
	}
	if err := ioutil.WriteFile(recoveriesFile, json, 0644); err != nil {
		// SEE error later
	}
}
