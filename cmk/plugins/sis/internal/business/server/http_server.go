package server

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/openkcm/plugin-sdk/pkg/catalog"
	"github.com/samber/oops"
	slogctx "github.com/veqryn/slog-context"
	"github.tools.sap/kms/sis-plugin/internal/config"
)

// registerHandlers registers the default http handlers for the status server
func registerHandlers(mux *http.ServeMux, cfg *config.Config, plugins *catalog.Catalog) {
	mux.HandleFunc("/ping", pingHandlerFunc(cfg, plugins))
}

// createStatusServer creates a status http server using the given config
func createHTTPServer(ctx context.Context, cfg *config.Config, plugins *catalog.Catalog) *http.Server {
	mux := http.NewServeMux()
	registerHandlers(mux, cfg, plugins)

	slogctx.Info(ctx, "Creating HTTP server", "address", cfg.HTTP.Address)

	return &http.Server{
		Addr:    cfg.HTTP.Address,
		Handler: mux,
	}
}

// StartHTTPServer starts the gRPC server using the given config.
func StartHTTPServer(ctx context.Context, cfg *config.Config, plugins *catalog.Catalog) error {
	if err := initMeters(ctx, cfg); err != nil {
		return err
	}

	server := createHTTPServer(ctx, cfg, plugins)

	slogctx.Info(ctx, "Starting HTTP listener", "address", server.Addr)

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return oops.In("HTTP Server").
			WithContext(ctx).
			Wrapf(err, "Failed creating HTTP listener")
	}

	go func() {
		slogctx.Info(ctx, "Starting HTTP server", "address", server.Addr)

		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slogctx.Error(ctx, "ErrorField serving HTTP endpoint", "error", err)
		}

		slogctx.Info(ctx, "Stopped HTTP server")
	}()

	<-ctx.Done()

	shutdownCtx, shutdownRelease := context.WithTimeout(ctx, cfg.HTTP.ShutdownTimeout)
	defer shutdownRelease()

	listErrors := make([]error, 0)
	err = plugins.Close()
	if err != nil {
		listErrors = append(listErrors, oops.In("HTTP Server").
			WithContext(ctx).
			Wrapf(err, "Failed to close plugins"))
	}

	if err = server.Shutdown(shutdownCtx); err != nil {
		listErrors = append(listErrors, oops.In("HTTP Server").
			WithContext(ctx).
			Wrapf(err, "Failed shutting down HTTP server"))
	}

	if len(listErrors) > 0 {
		return errors.Join(listErrors...)
	}

	slogctx.Info(ctx, "Completed graceful shutdown of HTTP server")

	return nil
}
