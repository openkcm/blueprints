package business

import (
	"context"
	"log/slog"

	"github.com/openkcm/common-sdk/pkg/commoncfg"
	"github.com/openkcm/plugin-sdk/pkg/catalog"
	"github.com/openkcm/sis-builtin-plugin/internal/builtin"
	"github.com/openkcm/sis-builtin-plugin/internal/business/server"
	"github.com/openkcm/sis-builtin-plugin/internal/config"
	servicewrapper "github.com/openkcm/sis-builtin-plugin/internal/service/wrapper"
	slogctx "github.com/veqryn/slog-context"
)

func Main(ctx context.Context, cfg *config.Config) error {
	buildInPlugins := catalog.CreateBuiltInPluginRegistry()
	builtin.RegisterAllBuiltInPlugins(buildInPlugins)

	registry, err := servicewrapper.CreateServiceRepository(ctx, catalog.Config{
		Logger:        slog.Default(),
		PluginConfigs: cfg.Plugins,
	}, buildInPlugins.Retrieve()...)
	if err != nil {
		return err
	}

	pluginBuildInfos := make([]string, 0)
	for _, pluginInfo := range registry.RawCatalog.ListPluginInfo() {
		pluginBuildInfos = append(pluginBuildInfos, pluginInfo.Build())
	}

	err = commoncfg.UpdateComponentsOfBuildInfo(&cfg.BaseConfig, pluginBuildInfos...)
	if err != nil {
		slogctx.Error(ctx, "Failed to update components of build info")
	}

	return server.StartHTTPServer(ctx, cfg, registry)
}
