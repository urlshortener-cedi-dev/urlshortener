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

// HandleListShortLink handles the listing of
// @BasePath /api/v1/
// @Summary       list shortlinks
// @Schemes       http https
// @Description   list shortlinks
// @Produce       text/plain
// @Produce       application/json
// @Success       200         {object} []ShortLink "Success"
// @Failure       401         {object} int         "Unauthorized"
// @Failure       404         {object} int         "NotFound"
// @Failure       500         {object} int         "InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/ [get]
// @Security bearerAuth
func (s *ShortlinkController) HandleListShortLink(ct *gin.Context) {
	contentType := ct.Request.Header.Get("accept")

	// Extract span from the request context
	ctx := ct.Request.Context()
	span := trace.SpanFromContext(ctx)

	// Check if the span was sampled and is recording the data
	if !span.IsRecording() {
		ctx, span = s.tracer.Start(ctx, "ShortlinkController.HandleDeleteShortLink")
		defer span.End()
	}

	span.SetAttributes(
		attribute.String("content_type", contentType),
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

	shortlinkList, err := s.authenticatedClient.List(ctx, githubUser.Login)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to list ShortLink")

		statusCode := http.StatusInternalServerError

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}

		ginReturnError(ct, statusCode, contentType, err.Error())
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
		ct.JSON(http.StatusOK, targetList)
	} else if contentType == ContentTypeTextPlain {
		shortLinks := ""
		for _, shortlink := range targetList {
			shortLinks += fmt.Sprintf("%s: %s\n", shortlink.Name, shortlink.Spec.Target)
		}
		ct.Data(http.StatusOK, contentType, []byte(shortLinks))
	}
}
