package config

import (
	"strings"

	"github.com/morrocker/errors"
	"github.com/morrocker/logger"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var e = errors.Error{Path: "config"}

// Load loads the config for the server execution. By default config.json
func (c *Config) Load() error {
	e.SetFunc("LoadConfig()")
	name := viper.GetString("config")
	logger.TaskV("Loading Config file %s", name)
	name, cType := parseConfigName(name)
	if name == "" || cType == "" {
		e.New("config fileType not supported. Exiting")
		return e
	}

	viper.SetConfigName(name)
	viper.SetConfigType(cType)
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		e.New(err)
		return e
	}
	viper.Unmarshal(&c)
	return nil
}

// SetFlags sets the flags for the application
func SetFlags() error {
	e.SetFunc("SetFlags()")
	pflag.StringP("config", "c", "config.json", "Sets the configuration filename. [Default: config.json] ")
	pflag.BoolP("debug", "d", false, "Enables debug mode")
	pflag.BoolP("verbose", "v", false, "Enables verbose mode")

	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		e.New(err)
		return e
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
