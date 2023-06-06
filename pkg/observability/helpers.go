package observability

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

func RecordError(span trace.Span, log *logr.Logger, err error, msg string, args ...any) error {
	message := fmt.Sprintf(msg, args...)
	span.AddEvent(message)

	log.WithValues("traceID", span.SpanContext().TraceID()).Error(err, message)

	err = errors.Wrap(err, message)
	span.RecordError(err)
	return err
}

func RecordInfo(span trace.Span, log *logr.Logger, msg string, args ...any) {
	log.WithValues("traceID", span.SpanContext().TraceID()).Info(fmt.Sprintf(msg, args...))
	span.AddEvent(fmt.Sprintf(msg, args...))
}
