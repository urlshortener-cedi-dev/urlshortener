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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
// @Failure       401         {object}  int                     "Unauthorized"
// @Failure       404         {object}  int     				"NotFound"
// @Failure       500         {object}  int     				"InternalServerError"
// @Tags api/v1/
// @Router /api/v1/shortlink/{shortlink} [post]
// @Security bearerAuth
func (s *ShortlinkController) HandleCreateShortLink(c *gin.Context) {
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

	shortlink := v1alpha1.ShortLink{
		ObjectMeta: v1.ObjectMeta{
			Name: shortlinkName,
		},
		Spec: v1alpha1.ShortLinkSpec{},
	}

	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		observability.RecordError(span, s.log, err, "Failed to read request-body")
		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if err := json.Unmarshal([]byte(jsonData), &shortlink.Spec); err != nil {
		observability.RecordError(span, s.log, err, "Failed to read spec-json")
		ginReturnError(c, http.StatusInternalServerError, contentType, err.Error())
		return
	}

	if err := s.authenticatedClient.Create(ctx, githubUser.Login, &shortlink); err != nil {
		observability.RecordError(span, s.log, err, "Failed to create ShortLink")
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
