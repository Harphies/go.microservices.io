package tracing

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"sync"
	"time"
)

var initResourcesOnce sync.Once
var res *resource.Resource

type DistributedTracing struct {
	logger      *zap.Logger
	environment string
	serviceName string
}

// NewDistributedTracingWithOpenTelemetry ...
func NewDistributedTracingWithOpenTelemetry(logger *zap.Logger, environment, serviceName string) *DistributedTracing {
	return &DistributedTracing{
		logger:      logger,
		environment: environment,
		serviceName: serviceName,
	}
}

// InitProviderWithOpenTelemetryCollectorGrpcEndpoint - This enables the service to establish a GRPC connection with
// OpenTelemetry Collector using the standard environment variable `OTEL_EXPORTER_OTLP_ENDPOINT` on the process to look up the OpenTelemetry Collector GRPC endpoint.
// Then OpenTelemetry Collector Agent collect traces from this service
func (t *DistributedTracing) InitProviderWithOpenTelemetryCollectorGrpcEndpoint() (*trace.TracerProvider, error) {
	ctx := context.Background()

	traceExporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		t.logger.Error(fmt.Sprintf("failed to create trace exporter: %s", err.Error()))
	}

	bsp := trace.NewBatchSpanProcessor(traceExporter)
	tp := trace.NewTracerProvider(
		trace.WithSampler(t.getSampler()),
		trace.WithResource(t.newResource(ctx)),
		trace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

func (t *DistributedTracing) InitProviderWithOpenTelemetryCollectorHTTPEndpoint() (func(context.Context) error, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	address := os.Getenv("OPEN_TELEMETRY_COLLECTOR_HTTP_ENDPOINT")
	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.logger.Error(fmt.Sprintf("failed to create grpc connection to collector: %s", err.Error()))
	}
	// defer conn.Close()
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		t.logger.Error(fmt.Sprintf("failed to create trace exporter: %s", err.Error()))
	}

	bsp := trace.NewBatchSpanProcessor(traceExporter)
	tp := trace.NewTracerProvider(
		trace.WithSampler(t.getSampler()),
		trace.WithResource(t.newResource(ctx)),
		trace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp.Shutdown, nil
}

// Returns a new OpenTelemetry resource describing this application.
func (t *DistributedTracing) newResource(ctx context.Context) *resource.Resource {
	initResourcesOnce.Do(func() {
		extraResources, err := resource.New(ctx,
			resource.WithFromEnv(),
			resource.WithProcess(),
			resource.WithTelemetrySDK(),
			resource.WithHost(),
			resource.WithAttributes(semconv.ServiceNameKey.String(t.serviceName),
				attribute.String("environment", t.environment),
			),
		)
		if err != nil {
			t.logger.Error(fmt.Sprintf("%s: %v", "Failed to create resource", err))
		}
		res, _ = resource.Merge(
			resource.Default(),
			extraResources,
		)
	})
	return res
}

func (t *DistributedTracing) getSampler() trace.Sampler {
	switch t.environment {
	case "dev":
		return trace.AlwaysSample()
	case "prod":
		return trace.ParentBased(trace.TraceIDRatioBased(0.5))
	default:
		return trace.AlwaysSample()
	}
}
