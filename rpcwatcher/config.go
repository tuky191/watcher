package rpcwatcher

import (
	"rpc_watcher/rpcwatcher/configuration"

	"github.com/emerishq/demeris-backend-models/validation"
	"github.com/go-playground/validator/v10"
)

const (
	defaultRedisURL           = "redis-master:6379"
	defaultApiURL             = "http://api-server:8000"
	defaultProfilingServerURL = "localhost:6060"
)

type Config struct {
	DatabaseConnectionURL string `validate:"required"`
	RedisURL              string `validate:"required"`
	ApiURL                string `validate:"required,url"`
	ProfilingServerURL    string `validate:"hostname_port"`
	Debug                 bool
	JSONLogs              bool
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
		"RedisURL":              defaultRedisURL,
		"ApiURL":                defaultApiURL,
		"ProfilingServerURL":    defaultProfilingServerURL,
		"DatabaseConnectionURL": defaultProfilingServerURL,
	})
}
