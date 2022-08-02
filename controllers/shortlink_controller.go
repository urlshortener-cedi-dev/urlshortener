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
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	v1alpha1 "github.com/av0de/urlshortener/api/v1alpha1"
	shortlinkclient "github.com/av0de/urlshortener/pkg/client"
	"github.com/go-logr/logr"
)

// ShortLinkReconciler reconciles a ShortLink object
type ShortLinkReconciler struct {
	*shortlinkclient.ShortlinkClient
	Scheme *runtime.Scheme
	Log    *logr.Logger
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
func (r *ShortLinkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithName("reconciler").WithValues("shortlink", req.NamespacedName.String())

	shortlink, err := r.GetNamespaced(ctx, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Shortlink resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to fetch resource")
	}

	log.Info(fmt.Sprintf("Reconciling Shortlink %s (labels=%v)", shortlink.ObjectMeta.Name, shortlink.ObjectMeta.Labels))

	if shortlink.ObjectMeta.Labels == nil {
		shortlink.ObjectMeta.Labels = make(map[string]string)
	}

	if value, ok := shortlink.ObjectMeta.Labels["shortlink"]; !ok || value != shortlink.Spec.Alias {
		shortlink.ObjectMeta.Labels["shortlink"] = shortlink.Spec.Alias
		shortlink.Status.Ready = true

		if err := r.Save(ctx, shortlink); err != nil {
			log.Error(err, "Failed to update ShortLink")
			return ctrl.Result{}, err
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
