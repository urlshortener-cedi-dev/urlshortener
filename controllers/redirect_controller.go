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
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	urlshortenerv1alpha1 "github.com/av0de/urlshortener/api/v1alpha1"
	redirectclient "github.com/av0de/urlshortener/pkg/client"
	redirectpkg "github.com/av0de/urlshortener/pkg/redirect"
	urlshortenertrace "github.com/av0de/urlshortener/pkg/tracing"
	"github.com/pkg/errors"
)

// RedirectReconciler reconciles a Redirect object
type RedirectReconciler struct {
	client  client.Client
	rClient *redirectclient.RedirectClient

	scheme *runtime.Scheme
	o11y   *urlshortenertrace.ShortlinkObservability
}

// NewRedirectReconciler returns a new RedirectReconciler
func NewRedirectReconciler(client client.Client, rClient *redirectclient.RedirectClient, scheme *runtime.Scheme, o11y *urlshortenertrace.ShortlinkObservability) *RedirectReconciler {
	return &RedirectReconciler{
		client:  client,
		rClient: rClient,
		scheme:  scheme,
		o11y:    o11y,
	}
}

//+kubebuilder:rbac:groups=urlshortener.av0.de,resources=redirects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=urlshortener.av0.de,resources=redirects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=urlshortener.av0.de,resources=redirects/finalizers,verbs=update

//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *RedirectReconciler) Reconcile(c context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, span := r.o11y.Trace.Start(c, "RedirectReconciler.Reconcile", trace.WithAttributes(attribute.String("redirect", req.Name)))
	defer span.End()

	log := r.o11y.Log.WithName("reconciler").WithValues("redirect", req.NamespacedName)

	// fetch redirect object
	redirect, err := r.rClient.GetNamespaced(ctx, req.NamespacedName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			urlshortenertrace.RecordInfo(span, &log, "Shortlink resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		urlshortenertrace.RecordError(span, &log, err, "Failed to fetch Redirect resource")
		return ctrl.Result{}, err
	}

	// Check if the ingress already exists, if not create a new one
	ingress, err := r.upsertRedirectIngress(ctx, redirect)
	if err != nil {
		urlshortenertrace.RecordError(span, &log, err, "Failed to upsert redirect ingress")
	}

	// Update the Redirect status with the ingress name and the target
	// List the pods for this memcached's deployment
	ingressList := &networkingv1.IngressList{}
	listOpts := []client.ListOption{
		client.InNamespace(redirect.Namespace),
		client.MatchingLabels(redirectpkg.GetLabelsForRedirect(redirect.Name)),
	}

	if err = r.client.List(ctx, ingressList, listOpts...); err != nil {
		urlshortenertrace.RecordError(span, &log, err, "Failed to list ingresses")
		return ctrl.Result{}, err
	}

	// Update status.Nodes if needed
	redirect.Status.IngressName = redirectpkg.GetIngressNames(ingressList.Items)
	redirect.Status.Target = ingress.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/permanent-redirect"]
	err = r.client.Status().Update(ctx, redirect)
	if err != nil {
		urlshortenertrace.RecordError(span, &log, err, "Failed to update Redirect status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RedirectReconciler) upsertRedirectIngress(ctx context.Context, redirect *urlshortenerv1alpha1.Redirect) (*networkingv1.Ingress, error) {
	ingress := &networkingv1.Ingress{}
	err := r.client.Get(ctx, types.NamespacedName{Name: redirect.Name, Namespace: redirect.Namespace}, ingress)
	ingress = redirectpkg.NewRedirectIngress(ingress, redirect)

	// Set Redirect instance as the owner and controller
	ctrl.SetControllerReference(redirect, ingress, r.scheme)

	if err != nil && k8serrors.IsNotFound(err) {
		if err := r.client.Create(ctx, ingress); err != nil {
			return nil, errors.Wrap(err, "Failed to create new Ingress")
		}
	} else if err != nil {
		return nil, errors.Wrap(err, "Failed to get redirect Ingress")
	}

	if err := r.client.Update(ctx, ingress); err != nil {
		return nil, errors.Wrap(err, "Failed to update redirect Ingress")
	}

	return ingress, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedirectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&urlshortenerv1alpha1.Redirect{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}
