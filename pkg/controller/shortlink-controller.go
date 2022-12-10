package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cedi/urlshortener/api/v1alpha1"
	_ "github.com/cedi/urlshortener/api/v1alpha1"
	shortlinkClient "github.com/cedi/urlshortener/pkg/client"
	"github.com/cedi/urlshortener/pkg/tracing"
	urlshortenerTrace "github.com/cedi/urlshortener/pkg/tracing"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ShortlinkController is an object who handles the requests made towards our shortlink-application
type ShortlinkController struct {
	o11y   *urlshortenerTrace.ShortlinkObservability
	client *shortlinkClient.ShortlinkClient
}

// NewShortlinkController creates a new ShortlinkController
func NewShortlinkController(o11y *urlshortenerTrace.ShortlinkObservability, client *shortlinkClient.ShortlinkClient) *ShortlinkController {
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
// @Tags default
// @Router /{shortlink} [get]
func (s *ShortlinkController) HandleShortLink(c *gin.Context) {
	shortlinkName := c.Param("shortlink")

	// Call the HTML method of the Context to render a template
	ctx, span := s.o11y.Trace.Start(c.Request.Context(), "ShortlinkController.HandleShortLink", trace.WithAttributes(attribute.String("shortlink", shortlinkName)))
	defer span.End()

	span.AddEvent("shortlink", trace.WithAttributes(attribute.String("shortlink", shortlinkName)))

	c.Header("Cache-Control", "public, max-age=900, stale-if-error=3600") // max-age = 15min; stale-if-error = 1h

	shortlink, err := s.client.Get(ctx, shortlinkName)
	if err != nil {
		tracing.RecordError(span, &s.o11y.Log, err, "Failed to get ShortLink")

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
	ctx, span := s.o11y.Trace.Start(c.Request.Context(), "ShortlinkController.HandleListShortLink", trace.WithAttributes(attribute.String("accepted_content_type", contentType)))
	defer span.End()

	shortlinkList, err := s.client.List(ctx)
	if err != nil {
		tracing.RecordError(span, &s.o11y.Log, err, "Failed to list ShortLinks")

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

// HandleGetShortLink returns the shortlink
// @BasePath      /api/v1/
// @Summary       get a shortlink
// @Schemes       http https
// @Description   get a shortlink
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string  false          "the shortlink URL part (shortlink id)" example(home)
// @Success       200         {object}  ShortLink "Success"
// @Failure       404         {object}  int       "NotFound"
// @Failure       500         {object}  int       "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [get]
func (s *ShortlinkController) HandleGetShortLink(c *gin.Context) {
	shortlinkName := c.Param("shortlink")

	contentType := c.Request.Header.Get("accept")

	// Call the HTML method of the Context to render a template
	ctx, span := s.o11y.Trace.Start(c.Request.Context(), "ShortlinkController.HandleGetShortLink", trace.WithAttributes(attribute.String("shortlink", shortlinkName), attribute.String("accepted_content_type", contentType)))
	defer span.End()

	shortlink, err := s.client.Get(ctx, shortlinkName)
	if err != nil {
		tracing.RecordError(span, &s.o11y.Log, err, "Failed to get ShortLink")

		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		ginReturnError(c, statusCode, contentType, err.Error())
		return
	}

	if contentType == ContentTypeTextPlain {
		c.Data(http.StatusOK, contentType, []byte(shortlink.Spec.Target))
	} else if contentType == ContentTypeApplicationJSON {
		c.JSON(http.StatusOK, ShortLink{
			Name:   shortlink.Name,
			Spec:   shortlink.Spec,
			Status: shortlink.Status,
		})
	}
}

// HandleCreateShortLink handles the creation of a shortlink and redirects according to the configuration
// @BasePath /api/v1/
// @Summary       create new shortlink
// @Schemes       http https
// @Description   create a new shortlink
// @Accept        application/json
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string                 	false  					"the shortlink URL part (shortlink id)" example(home)
// @Param         spec        body      v1alpha1.ShortLinkSpec 	true   					"shortlink spec"
// @Success       200         {object}  int     				"Success"
// @Success       301         {object}  int     				"MovedPermanently"
// @Success       302         {object}  int     				"Found"
// @Success       307         {object}  int     				"TemporaryRedirect"
// @Success       308         {object}  int     				"PermanentRedirect"
// @Failure       404         {object}  int     				"NotFound"
// @Failure       500         {object}  int     				"InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [post]
func (s *ShortlinkController) HandleCreateShortLink(c *gin.Context) {
	shortlinkName := c.Param("shortlink")
	contentType := c.Request.Header.Get("accept")

	// Call the HTML method of the Context to render a template
	ctx, span := s.o11y.Trace.Start(c.Request.Context(), "ShortlinkController.HandleGetShortLink", trace.WithAttributes(attribute.String("shortlink", shortlinkName), attribute.String("accepted_content_type", contentType)))
	defer span.End()

	shortlink := v1alpha1.ShortLink{
		ObjectMeta: v1.ObjectMeta{
			Name: shortlinkName,
		},
		Spec: v1alpha1.ShortLinkSpec{},
	}

	jsonData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		tracing.RecordError(span, &s.o11y.Log, err, "Unable to read request-body")
		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if err := json.Unmarshal([]byte(jsonData), &shortlink.Spec); err != nil {
		tracing.RecordError(span, &s.o11y.Log, err, "Unable to read spec-json")
		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if err := s.client.Create(ctx, &shortlink); err != nil {
		tracing.RecordError(span, &s.o11y.Log, err, "Unable to create shortlink")
		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if contentType == ContentTypeTextPlain {
		c.Data(http.StatusOK, contentType, []byte(fmt.Sprintf("%s: %s\n", shortlink.Name, shortlink.Spec.Target)))
	} else if contentType == ContentTypeApplicationJSON {
		c.JSON(http.StatusOK, ShortLink{
			Name:   shortlink.Name,
			Spec:   shortlink.Spec,
			Status: shortlink.Status,
		})
	}
}

// HandleDeleteShortLink handles the update of a shortlink
// @BasePath /api/v1/
// @Summary       update existing shortlink
// @Schemes       http https
// @Description   update a new shortlink
// @Accept        application/json
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string                 true   "the shortlink URL part (shortlink id)" example(home)
// @Param         spec        body      v1alpha1.ShortLinkSpec true   "shortlink spec"
// @Success       200         {object}  int     "Success"
// @Failure       404         {object}  int     "NotFound"
// @Failure       500         {object}  int     "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [put]
func (s *ShortlinkController) HandleUpdateShortLink(c *gin.Context) {
	shortlinkName := c.Param("shortlink")

	contentType := c.Request.Header.Get("accept")

	// Call the HTML method of the Context to render a template
	ctx, span := s.o11y.Trace.Start(c.Request.Context(), "ShortlinkController.HandleGetShortLink", trace.WithAttributes(attribute.String("shortlink", shortlinkName), attribute.String("accepted_content_type", contentType)))
	defer span.End()

	shortlink, err := s.client.Get(ctx, shortlinkName)
	if err != nil {
		tracing.RecordError(span, &s.o11y.Log, err, "Failed to get ShortLink")

		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		ginReturnError(c, statusCode, contentType, err.Error())
		return
	}

	shortlinkSpec := v1alpha1.ShortLinkSpec{}

	jsonData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		tracing.RecordError(span, &s.o11y.Log, err, "Unable to read request-body")
		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if err := json.Unmarshal([]byte(jsonData), &shortlinkSpec); err != nil {
		tracing.RecordError(span, &s.o11y.Log, err, "Failed to read ShortLink Spec JSON")
		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	shortlink.Spec = shortlinkSpec

	if err := s.client.Update(ctx, shortlink); err != nil {
		tracing.RecordError(span, &s.o11y.Log, err, "Failed to update ShortLink")
		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	ginReturnError(c, http.StatusOK, contentType, "")
}

// HandleDeleteShortLink handles the deletion of a shortlink
// @BasePath /api/v1/
// @Summary       delete shortlink
// @Schemes       http https
// @Description   delete shortlink
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string                 true   "the shortlink URL part (shortlink id)" example(home)
// @Success       200         {object}  int     "Success"
// @Failure       404         {object}  int     "NotFound"
// @Failure       500         {object}  int     "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [delete]
func (s *ShortlinkController) HandleDeleteShortLink(c *gin.Context) {
	shortlinkName := c.Param("shortlink")

	contentType := c.Request.Header.Get("accept")

	// Call the HTML method of the Context to render a template
	ctx, span := s.o11y.Trace.Start(c.Request.Context(), "ShortlinkController.HandleGetShortLink", trace.WithAttributes(attribute.String("shortlink", shortlinkName), attribute.String("accepted_content_type", contentType)))
	defer span.End()

	shortlink, err := s.client.Get(ctx, shortlinkName)
	if err != nil {
		tracing.RecordError(span, &s.o11y.Log, err, "Failed to get ShortLink")

		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		ginReturnError(c, statusCode, contentType, err.Error())
		return
	}

	if err := s.client.Delete(ctx, shortlink); err != nil {
		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		tracing.RecordError(span, &s.o11y.Log, err, "Unable to delete ShortLink")
		ginReturnError(c, statusCode, contentType, err.Error())
		return
	}
}
