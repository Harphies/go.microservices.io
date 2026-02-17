package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.uber.org/zap"
)

// DistributedTracing configures and initialises an OpenTelemetry TracerProvider
// that exports spans via OTLP gRPC to an OpenTelemetry Collector.
type DistributedTracing struct {
	logger      *zap.Logger
	environment string
	serviceName string
	version     string
}

// Option applies optional configuration to DistributedTracing.
type Option func(*DistributedTracing)

// WithServiceVersion sets the service.version resource attribute.
func WithServiceVersion(version string) Option {
	return func(dt *DistributedTracing) {
		dt.version = version
	}
}

// NewDistributedTracingWithOpenTelemetry creates a new DistributedTracing instance.
// The opts parameter is variadic so existing callers without options continue to work.
func NewDistributedTracingWithOpenTelemetry(logger *zap.Logger, environment, serviceName string, opts ...Option) *DistributedTracing {
	dt := &DistributedTracing{
		logger:      logger,
		environment: environment,
		serviceName: serviceName,
	}
	for _, opt := range opts {
		opt(dt)
	}
	return dt
}

// InitProviderWithOpenTelemetryCollectorGrpcEndpoint creates a TracerProvider that
// exports spans to an OpenTelemetry Collector over gRPC. The collector endpoint is
// read from the standard OTEL_EXPORTER_OTLP_ENDPOINT environment variable.
//
// Returns a non-nil error if the exporter cannot be created. Resource creation
// warnings are logged but do not prevent initialisation.
func (t *DistributedTracing) InitProviderWithOpenTelemetryCollectorGrpcEndpoint() (*trace.TracerProvider, error) {
	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create OTLP gRPC exporter: %w", err)
	}

	res, err := t.newResource(ctx)
	if err != nil {
		t.logger.Warn("resource creation had partial errors, continuing with best-effort resource", zap.Error(err))
	}

	bsp := trace.NewBatchSpanProcessor(exporter,
		trace.WithBatchTimeout(5*time.Second),
		trace.WithMaxExportBatchSize(512),
	)

	tp := trace.NewTracerProvider(
		trace.WithSampler(t.getSampler()),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t.logger.Info("OpenTelemetry tracer provider initialised",
		zap.String("service", t.serviceName),
		zap.String("environment", t.environment),
	)

	return tp, nil
}

// newResource builds an OpenTelemetry resource with standard semantic convention
// attributes. Each call creates a fresh resource (no global state).
func (t *DistributedTracing) newResource(ctx context.Context) (*resource.Resource, error) {
	attrs := []resource.Option{
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceName(t.serviceName),
			semconv.DeploymentEnvironment(t.environment),
		),
	}

	if t.version != "" {
		attrs = append(attrs, resource.WithAttributes(semconv.ServiceVersion(t.version)))
	}

	extraResources, err := resource.New(ctx, attrs...)
	if err != nil {
		return resource.Default(), fmt.Errorf("create extra resources: %w", err)
	}

	merged, mergeErr := resource.Merge(resource.Default(), extraResources)
	if mergeErr != nil {
		return resource.Default(), fmt.Errorf("merge resources: %w", mergeErr)
	}

	return merged, err
}

// getSampler returns a trace.Sampler appropriate for the target environment:
//
//	dev, development → AlwaysSample (100 %)
//	staging          → ParentBased(TraceIDRatioBased(0.5)) (50 %)
//	prod, production → ParentBased(TraceIDRatioBased(0.1)) (10 %)
//	default          → AlwaysSample
func (t *DistributedTracing) getSampler() trace.Sampler {
	switch t.environment {
	case "dev", "development":
		return trace.AlwaysSample()
	case "staging":
		return trace.ParentBased(trace.TraceIDRatioBased(0.5))
	case "prod", "production":
		return trace.ParentBased(trace.TraceIDRatioBased(0.1))
	default:
		return trace.AlwaysSample()
	}
}
