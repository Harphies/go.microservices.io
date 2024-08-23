package tracing

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/trace"
	"os"
)

func (t *DistributedTracing) InitProviderWithJaegerExporter(ctx context.Context) (func(context.Context) error, error) {
	exp, err := t.exporterToJaeger()
	if err != nil {
		t.logger.Error(fmt.Sprintf("error: %s", err.Error()))
	}
	tp := trace.NewTracerProvider(
		trace.WithSampler(t.getSampler()),
		trace.WithBatcher(exp),
		trace.WithResource(t.newResource(ctx)),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// Creates Jaeger exporter
func (t *DistributedTracing) exporterToJaeger() (*jaeger.Exporter, error) {
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(os.Getenv("OPEN_TELEMETRY_COLLECTOR_JAEGER_EXPORTER_ENDPOINT"))))
}
