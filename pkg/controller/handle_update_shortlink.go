package controller

import (
	"encoding/json"
	"fmt"
	"io"
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
func (s *ShortlinkController) HandleUpdateShortLink(ct *gin.Context) {
	shortlinkName := ct.Param("shortlink")
	contentType := ct.Request.Header.Get("accept")

	ctx := ct.Request.Context()
	span := trace.SpanFromContext(ctx)

	// Check if the span was sampled and is recording the data
	if !span.IsRecording() {
		ctx, span = s.tracer.Start(ctx, "ShortlinkController.HandleShortLink")
		defer span.End()
	}

	span.SetAttributes(
		attribute.String("shortlink", shortlinkName),
		attribute.String("referrer", ct.Request.Referer()),
	)

	bearerToken := ct.Request.Header.Get("Authorization")
	bearerToken = strings.TrimPrefix(bearerToken, "Bearer")
	bearerToken = strings.TrimPrefix(bearerToken, "token")
	if len(bearerToken) == 0 {
		err := fmt.Errorf("no credentials provided")
		span.RecordError(err)
		ginReturnError(ct, http.StatusUnauthorized, contentType, err.Error())
		return
	}

	githubUser, err := getGitHubUserInfo(ctx, bearerToken)
	if err != nil {
		span.RecordError(err)
		ginReturnError(ct, http.StatusUnauthorized, contentType, err.Error())
		return
	}

	shortlink, err := s.authenticatedClient.Get(ctx, githubUser.Login, shortlinkName)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to get ShortLink")

		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		ginReturnError(ct, statusCode, contentType, err.Error())
		return
	}

	// When shortlink was not found
	if shortlink == nil {
		ginReturnError(ct, http.StatusNotFound, contentType, "Shortlink not found")
		return
	}

	shortlinkSpec := v1alpha1.ShortLinkSpec{}

	jsonData, err := io.ReadAll(ct.Request.Body)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to read request-body")

		ginReturnError(ct, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if err := json.Unmarshal([]byte(jsonData), &shortlinkSpec); err != nil {
		observability.RecordError(span, s.log, err, "Failed to read ShortLink Spec JSON")

		ginReturnError(ct, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	shortlink.Spec = shortlinkSpec

	if err := s.authenticatedClient.Update(ctx, githubUser.Login, shortlink); err != nil {
		observability.RecordError(span, s.log, err, "Failed to update ShortLink")

		ginReturnError(ct, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	ginReturnError(ct, http.StatusOK, contentType, "")
}
