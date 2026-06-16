package fipcontroller

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// tracerName is the instrumentation scope name used for all spans.
const tracerName = "github.com/cbeneke/hcloud-fip-controller"

// tracer returns the globally configured tracer. When tracing has not been
// initialised this is a no-op tracer, so spans can be created unconditionally.
func tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// InitTracing configures the global OpenTelemetry tracer provider with an OTLP
// gRPC exporter pointing at the given endpoint. When the endpoint is empty no
// provider is configured and a no-op shutdown function is returned, so traces
// are not emitted unless a target is configured.
func InitTracing(ctx context.Context, endpoint, serviceName, serviceVersion string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	if endpoint == "" {
		return noop, nil
	}

	// otlptracegrpc.WithEndpointURL understands the scheme (http -> insecure,
	// https -> TLS). Allow a bare "host:port" by defaulting to http.
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(endpoint))
	if err != nil {
		return noop, fmt.Errorf("could not create OTLP trace exporter: %v", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("service.version", serviceVersion),
	))
	if err != nil {
		return noop, fmt.Errorf("could not create trace resource: %v", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return provider.Shutdown, nil
}
