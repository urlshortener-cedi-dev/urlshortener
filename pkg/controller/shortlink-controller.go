package api

import (
	"fmt"
	"net/http"

	shortlinkclient "github.com/av0de/urlshortener/pkg/client"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const ContentTypeApplicationJSON = "application/json"

type ShortlinkController struct {
	log    *logr.Logger
	tracer trace.Tracer
	client *shortlinkclient.ShortlinkClient
}

func NewShortlinkController(log *logr.Logger, tracer trace.Tracer, client *shortlinkclient.ShortlinkClient) *ShortlinkController {
	return &ShortlinkController{
		log:    log,
		tracer: tracer,
		client: client,
	}

}

func (s *ShortlinkController) HandleShortLink(c *gin.Context) {
	shortlink := c.Request.URL.Path[1:]

	// Call the HTML method of the Context to render a template
	ctx, span := s.tracer.Start(c.Request.Context(), "/:shortlink", trace.WithAttributes(attribute.String("shortlink", shortlink)))
	defer span.End()

	shortlinks, err := s.client.Query(ctx, shortlink)
	if err != nil || len(shortlinks.Items) > 1 {
		if err != nil {
			span.RecordError(err)
		} else {
			span.RecordError(fmt.Errorf("more than one shortlink definition found"))
		}

		c.HTML(
			// Set the HTTP status to 500 (Internal Server Error)
			http.StatusInternalServerError,

			// Use the index.html template
			"500.html",

			gin.H{},
		)

		return
	}

	if len(shortlinks.Items) == 0 {
		span.RecordError(fmt.Errorf("no shortlink definition '%s' found", shortlink))
		c.HTML(
			// Set the HTTP status to 404 (Not Found)
			http.StatusNotFound,

			// Use the index.html template
			"404.html",

			gin.H{},
		)

		return
	}

	if len(shortlinks.Items) == 1 {
		shortlinkObj := shortlinks.Items[0]

		shortlinkObj.Status.Count++

		c.HTML(
			// Set the HTTP status to 200 (OK)
			http.StatusOK,

			// Use the index.html template
			"redirect.html",

			// Pass the data that the page uses (in this case, 'title')
			gin.H{
				"redirectFrom":  c.Request.URL.Path,
				"redirectTo":    shortlinkObj.Spec.Target,
				"redirectAfter": shortlinkObj.Spec.RedirectAfter,
			},
		)

		span.SetAttributes(
			attribute.String("Target", shortlinkObj.Spec.Target),
			attribute.Int64("RedirectAfter", shortlinkObj.Spec.RedirectAfter),
			attribute.Int("InvocationCount", shortlinkObj.Status.Count),
		)

		err = s.client.Save(c.Request.Context(), &shortlinkObj)
		if err != nil {
			span.RecordError(err)
		}
	}
}
