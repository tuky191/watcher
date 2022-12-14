package configuration

import (
	"errors"
	"fmt"
	"strings"

	"github.com/iamolegga/enviper"

	"github.com/spf13/viper"
)

// Validator is an object that implements a validation method, which accepts no argument and returns an error.
type Validator interface {
	Validate() error
}

// ReadConfig reads the TOML configuration file in predefined standard paths into v, returns an error if v.Validate()
// returns error, or some configuration file reading error happens.
// v is the destination struct, configName is the name used for the configuration file.
// ReadConfig will not return an error for missing configuration file, since the fields contained in v can be also
// read from environment variables.
func ReadConfig(v Validator, configName string, defaultValues map[string]string) error {
	vip := enviper.New(viper.New())

	for k, v := range defaultValues {
		vip.SetDefault(k, v)
	}

	vip.SetConfigName(configName)
	vip.SetConfigType("yaml")
	vip.AddConfigPath(fmt.Sprintf("/etc/%s", configName))
	vip.AddConfigPath(fmt.Sprintf("$HOME/%s", configName))
	vip.AddConfigPath("./")
	vip.SetEnvPrefix(strings.ToLower(configName))
	vip.AutomaticEnv()
	fmt.Println(vip.ConfigFileUsed())
	if err := vip.ReadInConfig(); err != nil {
		// We only return from here if
		// err is not a viper.ConfigFileNotFoundError type
		// Because if no config file found, we can still have
		// configurations from defaultValues or from env variables.
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return err
		}
	}

	if err := vip.Unmarshal(v); err != nil {
		return fmt.Errorf("config error: %s", err)
	}

	return v.Validate()
}
