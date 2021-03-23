package director

import (
	"encoding/json"
	"io/ioutil"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
)

// WriteRecoveryJSON writes the recoveries data into a JSON
func (d *Director) writeRecoveryJSON() error {
	log.Task("Writing Recovery JSON")
	op := "director.WriteRecoveryJSON()"
	json, err := json.MarshalIndent(d.Recoveries, "", "  ")
	if err != nil {
		return errors.New(op, err)
	}
	if err := ioutil.WriteFile(config.Data.RecoveriesJSON, json, 0644); err != nil {
		return errors.New(op, err)
	}
	return nil
}

// ReadRecoveryJSON reads in the recoveries data JSON file
func (d *Director) readRecoveryJSON() error {
	log.Task("Reading Recovery JSON")
	op := "director.ReadRecoveryJSON()"
	jsonBytes, err := ioutil.ReadFile(config.Data.RecoveriesJSON)
	if err != nil {
		return errors.New(op, err)
	}
	if err := json.Unmarshal(jsonBytes, &d.Recoveries); err != nil {
		return errors.New(op, err)
	}
	return nil
}
