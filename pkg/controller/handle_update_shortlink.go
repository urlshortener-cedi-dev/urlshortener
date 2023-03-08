package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cedi/urlshortener/api/v1alpha1"
	"github.com/cedi/urlshortener/pkg/observability"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

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
// @Failure       401         {object}  int     "Unauthorized"
// @Failure       404         {object}  int     "NotFound"
// @Failure       500         {object}  int     "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [put]
// @Security bearerAuth
func (s *ShortlinkController) HandleUpdateShortLink(c *gin.Context) {
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

	// When shortlink was not found
	if shortlink == nil {
		ginReturnError(c, http.StatusNotFound, contentType, "Shortlink not found")
		return
	}

	shortlinkSpec := v1alpha1.ShortLinkSpec{}

	jsonData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to read request-body")

		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if err := json.Unmarshal([]byte(jsonData), &shortlinkSpec); err != nil {
		observability.RecordError(span, s.log, err, "Failed to read ShortLink Spec JSON")

		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	shortlink.Spec = shortlinkSpec

	if err := s.authenticatedClient.Update(ctx, githubUser.Login, shortlink); err != nil {
		observability.RecordError(span, s.log, err, "Failed to update ShortLink")

		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	ginReturnError(c, http.StatusOK, contentType, "")
}
