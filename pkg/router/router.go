package router

import (
	"fmt"
	"net/http"

	docs "github.com/cedi/urlshortener/docs"
	urlShortenerController "github.com/cedi/urlshortener/pkg/controller"

	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/gin"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title 			URL Shortener
// @version         1.0
// @description     A url shortener, written in Go running on Kubernetes

// @contact.name   Cedric Kienzler
// @contact.url    cedi.dev
// @contact.email  urlshortener-api@cedi.dev

// @license.name  	Apache 2.0
// @license.url   	http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /

// @securityDefinitions.apiKey bearerAuth
// @in header
// @name Authorization

func NewGinGonicHTTPServer(setupLog *logr.Logger, bindAddr, serviceName string) (*gin.Engine, *http.Server) {
	router := gin.New()
	router.Use(
		otelgin.Middleware(serviceName),
		secure.Secure(secure.Options{
			SSLRedirect:           true,
			SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
			STSIncludeSubdomains:  true,
			FrameDeny:             true,
			ContentTypeNosniff:    true,
			BrowserXssFilter:      true,
			ContentSecurityPolicy: "default-src 'self' data: 'unsafe-inline'",
		}),
	)

	//load html file
	router.LoadHTMLGlob("html/templates/*.html")

	//static path
	router.Static("assets", "./html/assets")

	setupLog.Info(fmt.Sprintf("Starting gin-tonic router on binAddr: '%s'", bindAddr))
	srv := &http.Server{
		Addr:    bindAddr,
		Handler: router,
	}

	docs.SwaggerInfo.BasePath = "/"

	return router, srv
}

func Load(router *gin.Engine, shortlinkController *urlShortenerController.ShortlinkController) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.GET("/:shortlink", shortlinkController.HandleShortLink)

	{
		v1 := router.Group("/api/v1")
		v1.GET("/shortlink/", shortlinkController.HandleListShortLink)
		v1.GET("/shortlink/:shortlink", shortlinkController.HandleGetShortLink)
		v1.POST("/shortlink/:shortlink", shortlinkController.HandleCreateShortLink)
		v1.PUT("/shortlink/:shortlink", shortlinkController.HandleUpdateShortLink)
		v1.DELETE("/shortlink/:shortlink", shortlinkController.HandleDeleteShortLink)
	}
}
