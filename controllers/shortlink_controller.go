/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/trace"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	v1alpha1 "github.com/cedi/urlshortener/api/v1alpha1"
	shortlinkclient "github.com/cedi/urlshortener/pkg/client"
	"github.com/cedi/urlshortener/pkg/observability"
	"github.com/go-logr/logr"

	"github.com/prometheus/client_golang/prometheus"
)

var activeShortlinks = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "urlshortener_active_shortlinks",
		Help: "Number of shortlinks installed for this urlshortener instance",
	},
)

var redirectInvocations = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "urlshortener_shortlink_invocation",
		Help: "Counts of how often a shortlink was invoked",
	},
	[]string{"name", "namespace"},
)

func init() {
	metrics.Registry.MustRegister(activeShortlinks)
	metrics.Registry.MustRegister(redirectInvocations)
}

// ShortLinkReconciler reconciles a ShortLink object
type ShortLinkReconciler struct {
	client *shortlinkclient.ShortlinkClient
	scheme *runtime.Scheme
	log    *logr.Logger
	tracer trace.Tracer

	shortlinkCount int
	shortlinks     instrument.Int64UpDownCounter
	latency        instrument.Int64Histogram
}

// NewShortLinkReconciler returns a new ShortLinkReconciler
func NewShortLinkReconciler(client *shortlinkclient.ShortlinkClient, scheme *runtime.Scheme, log *logr.Logger, tracer trace.Tracer, meter metric.Meter) *ShortLinkReconciler {
	var shortlinks, _ = meter.Int64UpDownCounter(
		"urlshortener.active_shortlinks",
		instrument.WithUnit("count"),
		instrument.WithDescription("Amount of shortlinks (redirect a short-name to another URI)"),
	)

	var shortlinkReconcileLatency, _ = meter.Int64Histogram(
		"urlshortener.shortlink_controller.reconcile_latency",
		instrument.WithUnit("microseconds"),
		instrument.WithDescription("How long does the reconcile function run for"),
	)

	return &ShortLinkReconciler{
		client: client,
		scheme: scheme,
		log:    log,
		tracer: tracer,

		shortlinkCount: 0,
		shortlinks:     shortlinks,
		latency:        shortlinkReconcileLatency,
	}
}

//+kubebuilder:rbac:groups=urlshortener.cedi.dev,resources=shortlinks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=urlshortener.cedi.dev,resources=shortlinks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=urlshortener.cedi.dev,resources=shortlinks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *ShortLinkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	startTime := time.Now()

	defer func() {
		r.latency.Record(ctx, time.Since(startTime).Microseconds(), attribute.String("shortlink", req.NamespacedName.String()))
	}()

	log := r.log.WithName("reconciler").WithValues("shortlink", req.NamespacedName.String())

	span := trace.SpanFromContext(ctx)

	// Check if the span was sampled and is recording the data
	if !span.IsRecording() {
		ctx, span = r.tracer.Start(ctx, "ShortLinkReconciler.Reconcile")
		defer span.End()
	}

	span.SetAttributes(attribute.String("shortlink", req.NamespacedName.String()))

	// Get ShortLink from etcd
	shortlink, err := r.client.GetNamespaced(ctx, req.NamespacedName)
	if err != nil || shortlink == nil {
		if errors.IsNotFound(err) {
			observability.RecordInfo(span, &log, "Shortlink resource not found. Ignoring since object must be deleted")
		} else {
			observability.RecordError(span, &log, err, "Failed to fetch ShortLink resource")
		}
	}

	if shortlinkList, err := r.client.ListNamespaced(ctx, req.Namespace); shortlinkList != nil && err == nil {
		r.shortlinks.Add(ctx, int64(len(shortlinkList.Items)-r.shortlinkCount))
		r.shortlinkCount = len(shortlinkList.Items)

		activeShortlinks.Set(float64(len(shortlinkList.Items)))

		for _, shortlink := range shortlinkList.Items {
			redirectInvocations.WithLabelValues(
				shortlink.ObjectMeta.Name,
				shortlink.ObjectMeta.Namespace,
			).Set(float64(shortlink.Status.Count))
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShortLinkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ShortLink{}).
		Complete(r)
}
