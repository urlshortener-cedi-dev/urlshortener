package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const ContentTypeApplicationJSON = "application/json"

type ShortlinkController struct {
	log    *logr.Logger
	tracer trace.Tracer
}

func NewShortlinkController(log *logr.Logger, tracer trace.Tracer) *ShortlinkController {
	return &ShortlinkController{
		log:    log,
		tracer: tracer,
	}

}

func (s *ShortlinkController) HandleShortLink(c *gin.Context) {
	// Call the HTML method of the Context to render a template
	_, span := s.tracer.Start(c.Request.Context(), "/:shortlink", trace.WithAttributes(attribute.String("shortlink", c.Request.URL.Path)))
	defer span.End()

	c.HTML(
		// Set the HTTP status to 200 (OK)
		http.StatusOK,

		// Use the index.html template
		"redirect.html",

		// Pass the data that the page uses (in this case, 'title')
		gin.H{
			"redirectFrom": c.Request.URL.Path,
		},
	)
}
