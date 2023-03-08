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
func (s *ShortlinkController) HandleListShortLink(c *gin.Context) {
	contentType := c.Request.Header.Get("accept")

	trace.SpanFromContext(c)

	// Call the HTML method of the Context to render a template
	ctx, span := s.tracer.Start(c.Request.Context(), "ShortlinkController.HandleListShortLink", trace.WithAttributes(attribute.String("accepted_content_type", contentType)))
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

	shortlinkList, err := s.authenticatedClient.List(ctx, githubUser.Login)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to list ShortLink")

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
