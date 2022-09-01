package api

import (
	"fmt"
	"net/http"

	_ "github.com/av0de/urlshortener/api/v1alpha1"
	shortlinkclient "github.com/av0de/urlshortener/pkg/client"
	urlshortenertrace "github.com/av0de/urlshortener/pkg/tracing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const ContentTypeApplicationJSON = "application/json"

type ShortlinkController struct {
	o11y   *urlshortenertrace.ShortlinkObservability
	client *shortlinkclient.ShortlinkClient
}

func NewShortlinkController(o11y *urlshortenertrace.ShortlinkObservability, client *shortlinkclient.ShortlinkClient) *ShortlinkController {
	return &ShortlinkController{
		o11y:   o11y,
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
// @Router /{shortlink} [get]
func (s *ShortlinkController) HandleShortLink(c *gin.Context) {
	shortlink := c.Request.URL.Path[1:]

	// Call the HTML method of the Context to render a template
	ctx, span := s.o11y.Trace.Start(c.Request.Context(), "ShortlinkController.HandleShortLink", trace.WithAttributes(attribute.String("shortlink", shortlink)))
	defer span.End()

	span.AddEvent("shortlink", trace.WithAttributes(attribute.String("shortlink", shortlink)))

	shortlinks, err := s.client.Query(ctx, shortlink)
	if err != nil || len(shortlinks.Items) > 1 {
		if err != nil {
			span.RecordError(err)
		} else {
			span.RecordError(fmt.Errorf("more than one shortlink definition found"))
		}

		c.HTML(http.StatusInternalServerError, "500.html", gin.H{})
		return
	}

	if len(shortlinks.Items) == 0 {
		span.RecordError(fmt.Errorf("no shortlink definition '%s' found", shortlink))
		c.HTML(http.StatusNotFound, "404.html", gin.H{})
		return
	}

	shortlinkObj := shortlinks.Items[0]

	// Increase hit counter
	s.client.IncrementInvocationCount(ctx, &shortlinkObj)

	span.SetAttributes(
		attribute.String("Target", shortlinkObj.Spec.Target),
		attribute.Int64("RedirectAfter", shortlinkObj.Spec.RedirectAfter),
		attribute.Int("InvocationCount", shortlinkObj.Status.Count),
	)

	if shortlinkObj.Spec.Code != 200 {
		c.Redirect(shortlinkObj.Spec.Code, shortlinkObj.Spec.Target)
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
			"redirectTo":    shortlinkObj.Spec.Target,
			"redirectAfter": shortlinkObj.Spec.RedirectAfter,
		},
	)
}

// HandleListShortLink handles the listing of
// @BasePath /api/v1/
// @Summary       get a shortlink
// @Schemes       http https
// @Description   get a shorlink
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string  false                   "the shortlink URL part (shortlink id)" example(home)
// @Success       200         {object}  int     "Success"
// @Failure       404         {object}  int     "NotFound"
// @Failure       500         {object}  int     "InternalServerError"
// @Router /api/v1/shortlink             [get]
// @Router /api/v1/shortlink/{shortlink} [get]
func (s *ShortlinkController) HandleListShortLink(c *gin.Context) {
	c.HTML(http.StatusNotFound, "404.html", gin.H{})
}

// HandleCreateShortLink handles the creation of a shortlink and redirects according to the configuration
// @BasePath /api/v1/
// @Summary       create new shortlink
// @Schemes       http https
// @Description   create a new shorlink
// @Accept        application/json
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string                 false  "the shortlink URL part (shortlink id)" example(home)
// @Param         spec        body      v1alpha1.ShortLinkSpec true   "shortlink spec"
// @Success       200         {object}  int     "Success"
// @Success       301         {object}  int     "MovedPermanently"
// @Success       302         {object}  int     "Found"
// @Success       307         {object}  int     "TemporaryRedirect"
// @Success       308         {object}  int     "PermanentRedirect"
// @Failure       404         {object}  int     "NotFound"
// @Failure       500         {object}  int     "InternalServerError"
// @Router /api/v1/shortlink/{shortlink} [post]
func (s *ShortlinkController) HandleCreateShortLink(c *gin.Context) {
	c.HTML(http.StatusNotFound, "404.html", gin.H{})
}

// HandleDeleteShortLink handles the update of a shortlink
// @BasePath /api/v1/
// @Summary       update existing shortlink
// @Schemes       http https
// @Description   update a new shorlink
// @Accept        application/json
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string                 true   "the shortlink URL part (shortlink id)" example(home)
// @Param         spec        body      v1alpha1.ShortLinkSpec true   "shortlink spec"
// @Success       200         {object}  int     "Success"
// @Failure       404         {object}  int     "NotFound"
// @Failure       500         {object}  int     "InternalServerError"
// @Router /api/v1/shortlink/{shortlink} [put]
func (s *ShortlinkController) HandleUpdateShortLink(c *gin.Context) {
	c.HTML(http.StatusNotFound, "404.html", gin.H{})
}

// HandleDeleteShortLink handles the deletion of a shortlink
// @BasePath /api/v1/
// @Summary       delete shortlink
// @Schemes       http https
// @Description   delete shorlink
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string                 true   "the shortlink URL part (shortlink id)" example(home)
// @Success       200         {object}  int     "Success"
// @Failure       404         {object}  int     "NotFound"
// @Failure       500         {object}  int     "InternalServerError"
// @Router /api/v1/shortlink/{shortlink} [delete]
func (s *ShortlinkController) HandleDeleteShortLink(c *gin.Context) {
	c.HTML(http.StatusNotFound, "404.html", gin.H{})
}
