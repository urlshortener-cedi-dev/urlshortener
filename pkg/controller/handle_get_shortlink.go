package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cedi/urlshortener/pkg/observability"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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
func (s *ShortlinkController) HandleGetShortLink(ct *gin.Context) {
	shortlinkName := ct.Param("shortlink")
	contentType := ct.Request.Header.Get("accept")

	ctx := ct.Request.Context()
	span := trace.SpanFromContext(ctx)

	// Check if the span was sampled and is recording the data
	if !span.IsRecording() {
		ctx, span = s.tracer.Start(ctx, "ShortlinkController.HandleGetShortLink")
		defer span.End()
	}

	span.SetAttributes(
		attribute.String("shortlink", shortlinkName),
		attribute.String("content_type", contentType),
		attribute.String("referrer", ct.Request.Referer()),
	)

	log := s.zapLog.Sugar().With(zap.String("shortlink", shortlinkName),
		zap.String("operation", "create"),
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
		observability.RecordError(span, log, err, "Failed to get ShortLink")

		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		ginReturnError(ct, statusCode, contentType, err.Error())
		return
	}

	if contentType == ContentTypeTextPlain {
		ct.Data(http.StatusOK, contentType, []byte(shortlink.Spec.Target))
	} else if contentType == ContentTypeApplicationJSON {
		ct.JSON(http.StatusOK, ShortLink{
			Name:   shortlink.Name,
			Spec:   shortlink.Spec,
			Status: shortlink.Status,
		})
	}
}
