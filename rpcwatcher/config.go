package rpcwatcher

import (
	"rpc_watcher/rpcwatcher/configuration"

	"github.com/emerishq/demeris-backend-models/validation"
	"github.com/go-playground/validator/v10"
)

const (
	defaultPulsarURL          = "pulsar://localhost:6650"
	defaultRpcURL             = "http://127.0.0.1:26657"
	defaultProfilingServerURL = "localhost:6060"
)

type Config struct {
	PulsarURL          string `validate:"required"`
	RpcURL             string `validate:"required"`
	ProfilingServerURL string `validate:"hostname_port"`
	ApiURL             string `validate:"required"`
	Debug              bool
	JSONLogs           bool
}

func (c *Config) Validate() error {
	err := validator.New().Struct(c)
	if err == nil {
		return nil
	}

	return validation.MissingFieldsErr(err, false)
}

func ReadConfig() (*Config, error) {
	var c Config
	return &c, configuration.ReadConfig(&c, "rpcwatcher", map[string]string{
		"PulsarURL":          defaultPulsarURL,
		"RpcURL":             defaultRpcURL,
		"ProfilingServerURL": defaultProfilingServerURL,
	})
}
