package controller

import (
	"fmt"
	"net/http"
	"strings"

	shortlinkClient "github.com/cedi/urlshortener/pkg/client"
	"github.com/cedi/urlshortener/pkg/observability"
	"github.com/go-logr/logr"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ShortlinkController is an object who handles the requests made towards our shortlink-application
type ShortlinkController struct {
	client *shortlinkClient.ShortlinkClient
	log    *logr.Logger
	tracer trace.Tracer
}

// NewShortlinkController creates a new ShortlinkController
func NewShortlinkController(log *logr.Logger, tracer trace.Tracer, client *shortlinkClient.ShortlinkClient) *ShortlinkController {
	return &ShortlinkController{
		log:    log,
		tracer: tracer,
		client: client,
	}
}

// HandleShortlink handles the shortlink and redirects according to the configuration
// @BasePath /
// @Summary       redirect to target
// @Schemes       http https
// @Description   redirect to target as per configuration of the shortlink
// @Produce       text/html
// @Param         shortlink   path      string  true  "shortlink id"
// @Success       200         {object}  int     "Success"
// @Success       300         {object}  int     "MultipleChoices"
// @Success       301         {object}  int     "MovedPermanently"
// @Success       302         {object}  int     "Found"
// @Success       303         {object}  int     "SeeOther"
// @Success       304         {object}  int     "NotModified"
// @Success       305         {object}  int     "UseProxy"
// @Success       307         {object}  int     "TemporaryRedirect"
// @Success       308         {object}  int     "PermanentRedirect"
// @Failure       404         {object}  int     "NotFound"
// @Failure       500         {object}  int     "InternalServerError"
// @Tags default
// @Router /{shortlink} [get]
func (s *ShortlinkController) HandleShortLink(c *gin.Context) {
	shortlinkName := c.Param("shortlink")

	// Call the HTML method of the Context to render a template
	ctx, span := s.tracer.Start(c.Request.Context(), "ShortlinkController.HandleShortLink", trace.WithAttributes(attribute.String("shortlink", shortlinkName)))
	defer span.End()

	span.AddEvent("shortlink", trace.WithAttributes(attribute.String("shortlink", shortlinkName)))

	c.Header("Cache-Control", "public, max-age=900, stale-if-error=3600") // max-age = 15min; stale-if-error = 1h

	shortlink, err := s.client.Get(ctx, shortlinkName)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to get ShortLink")

		if strings.Contains(err.Error(), "not found") {
			c.HTML(http.StatusNotFound, "404.html", gin.H{})
		} else {
			c.HTML(http.StatusInternalServerError, "500.html", gin.H{})
		}
		return
	}

	// Increase hit counter
	s.client.IncrementInvocationCount(ctx, shortlink)

	span.SetAttributes(
		attribute.String("Target", shortlink.Spec.Target),
		attribute.Int64("RedirectAfter", shortlink.Spec.RedirectAfter),
		attribute.Int("InvocationCount", shortlink.Status.Count),
	)

	target := shortlink.Spec.Target

	if !strings.HasPrefix(target, "http") {
		target = fmt.Sprintf("http://%s", target)

		span.AddEvent("change prefix", trace.WithAttributes(
			attribute.String("from", shortlink.Spec.Target),
			attribute.String("to", target),
		))
	}

	if shortlink.Spec.Code != 200 {
		c.Redirect(shortlink.Spec.Code, target)
		return
	}

	c.HTML(
		// Set the HTTP status to 200 (OK)
		http.StatusOK,

		// Use the index.html template
		"redirect.html",

		// Pass the data that the page uses (in this case, 'title')
		gin.H{
			"redirectFrom":  c.Request.URL.Path,
			"redirectTo":    target,
			"redirectAfter": shortlink.Spec.RedirectAfter,
		},
	)
}

// HandleListShortLink handles the listing of
// @BasePath /api/v1/
// @Summary       list shortlinks
// @Schemes       http https
// @Description   list shortlinks
// @Produce       text/plain
// @Produce       application/json
// @Success       200         {object} []ShortLink "Success"
// @Failure       404         {object} int         "NotFound"
// @Failure       500         {object} int         "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/ [get]
func (s *ShortlinkController) HandleListShortLink(c *gin.Context) {
	contentType := c.Request.Header.Get("accept")

	// Call the HTML method of the Context to render a template
	ctx, span := s.tracer.Start(c.Request.Context(), "ShortlinkController.HandleListShortLink", trace.WithAttributes(attribute.String("accepted_content_type", contentType)))
	defer span.End()

	shortlinkList, err := s.client.List(ctx)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to list ShortLinks")

		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		ginReturnError(c, statusCode, contentType, err.Error())
		return
	}

	targetList := make([]ShortLink, len(shortlinkList.Items))

	for idx, shortlink := range shortlinkList.Items {
		targetList[idx] = ShortLink{
			Name:   shortlink.ObjectMeta.Name,
			Spec:   shortlink.Spec,
			Status: shortlink.Status,
		}
	}

	if contentType == ContentTypeApplicationJSON {
		c.JSON(http.StatusOK, targetList)
	} else if contentType == ContentTypeTextPlain {
		shortLinks := ""
		for _, shortlink := range targetList {
			shortLinks += fmt.Sprintf("%s: %s\n", shortlink.Name, shortlink.Spec.Target)
		}
		c.Data(http.StatusOK, contentType, []byte(shortLinks))
	}
}
