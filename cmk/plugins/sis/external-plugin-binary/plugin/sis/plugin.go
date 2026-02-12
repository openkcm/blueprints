package sis

import (
	"context"
	"log/slog"

	"buf.build/go/protovalidate"
	"github.com/hashicorp/go-hclog"
	"github.com/openkcm/plugin-sdk/pkg/catalog"
	systeminformationv1 "github.com/openkcm/plugin-sdk/proto/plugin/systeminformation/v1"
	configv1 "github.com/openkcm/plugin-sdk/proto/service/common/config/v1"
	slogctx "github.com/veqryn/slog-context"
	"gopkg.in/yaml.v3"

	"github.com/openkcm/plugin-sdk/pkg/hclog2slog"
)

type Plugin struct {
	configv1.UnsafeConfigServer
	systeminformationv1.UnimplementedSystemInformationServiceServer

	buildInfo string
}

var (
	_ systeminformationv1.SystemInformationServiceServer = (*Plugin)(nil)
	_ configv1.ConfigServer                              = (*Plugin)(nil)
)

func BuiltIn() catalog.BuiltInPlugin {
	return builtin(NewPlugin("{}"))
}

func builtin(p *Plugin) catalog.BuiltInPlugin {
	return catalog.AsBuiltIn("sis",
		systeminformationv1.SystemInformationServicePluginServer(p),
		configv1.ConfigServiceServer(p))
}

func NewPlugin(buildInfo string) *Plugin {
	return &Plugin{
		buildInfo: buildInfo,
	}
}

// SetLogger method is called whenever the plugin start and giving the logger of host application
func (p *Plugin) SetLogger(logger hclog.Logger) {
	slog.SetDefault(hclog2slog.New(logger.Named("plugin.sis")))
}

// Configure configures the plugin with the given configuration
func (p *Plugin) Configure(ctx context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error) {
	slogctx.Info(ctx, "Configuring plugin")

	cfg := &Config{}
	err := yaml.Unmarshal([]byte(req.GetYamlConfiguration()), cfg)
	if err != nil {
		return nil, err
	}

	//TODO: Additional business logic to be added here using the plugin configuration
	// Use additional cfg.CustomX plugin configuration

	return &configv1.ConfigureResponse{
		BuildInfo: &p.buildInfo,
	}, nil
}

// Get Plugin method/operation
func (p *Plugin) Get(ctx context.Context, req *systeminformationv1.GetRequest) (*systeminformationv1.GetResponse, error) {
	slogctx.Debug(ctx, "SIS Get called", "req", req)
	if err := protovalidate.Validate(req); err != nil {
		return nil, err
	}

	//TODO: Business logic here

	return &systeminformationv1.GetResponse{}, nil
}
