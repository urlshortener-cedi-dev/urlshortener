package controller

import (
	shortlinkClient "github.com/cedi/urlshortener/pkg/client"
	"github.com/go-logr/logr"

	"go.opentelemetry.io/otel/trace"
)

// ShortlinkController is an object who handles the requests made towards our shortlink-application
type ShortlinkController struct {
	client              *shortlinkClient.ShortlinkClient
	authenticatedClient *shortlinkClient.ShortlinkClientAuth
	log                 *logr.Logger
	tracer              trace.Tracer
}

// NewShortlinkController creates a new ShortlinkController
func NewShortlinkController(log *logr.Logger, tracer trace.Tracer, client *shortlinkClient.ShortlinkClient) *ShortlinkController {
	controller := &ShortlinkController{
		log:                 log,
		tracer:              tracer,
		client:              client,
		authenticatedClient: shortlinkClient.NewAuthenticatedShortlinkClient(log, tracer, client),
	}

	return controller
}
