package config

import (
	"strings"

	"github.com/morrocker/errors"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Conf stores the configuration info imported from the configuration file
var err = errors.Error{Path: "config"}

// LoadConfig loads the config for the server execution. By default config.json
func LoadConfig() error {
	name := viper.GetString("config")
	err.SetFunc("LoadConfig()")
	name, cType := parseConfigName(name)
	if name == "" || cType == "" {
		err.New("config fileType not supported. Exiting")
		return err
	}

	viper.SetConfigName(name)
	viper.SetConfigType(cType)
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	viper.Unmarshal(&Data)
	return nil
}

// SetFlags sets the flags for the application
func SetFlags() {
	pflag.StringP("config", "c", "config.json", "Sets the configuration filename. [Default: config.json] ")
	pflag.BoolP("debug", "d", false, "Enables debug mode")
	pflag.BoolP("verbose", "v", false, "Enables verbose mode")

	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
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
