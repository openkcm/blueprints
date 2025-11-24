package business

import (
	"context"
	"log/slog"

	"github.com/openkcm/common-sdk/pkg/commoncfg"
	"github.com/openkcm/plugin-sdk/pkg/catalog"
	slogctx "github.com/veqryn/slog-context"
	"github.tools.sap/kms/sis-plugin/internal/business/server"
	"github.tools.sap/kms/sis-plugin/internal/config"
)

func Main(ctx context.Context, cfg *config.Config) error {

	// Loading all plugins given through config.yaml file as configuration
	plugins, err := catalog.Load(ctx, catalog.Config{
		Logger:        slog.Default(),
		PluginConfigs: cfg.Plugins,
	})
	if err != nil {
		return err
	}

	pluginBuildInfos := make([]string, 0)
	for _, pluginInfo := range plugins.ListPluginInfo() {
		pluginBuildInfos = append(pluginBuildInfos, pluginInfo.Build())
	}

	err = commoncfg.UpdateComponentsOfBuildInfo(&cfg.BaseConfig, pluginBuildInfos...)
	if err != nil {
		slogctx.Error(ctx, "Failed to update components of build info")
	}

	return server.StartHTTPServer(ctx, cfg, plugins)
}
