package controller

import (
	shortlinkClient "github.com/cedi/urlshortener/pkg/client"

	"go.opentelemetry.io/otel/trace"
)

// ShortlinkController is an object who handles the requests made towards our shortlink-application
type ShortlinkController struct {
	client              *shortlinkClient.ShortlinkClient
	authenticatedClient *shortlinkClient.ShortlinkClientAuth
	tracer              trace.Tracer
}

// NewShortlinkController creates a new ShortlinkController
func NewShortlinkController(tracer trace.Tracer, client *shortlinkClient.ShortlinkClient) *ShortlinkController {
	controller := &ShortlinkController{
		tracer:              tracer,
		client:              client,
		authenticatedClient: shortlinkClient.NewAuthenticatedShortlinkClient(tracer, client),
	}

	return controller
}
