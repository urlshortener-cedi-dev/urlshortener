package router

import (
	"fmt"
	"net/http"

	urlshortenercontroller "github.com/av0de/urlshortener/pkg/controller"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/trace"
)

func NewGinGonicHTTPServer(setupLog *logr.Logger, bindAddr string) (*gin.Engine, *http.Server) {
	router := gin.New()
	router.Use(otelgin.Middleware("urlshortener"))

	//load html file
	router.LoadHTMLGlob("html/templates/*.html")

	//static path
	router.Static("assets", "./html/assets")

	setupLog.Info(fmt.Sprintf("Starting gin-tonic router on binAddr: '%s'", bindAddr))
	srv := &http.Server{
		Addr:    bindAddr,
		Handler: router,
	}

	return router, srv
}

func Load(
	router *gin.Engine,
	log *logr.Logger,
	tracer trace.Tracer,
	shortlinkController *urlshortenercontroller.ShortlinkController,
) {
	router.GET("/:shortlink", shortlinkController.HandleShortLink)
}
