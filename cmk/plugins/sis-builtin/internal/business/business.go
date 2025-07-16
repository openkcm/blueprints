package business

import (
	"context"
	"log/slog"

	"github.com/openkcm/plugin-sdk/pkg/catalog"
	"github.tools.sap/kms/sis-builtin-plugin/internal/builtin"
	"github.tools.sap/kms/sis-builtin-plugin/internal/business/server"
	"github.tools.sap/kms/sis-builtin-plugin/internal/config"
)

func Main(ctx context.Context, cfg *config.Config) error {
	plugins, err := catalog.Load(ctx, catalog.Config{
		Logger:        slog.Default(),
		PluginConfigs: cfg.Plugins,
	}, builtin.BuiltIns()...)
	if err != nil {
		return err
	}

	return server.StartHTTPServer(ctx, cfg, plugins)
}
