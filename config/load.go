package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"

	"github.com/spf13/pflag"
)

var debug, verbose bool

// Load loads the config for the server execution. By default config.json
func (c *Config) Load() error {
	op := "config.Load()"
	var confName string

	pflag.StringVarP(&confName, "config", "c", "config.json", "Sets the configuration filename. [Default: config.json] ")
	pflag.BoolVarP(&debug, "debug", "d", false, "Enables debug mode")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "Enables verbose mode")
	pflag.Parse()

	log.TaskV("Loading Config file %s", confName)
	jsonBytes, err := ioutil.ReadFile(confName)
	if err != nil {
		return errors.Extend(op, err)
	}
	if err := json.Unmarshal(jsonBytes, &Data); err != nil {
		return errors.Extend(op, err)
	}
	return nil
}

func SetLogger() {
	op := "config.SetLogger()"
	// if debug {
	// 	log.SetMode("debug")
	// } else if verbose {
	// 	log.SetMode("verbose")
	// }

	srvLogPath := path.Join(Data.RootLogDir, "logs", "server")
	Data.SrvLogDir = srvLogPath
	if err := os.MkdirAll(srvLogPath, 0700); err != nil {
		log.Error(op, err)
		os.Exit(1)
	}

	rcvrLogPath := path.Join(Data.RootLogDir, "logs", "recoveries")
	Data.RcvrLogDir = rcvrLogPath
	if err := os.MkdirAll(rcvrLogPath, 0700); err != nil {
		log.Error(op, err)
		os.Exit(1)
	}

	// now := time.Now().Format("2006-01-02T15h04m")
	// log.SetScope(true, true, false)
	// log.Info("Setting logfile to %s", path.Join(srvLogPath, "recoveryServer-"+now+".log"))
	// log.OutputFile(path.Join(srvLogPath, "recoveryServer-"+now+".log"))
	// log.StartWriter()
	// log.ToggleDualMode()
}

func CreatePDFDir() {
	if err := os.MkdirAll(Data.DeliveryDir, 0700); err != nil {
		log.Error("config.CreatePDFDir()", err)
		os.Exit(1)
	}
}
