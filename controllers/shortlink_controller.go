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
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
)

var activeRedirects = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "urlshortener_active_redirects",
		Help: "Number of redirects installed for this urlshortener instance",
	},
)

func init() {
	prometheus.MustRegister(activeRedirects)
}

// ShortLinkReconciler reconciles a ShortLink object
type ShortLinkReconciler struct {
	client *shortlinkclient.ShortlinkClient
	scheme *runtime.Scheme
	log    *logr.Logger
	tracer trace.Tracer
}

func NewShortLinkReconciler(client *shortlinkclient.ShortlinkClient, scheme *runtime.Scheme, log *logr.Logger, tracer trace.Tracer) *ShortLinkReconciler {
	return &ShortLinkReconciler{
		client: client,
		scheme: scheme,
		log:    log,
		tracer: tracer,
	}
}

//+kubebuilder:rbac:groups=urlshortener.av0.de,resources=shortlinks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=urlshortener.av0.de,resources=shortlinks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=urlshortener.av0.de,resources=shortlinks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ShortLink object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *ShortLinkReconciler) Reconcile(c context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, span := r.tracer.Start(c, "ShortLinkReconciler.Reconcile", trace.WithAttributes(attribute.String("shortlink", req.Name)))
	defer span.End()

	log := r.log.WithName("reconciler").WithValues("shortlink", req.NamespacedName.String())

	shortlink, err := r.client.GetNamespaced(ctx, req.NamespacedName)
	if err != nil {
		if errors.IsNotFound(err) {
			activeRedirects.Dec()
			log.Info("Shortlink resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to fetch resource")
	}

	if shortlink.ObjectMeta.Labels == nil {
		shortlink.ObjectMeta.Labels = make(map[string]string)
	}

	labelValue, ok := shortlink.ObjectMeta.Labels["shortlink"]
	if !ok || labelValue != shortlink.Spec.Alias || !shortlink.Status.Ready {
		shortlink.ObjectMeta.Labels["shortlink"] = shortlink.Spec.Alias

		if err := r.client.Save(ctx, shortlink); err != nil {
			log.Error(err, "Failed to update ShortLink")
			return ctrl.Result{}, err
		}

		shortlink.Status.Ready = true

		if err := r.client.SaveStatus(ctx, shortlink); err != nil {
			log.Error(err, "Failed to update ShortLink status")
			return ctrl.Result{}, err
		}
	}

	if shortlinkList, err := r.client.List(ctx); shortlinkList != nil && err == nil {
		activeRedirects.Set(float64(len(shortlinkList.Items)))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShortLinkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ShortLink{}).
		Complete(r)
}
