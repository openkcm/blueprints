package business

import (
	"context"
	"log/slog"

	"github.com/openkcm/plugin-sdk/pkg/catalog"
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

	return server.StartHTTPServer(ctx, cfg, plugins)
}
