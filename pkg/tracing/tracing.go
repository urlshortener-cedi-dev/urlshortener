package tracing

import (
	"context"
	"fmt"
	"os"

	"github.com/MrAlias/flow"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

type ShortlinkObservability struct {
	Trace         trace.Tracer
	TraceProvider *sdktrace.TracerProvider
	Log           logr.Logger
}

func NewShortlinkObservability(serviceName, serviceVersion string, log logr.Logger) (*ShortlinkObservability, error) {
	o11y := &ShortlinkObservability{
		Log: log,
	}

	err := o11y.initTracer(serviceName, serviceVersion)

	return o11y, err
}

func (s *ShortlinkObservability) initTracer(serviceName, serviceVersion string) error {
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint("localhost:4318"),
		otlptracehttp.WithInsecure(),
	)

	otlptracehttpExporter, err := otlptrace.New(context.TODO(), client)
	if err != nil {
		return errors.Wrap(err, "failed creating OTLP trace exporter")
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(serviceVersion),
		semconv.ServiceInstanceIDKey.String(hostname),
	)

	s.TraceProvider = sdktrace.NewTracerProvider(
		flow.WithBatcher(otlptracehttpExporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resources),
	)

	s.Trace = s.TraceProvider.Tracer(
		serviceName,
		trace.WithInstrumentationVersion(serviceVersion),
		trace.WithSchemaURL(semconv.SchemaURL),
	)

	otel.SetTracerProvider(s.TraceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return nil
}

func (s *ShortlinkObservability) ShutdownTraceProvider(ctx context.Context) error {
	return s.TraceProvider.Shutdown(ctx)
}

func RecordError(span trace.Span, log *logr.Logger, err error, msg string, args ...any) {
	log.Error(err, fmt.Sprintf(msg, args...))
	span.AddEvent(fmt.Sprintf(msg, args...))
	span.RecordError(err)
}

func RecordInfo(span trace.Span, log *logr.Logger, msg string, args ...any) {
	log.Info(fmt.Sprintf(msg, args...))
	span.AddEvent(fmt.Sprintf(msg, args...))
}
