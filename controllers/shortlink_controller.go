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

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	v1alpha1 "github.com/av0de/urlshortener/api/v1alpha1"
	shortlinkclient "github.com/av0de/urlshortener/pkg/client"
	urlshortenertrace "github.com/av0de/urlshortener/pkg/tracing"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var activeRedirects = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "urlshortener_active_redirects",
		Help: "Number of redirects installed for this urlshortener instance",
	},
)

var redirectInvocations = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "urlshortener_shortlink_invocation",
		Help: "Counts of how often a shortlink was invoked",
	},
	[]string{"name", "namespace", "alias"},
)

func init() {
	metrics.Registry.MustRegister(activeRedirects, redirectInvocations)
}

// ShortLinkReconciler reconciles a ShortLink object
type ShortLinkReconciler struct {
	client *shortlinkclient.ShortlinkClient
	scheme *runtime.Scheme
	o11y   *urlshortenertrace.ShortlinkObservability
}

// NewShortLinkReconciler returns a new ShortLinkReconciler
func NewShortLinkReconciler(client *shortlinkclient.ShortlinkClient, scheme *runtime.Scheme, o11y *urlshortenertrace.ShortlinkObservability) *ShortLinkReconciler {
	return &ShortLinkReconciler{
		client: client,
		scheme: scheme,
		o11y:   o11y,
	}
}

//+kubebuilder:rbac:groups=urlshortener.av0.de,resources=shortlinks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=urlshortener.av0.de,resources=shortlinks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=urlshortener.av0.de,resources=shortlinks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *ShortLinkReconciler) Reconcile(c context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, span := r.o11y.Trace.Start(c, "ShortLinkReconciler.Reconcile", trace.WithAttributes(attribute.String("shortlink", req.Name)))
	defer span.End()

	log := r.o11y.Log.WithName("reconciler").WithValues("shortlink", req.NamespacedName.String())

	shortlink, err := r.client.GetNamespaced(ctx, req.NamespacedName)
	if err != nil {
		if errors.IsNotFound(err) {
			activeRedirects.Dec()
			urlshortenertrace.RecordInfo(span, &log, "Shortlink resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		urlshortenertrace.RecordError(span, &log, err, "Failed to fetch ShortLink resource")
		return ctrl.Result{}, err
	}

	if shortlink.ObjectMeta.Labels == nil {
		shortlink.ObjectMeta.Labels = make(map[string]string)
	}

	labelValue, ok := shortlink.ObjectMeta.Labels["shortlink"]
	if !ok || labelValue != shortlink.Spec.Alias || !shortlink.Status.Ready {
		shortlink.ObjectMeta.Labels["shortlink"] = shortlink.Spec.Alias

		if err := r.client.Save(ctx, shortlink); err != nil {
			// error here!
			urlshortenertrace.RecordError(span, &log, err, "Failed to update ShortLink")
			return ctrl.Result{}, err
		}

		shortlink.Status.Ready = true

		if err := r.client.SaveStatus(ctx, shortlink); err != nil {
			urlshortenertrace.RecordError(span, &log, err, "Failed to update ShortLink status")
			return ctrl.Result{}, err
		}
	}

	if shortlinkList, err := r.client.List(ctx); shortlinkList != nil && err == nil {
		activeRedirects.Set(float64(len(shortlinkList.Items)))

		for _, shortlink := range shortlinkList.Items {
			redirectInvocations.WithLabelValues(
				shortlink.ObjectMeta.Name,
				shortlink.ObjectMeta.Namespace,
				shortlink.Spec.Alias,
			).Set(float64(shortlink.Status.Count))
		}
	}

	urlshortenertrace.RecordInfo(span, &log, "Successfully processed shortlink %s", shortlink.ObjectMeta.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShortLinkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ShortLink{}).
		Complete(r)
}
