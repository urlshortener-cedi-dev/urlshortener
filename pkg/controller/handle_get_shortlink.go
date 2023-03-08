package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cedi/urlshortener/pkg/observability"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HandleGetShortLink returns the shortlink
// @BasePath      /api/v1/
// @Summary       get a shortlink
// @Schemes       http https
// @Description   get a shortlink
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string    false          "the shortlink URL part (shortlink id)" example(home)
// @Success       200         {object}  ShortLink "Success"
// @Failure       401         {object}  int       "Unauthorized"
// @Failure       404         {object}  int       "NotFound"
// @Failure       500         {object}  int       "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [get]
// @Security bearerAuth
func (s *ShortlinkController) HandleGetShortLink(c *gin.Context) {
	shortlinkName := c.Param("shortlink")

	contentType := c.Request.Header.Get("accept")

	// Call the HTML method of the Context to render a template
	ctx, span := s.tracer.Start(c.Request.Context(), "ShortlinkController.HandleGetShortLink", trace.WithAttributes(attribute.String("shortlink", shortlinkName), attribute.String("accepted_content_type", contentType)))
	defer span.End()

	bearerToken := c.Request.Header.Get("Authorization")
	bearerToken = strings.TrimPrefix(bearerToken, "Bearer")
	bearerToken = strings.TrimPrefix(bearerToken, "token")
	if len(bearerToken) == 0 {
		err := fmt.Errorf("no credentials provided")
		span.RecordError(err)
		ginReturnError(c, http.StatusUnauthorized, contentType, err.Error())
		return
	}

	githubUser, err := getGitHubUserInfo(ctx, bearerToken)
	if err != nil {
		span.RecordError(err)
		ginReturnError(c, http.StatusUnauthorized, contentType, err.Error())
		return
	}

	shortlink, err := s.authenticatedClient.Get(ctx, githubUser.Login, shortlinkName)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to get ShortLink")

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
