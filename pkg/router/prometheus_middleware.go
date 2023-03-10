package router

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var ginDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "gin_request_duration",
		Help: "In microseconds",
	},
	[]string{
		"service",
		"path",
		"http_status_code",
	},
)

func init() {
	metrics.Registry.MustRegister(ginDuration)
}

func PromMiddleware(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		savedCtx := c.Request.Context()
		defer func() {
			c.Request = c.Request.WithContext(savedCtx)
		}()

		startTime := time.Now()

		// serve the request to the next middleware
		c.Next()

		stopTime := time.Now()

		status := fmt.Sprintf("%d", c.Writer.Status())
		ginDuration.WithLabelValues(service, c.FullPath(), status).Observe(float64(stopTime.Sub(startTime).Microseconds()))
	}
}
