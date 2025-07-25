package main

import (
	"context"

	"github.com/hashicorp/go-hclog"
	systeminformationv1 "github.com/openkcm/plugin-sdk/proto/plugin/systeminformation/v1"
	configv1 "github.com/openkcm/plugin-sdk/proto/service/common/config/v1"
	"gopkg.in/yaml.v3"
)

type Plugin struct {
	configv1.UnsafeConfigServer
	systeminformationv1.UnimplementedSystemInformationServiceServer

	logger hclog.Logger
}

var (
	_ systeminformationv1.SystemInformationServiceServer = (*Plugin)(nil)
	_ configv1.ConfigServer                              = (*Plugin)(nil)
)

func NewPlugin() *Plugin {
	return &Plugin{}
}

// SetLogger method is called whenever the plugin start and giving the logger of host application
func (p *Plugin) SetLogger(logger hclog.Logger) {
	p.logger = logger.Named("plugin.sis")
}

// Configure configures the plugin with the given configuration
func (p *Plugin) Configure(_ context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error) {
	p.logger.Info("Configuring plugin")

	cfg := &Config{}
	err := yaml.Unmarshal([]byte(req.GetYamlConfiguration()), cfg)
	if err != nil {
		return nil, err
	}

	//TODO: Additional business logic to be added here using the plugin configuration
	// Use additional cfg.CustomX plugin configuration

	return &configv1.ConfigureResponse{}, nil
}

// Get Plugin method/operation
func (p *Plugin) Get(_ context.Context, req *systeminformationv1.GetRequest) (*systeminformationv1.GetResponse, error) {

	p.logger.Debug("SIS Get called", "req", req.GetId())

	//TODO: Business logic here

	return &systeminformationv1.GetResponse{}, nil
}
