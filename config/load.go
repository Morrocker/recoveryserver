package config

import (
	"os"
	"path"
	"strings"
	"time"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Load loads the config for the server execution. By default config.json
func (c *Config) Load() error {
	errPath := "config.Load()"
	name := viper.GetString("config")
	log.TaskV("Loading Config file %s", name)
	name, cType := parseConfigName(name)
	if name == "" || cType == "" {
		return errors.New(errPath, "config fileType not supported. Exiting")
	}

	viper.SetConfigName(name)
	viper.SetConfigType(cType)
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return errors.New(errPath, err)
	}
	viper.Unmarshal(&c)
	return nil
}

// SetFlags sets the flags for the application
func SetFlags() error {
	errPath := "config.SetFlags()"
	pflag.StringP("config", "c", "config.json", "Sets the configuration filename. [Default: config.json] ")
	pflag.BoolP("debug", "d", false, "Enables debug mode")
	pflag.BoolP("benchmark", "b", false, "Enables benchmark mode")
	pflag.BoolP("verbose", "v", false, "Enables verbose mode")

	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return errors.New(errPath, err)
	}
	return nil
}

func parseConfigName(filename string) (name, extension string) {
	fields := strings.Split(filename, ".")
	if fields[len(fields)-1] == "json" {
		name = fields[0]
		extension = "json"
		for x := 1; x < len(fields)-1; x++ {
			name = name + "." + fields[x]
		}
		return
	}
	return
}

func SetLogger() {
	errPath := "config.SetLogger()"
	if viper.GetBool("debug") {
		log.SetMode("debug")
	} else if viper.GetBool("verbose") {
		log.SetMode("verbose")
	}
	srvLogPath := path.Join(Data.RootLogDir, "logs", "server")
	Data.SrvLogDir = srvLogPath
	if err := os.MkdirAll(srvLogPath, 0700); err != nil {
		log.Error(errPath, err)
		os.Exit(1)
	}
	rcvrLogPath := path.Join(Data.RootLogDir, "logs", "recoveries")
	Data.RcvrLogDir = rcvrLogPath
	if err := os.MkdirAll(rcvrLogPath, 0700); err != nil {
		log.Error(errPath, err)
		os.Exit(1)
	}
	now := time.Now().Format("2006-01-02T15h04m")
	// log.ToggleTimestamp()
	log.SetScope(true, true, false)
	log.Info("Setting logfile to %s", path.Join(srvLogPath, "recoveryServer-"+now+".log"))
	log.OutputFile(path.Join(srvLogPath, "recoveryServer-"+now+".log"))
	log.StartWriter()
	log.ToggleDualMode()
}
