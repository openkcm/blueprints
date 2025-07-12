package sis

import (
	"context"
	"log/slog"

	"github.com/hashicorp/go-hclog"
	systeminformationv1 "github.com/openkcm/plugin-sdk/proto/plugin/systeminformation/v1"
	configv1 "github.com/openkcm/plugin-sdk/proto/service/common/config/v1"
	slogctx "github.com/veqryn/slog-context"
	"github.tools.sap/kms/cert-issuer-plugin/internal/config"
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
	p.logger = logger
}

// Configure configures the plugin with the given configuration
func (p *Plugin) Configure(_ context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error) {
	slog.Info("Configuring plugin")

	cfg := &config.Config{}
	err := yaml.Unmarshal([]byte(req.GetYamlConfiguration()), cfg)
	if err != nil {
		return nil, err
	}

	// Logger initialization here in case if you don't have the above defined SetLogger method, that will inject the logger seamlessly
	//err = logger.InitAsDefault(cfg.Logger, cfg.Application)
	//if err != nil {
	//	return nil, oops.In("main").
	//		Wrapf(err, "Failed to initialise the logger")
	//}

	//TODO: Additional business logic to be added here using the plugin configuration
	// Use additional cfg.CustomX plugin configuration

	return &configv1.ConfigureResponse{}, nil
}

// Get Plugin method/operation
func (p *Plugin) Get(ctx context.Context, _ *systeminformationv1.GetRequest) (*systeminformationv1.GetResponse, error) {

	slogctx.Debug(ctx, "Get called")

	//TODO: Business logic here

	return &systeminformationv1.GetResponse{}, nil
}
