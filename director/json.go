package director

import (
	"encoding/json"
	"io/ioutil"

	"github.com/morrocker/errors"
	"github.com/morrocker/logger"
	"github.com/morrocker/recoveryserver/config"
)

// WriteRecoveryJSON writes the recoveries data into a JSON
func (d *Director) WriteRecoveryJSON() error {
	logger.TaskV("Writing Recovery JSON")
	errPath := "director.WriteRecoveryJSON()"
	json, err := json.MarshalIndent(d.Recoveries, "", "  ")
	if err != nil {
		return errors.New(errPath, err)
	}
	if err := ioutil.WriteFile(config.Data.RecoveriesJSON, json, 0644); err != nil {
		return errors.New(errPath, err)
	}
	return nil
}

// ReadRecoveryJSON reads in the recoveries data JSON file
func (d *Director) ReadRecoveryJSON() error {
	logger.TaskV("Reading Recovery JSON")
	errPath := "director.ReadRecoveryJSON()"
	jsonBytes, err := ioutil.ReadFile(config.Data.RecoveriesJSON)
	if err != nil {
		return errors.New(errPath, err)
	}
	if errb := json.Unmarshal(jsonBytes, &d.Recoveries); errb != nil {
		return errors.New(errPath, err)
	}
	return nil
}