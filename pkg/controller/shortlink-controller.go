package controller

import (
	shortlinkClient "github.com/cedi/urlshortener/pkg/client"
	"go.uber.org/zap"

	"go.opentelemetry.io/otel/trace"
)

// ShortlinkController is an object who handles the requests made towards our shortlink-application
type ShortlinkController struct {
	client              *shortlinkClient.ShortlinkClient
	authenticatedClient *shortlinkClient.ShortlinkClientAuth
	zapLog              *zap.Logger
	tracer              trace.Tracer
}

// NewShortlinkController creates a new ShortlinkController
func NewShortlinkController(zapLog *zap.Logger, tracer trace.Tracer, client *shortlinkClient.ShortlinkClient) *ShortlinkController {
	controller := &ShortlinkController{
		zapLog:              zapLog,
		tracer:              tracer,
		client:              client,
		authenticatedClient: shortlinkClient.NewAuthenticatedShortlinkClient(zapLog, tracer, client),
	}

	return controller
}
