// Package config defines the necessary types to configure the application.
// An example config file config.yaml is provided in the repository.
package config

import (
	"time"

	"github.com/openkcm/common-sdk/pkg/commoncfg"
	"github.com/openkcm/plugin-sdk/pkg/catalog"
)

type Config struct {
	commoncfg.BaseConfig `mapstructure:",squash"`

	HTTP    HTTPServer
	Plugins []catalog.PluginConfig
}

type HTTPServer struct {
	// HTTP.Address is the address to listen on for HTTP requests
	Address         string        `yaml:"address" default:":8080"`
	ShutdownTimeout time.Duration `yaml:"shutdownTimeout" default:"5s"`
}
