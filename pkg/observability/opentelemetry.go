package observability

import (
	"context"
	"os"
	"strings"

	"github.com/MrAlias/flow"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

func InitTracer(serviceName, serviceVersion string) (*sdkTrace.TracerProvider, trace.Tracer, error) {

	otlpEndpoint, ok := os.LookupEnv("OTLP_ENDPOINT")
	otlpInsecure := os.Getenv("OTLP_INSECURE")

	otlpOptions := make([]otlptracehttp.Option, 0)

	if ok {
		otlpOptions = append(otlpOptions, otlptracehttp.WithEndpoint(otlpEndpoint))

		if strings.ToLower(otlpInsecure) == "true" {
			otlpOptions = append(otlpOptions, otlptracehttp.WithInsecure())
		}
	} else {
		otlpOptions = append(otlpOptions, otlptracehttp.WithEndpoint("localhost:4318"))
		otlpOptions = append(otlpOptions, otlptracehttp.WithInsecure())
	}

	client := otlptracehttp.NewClient(otlpOptions...)

	otlptracehttpExporter, err := otlptrace.New(context.TODO(), client)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed creating OTLP trace exporter")
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, err
	}

	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(serviceVersion),
		semconv.ServiceInstanceIDKey.String(hostname),
	)

	traceProvider := sdkTrace.NewTracerProvider(
		flow.WithBatcher(otlptracehttpExporter),
		sdkTrace.WithSampler(sdkTrace.AlwaysSample()),
		sdkTrace.WithResource(resources),
	)

	trace := traceProvider.Tracer(
		serviceName,
		trace.WithInstrumentationVersion(serviceVersion),
		trace.WithSchemaURL(semconv.SchemaURL),
	)

	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return traceProvider, trace, nil
}
