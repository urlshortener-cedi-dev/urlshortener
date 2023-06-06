package observability

import (
	"fmt"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func RecordError(span trace.Span, zapLog *zap.SugaredLogger, err error, msg string, args ...any) error {
	message := fmt.Sprintf(msg, args...)
	span.AddEvent(message)

	zapLog.Errorw(msg, zap.Error(err))

	err = errors.Wrap(err, message)
	span.RecordError(err)
	return err
}

func RecordInfo(span trace.Span, zapLog *zap.SugaredLogger, msg string, args ...any) {
	zapLog.Infof(msg, args...)
	span.AddEvent(fmt.Sprintf(msg, args...))
}
