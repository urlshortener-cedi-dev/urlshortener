package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cedi/urlshortener/api/v1alpha1"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	ContentTypeApplicationJSON = "application/json"
	ContentTypeTextPlain       = "text/plain"
)

type ShortLink struct {
	Name   string                   `json:"name"`
	Spec   v1alpha1.ShortLinkSpec   `json:"spec,omitempty"`
	Status v1alpha1.ShortLinkStatus `json:"status,omitempty"`
}

type JsonReturnError struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

type GithubUser struct {
	Id         int    `json:"id,omitempty"`
	Login      string `json:"login,omitempty"`
	Avatar_url string `json:"avatar_url,omitempty"`
	Type       string `json:"type,omitempty"`
	Name       string `json:"name,omitempty"`
	Email      string `json:"email,omitempty"`
}

func ginReturnError(c *gin.Context, statusCode int, contentType string, err string) {
	if contentType == ContentTypeTextPlain {
		c.Data(statusCode, contentType, []byte(err))
	} else if contentType == ContentTypeApplicationJSON {
		c.JSON(statusCode, JsonReturnError{
			Code:  statusCode,
			Error: err,
		})
	}
}

func getGitHubUserInfo(c context.Context, bearerToken string) (*GithubUser, error) {
	// prepare request to GitHubs User endpoint
	req, err := http.NewRequestWithContext(c, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to build request to fetch GitHub API")
	}

	// Set headers
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	req.Header.Add("Authorization", "token "+bearerToken)

	// Perform request
	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch UserInfo from GitHub API")
	}
	defer resp.Body.Close()

	// If request was unsuccessful, we error out
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad credentials")
	}

	// If successful, we read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Error while reading the response")
	}

	// And parse it in our GithubUser model
	githubUser := &GithubUser{}
	err = json.Unmarshal(body, githubUser)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal GitHub UserInfo")
	}

	return githubUser, nil
}
