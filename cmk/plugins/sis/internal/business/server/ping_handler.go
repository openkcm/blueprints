package server

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/openkcm/common-sdk/pkg/commoncfg"
	"github.com/openkcm/common-sdk/pkg/otlp"
	"github.com/openkcm/plugin-sdk/pkg/catalog"
	systeminformationv1 "github.com/openkcm/plugin-sdk/proto/plugin/systeminformation/v1"
	slogctx "github.com/veqryn/slog-context"
	"github.tools.sap/kms/sis-plugin/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func pingHandlerFunc(cfg *config.Config, plugins *catalog.Catalog) func(http.ResponseWriter, *http.Request) {
	traceAttrs := otlp.CreateAttributesFrom(cfg.Application,
		attribute.String(commoncfg.AttrOperation, "ping"),
	)

	tracer := otel.Tracer("PingHandler", trace.WithInstrumentationAttributes(traceAttrs...))

	sisPlugin := plugins.LookupByTypeAndName("SystemInformationService", "sis")
	sisClient := systeminformationv1.NewSystemInformationServiceClient(sisPlugin.ClientConnection())

	return func(w http.ResponseWriter, req *http.Request) {
		// Request Id will be propagated through all method calls propagated of this HTTP handler
		ctx := slogctx.With(req.Context(),
			commoncfg.AttrRequestID, uuid.New().String(),
			commoncfg.AttrOperation, "ping",
		)

		// Manual OTEL Tracing
		parentCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))

		ctx, span := tracer.Start(
			parentCtx,
			"ping-span",
			trace.WithAttributes(traceAttrs...),
		)
		defer span.End()

		// Metrics
		requestStartTime := time.Now()
		defer func() {
			elapsedTime := float64(time.Since(requestStartTime)) / float64(time.Millisecond)

			// Metrics logic
			attrs := metric.WithAttributes(
				otlp.CreateAttributesFrom(cfg.Application,
					attribute.String("userAgent", req.UserAgent()),
					attribute.String(commoncfg.AttrOperation, "ping"),
				)...,
			)

			counter.Add(ctx, 1, attrs)
			hist.Record(ctx, elapsedTime, attrs)
		}()

		// Business Logic
		slogctx.Info(ctx, "Starting ping request")
		{
			w.Header().Set("Content-Type", "application/json")

			_, err := sisClient.Get(ctx, &systeminformationv1.GetRequest{
				Id:   uuid.New().String(),
				Type: systeminformationv1.RequestType_REQUEST_TYPE_SYSTEM,
			})
			if err != nil {
				_, err := w.Write([]byte("{ \"error\": \"" + err.Error() + "\" }"))
				if err != nil {
					return
				}
			}

			_, err = w.Write([]byte("{ \"result\": \"OK\" }"))
			if err != nil {
				return
			}
		}
		DoSomething(ctx)
		slogctx.Info(ctx, "Finished ping request")
		// End Business Logic
	}
}

func DoSomething(ctx context.Context) {
	slogctx.Info(ctx, "Method DoSomething has been called")
}
