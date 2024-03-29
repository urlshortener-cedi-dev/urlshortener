package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cedi/urlshortener/pkg/observability"
	"github.com/gin-gonic/gin"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// HandleDeleteShortLink handles the deletion of a shortlink
// @BasePath /api/v1/
// @Summary       delete shortlink
// @Schemes       http https
// @Description   delete shortlink
// @Produce       text/plain
// @Produce       application/json
// @Param         shortlink   path      string                 true   "the shortlink URL part (shortlink id)" example(home)
// @Success       200         {object}  int     "Success"
// @Failure       401         {object}  int     "Unauthorized"
// @Failure       404         {object}  int     "NotFound"
// @Failure       500         {object}  int     "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [delete]
// @Security bearerAuth
func (s *ShortlinkController) HandleDeleteShortLink(ct *gin.Context) {
	shortlinkName := ct.Param("shortlink")
	contentType := ct.Request.Header.Get("accept")

	ctx := ct.Request.Context()
	span := trace.SpanFromContext(ctx)

	// Check if the span was sampled and is recording the data
	if !span.IsRecording() {
		ctx, span = s.tracer.Start(ctx, "ShortlinkController.HandleDeleteShortLink")
		defer span.End()
	}

	span.SetAttributes(
		attribute.String("shortlink", shortlinkName),
		attribute.String("content_type", contentType),
		attribute.String("referrer", ct.Request.Referer()),
	)

	log := otelzap.L().Sugar().With(zap.String("shortlink", shortlinkName),
		zap.String("operation", "delete"),
	)

	bearerToken := ct.Request.Header.Get("Authorization")
	bearerToken = strings.TrimPrefix(bearerToken, "Bearer")
	bearerToken = strings.TrimPrefix(bearerToken, "token")
	if len(bearerToken) == 0 {
		err := fmt.Errorf("no credentials provided")
		observability.RecordError(ctx, span, log, err, "no credentials provided")
		ginReturnError(ct, http.StatusUnauthorized, contentType, err.Error())
		return
	}

	githubUser, err := getGitHubUserInfo(ctx, bearerToken)
	if err != nil {
		observability.RecordError(ctx, span, log, err, "GitHub User Info invalid")
		ginReturnError(ct, http.StatusUnauthorized, contentType, err.Error())
		return
	}

	shortlink, err := s.authenticatedClient.Get(ctx, githubUser.Login, shortlinkName)
	if err != nil {
		observability.RecordError(ctx, span, log, err, "Failed to get ShortLink")

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

	if err := s.authenticatedClient.Delete(ctx, githubUser.Login, shortlink); err != nil {
		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		observability.RecordError(ctx, span, log, err, "Failed to delete ShortLink")

		ginReturnError(ct, statusCode, contentType, err.Error())
		return
	}
}
