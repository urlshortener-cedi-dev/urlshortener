package client

import (
	"context"
	"io/ioutil"

	"github.com/cedi/urlshortener/api/v1alpha1"
	urlshortenertrace "github.com/cedi/urlshortener/pkg/tracing"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RedirectClient is a Kubernetes client for easy CRUD operations
type RedirectClient struct {
	client client.Client
	o11y   *urlshortenertrace.ShortlinkObservability
}

// NewRedirectClient creates a new Redirect Client
func NewRedirectClient(client client.Client, o11y *urlshortenertrace.ShortlinkObservability) *RedirectClient {
	return &RedirectClient{
		client: client,
		o11y:   o11y,
	}
}

func (c *RedirectClient) Get(ct context.Context, name string) (*v1alpha1.Redirect, error) {
	ctx, span := c.o11y.Trace.Start(ct, "RedirectClient.Get", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	// try to read the namespace from /var/run
	namespace, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "Unable to read current namespace")
	}

	return c.GetNamespaced(ctx, types.NamespacedName{Name: name, Namespace: string(namespace)})
}

// GetNameNamespace returns a Redirect for a given name in a given namespace
func (c *RedirectClient) GetNameNamespace(ct context.Context, name, namespace string) (*v1alpha1.Redirect, error) {
	ctx, span := c.o11y.Trace.Start(ct, "RedirectClient.GetNameNamespace", trace.WithAttributes(attribute.String("name", name), attribute.String("namespace", namespace)))
	defer span.End()

	return c.GetNamespaced(ctx, types.NamespacedName{Name: name, Namespace: namespace})
}

// Get returns a Redirect
func (c *RedirectClient) GetNamespaced(ct context.Context, nameNamespaced types.NamespacedName) (*v1alpha1.Redirect, error) {
	ctx, span := c.o11y.Trace.Start(
		ct, "RedirectClient.GetNamespaced",
		trace.WithAttributes(
			attribute.String("name", nameNamespaced.Name),
			attribute.String("namespace", nameNamespaced.Namespace),
		),
	)
	defer span.End()

	Redirect := &v1alpha1.Redirect{}

	err := c.client.Get(ctx, nameNamespaced, Redirect)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return Redirect, nil
}

// List returns a list of all Redirect
func (c *RedirectClient) List(ct context.Context) (*v1alpha1.RedirectList, error) {
	ctx, span := c.o11y.Trace.Start(ct, "RedirectClient.List")
	defer span.End()

	Redirects := &v1alpha1.RedirectList{}

	err := c.client.List(ctx, Redirects)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return Redirects, nil
}

// List returns a list of all Redirect that match the label Redirect with the parameter label
// ToDo: Rewrite and come up with a better way. This only works client-side and is absolutely ugly and inefficient
func (c *RedirectClient) Query(ct context.Context, label string) (*v1alpha1.RedirectList, error) {
	ctx, span := c.o11y.Trace.Start(ct, "RedirectClient.Query", trace.WithAttributes(attribute.String("label", "Redirect"), attribute.String("labelValue", label)))
	defer span.End()

	Redirects := &v1alpha1.RedirectList{}

	// Like `kubectl get Redirect -l Redirect=$Redirect
	RedirectReq, _ := labels.NewRequirement("Redirect", selection.Equals, []string{label})
	selector := labels.NewSelector()
	selector = selector.Add(*RedirectReq)

	err := c.client.List(ctx, Redirects, &client.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return Redirects, nil
}

func (c *RedirectClient) Save(ct context.Context, Redirect *v1alpha1.Redirect) error {
	ctx, span := c.o11y.Trace.Start(ct, "RedirectClient.Save", trace.WithAttributes(attribute.String("Redirect", Redirect.ObjectMeta.Name), attribute.String("namespace", Redirect.ObjectMeta.Namespace)))
	defer span.End()

	err := c.client.Update(ctx, Redirect)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

func (c *RedirectClient) SaveStatus(ct context.Context, Redirect *v1alpha1.Redirect) error {
	ctx, span := c.o11y.Trace.Start(ct, "RedirectClient.SaveStatus", trace.WithAttributes(attribute.String("Redirect", Redirect.ObjectMeta.Name), attribute.String("namespace", Redirect.ObjectMeta.Namespace)))
	defer span.End()

	err := c.client.Status().Update(ctx, Redirect)
	if err != nil {
		span.RecordError(err)
	}

	return err
}
