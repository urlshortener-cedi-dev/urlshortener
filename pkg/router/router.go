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
	router.Static("html/assets", "./assets")

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
	group := router.Group("/")
	group.GET("/", func(c *gin.Context) {
		// Call the HTML method of the Context to render a template
		log.WithName("controllers").WithName("Redirect").Info("Handling request for /")
		_, span := tracer.Start(c.Request.Context(), "/")
		defer span.End()

		c.HTML(
			// Set the HTTP status to 200 (OK)
			http.StatusOK,
			// Use the index.html template
			"index.html",
			// Pass the data that the page uses (in this case, 'title')
			gin.H{
				"title": "Home Page",
			},
		)
	})

	router.GET("/:shortlink", shortlinkController.HandleShortLink)
}
