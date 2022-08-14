package client

import (
	"context"
	"io/ioutil"

	"github.com/av0de/urlshortener/api/v1alpha1"
	urlshortenertrace "github.com/av0de/urlshortener/pkg/tracing"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ShortlinkClient is a Kubernetes client for easy CRUD operations
type ShortlinkClient struct {
	client client.Client
	o11y   *urlshortenertrace.ShortlinkObservability
}

// NewShortlinkClient creates a new shortlink Client
func NewShortlinkClient(client client.Client, o11y *urlshortenertrace.ShortlinkObservability) *ShortlinkClient {
	return &ShortlinkClient{
		client: client,
		o11y:   o11y,
	}
}

func (c *ShortlinkClient) Get(ct context.Context, name string) (*v1alpha1.ShortLink, error) {
	ctx, span := c.o11y.Trace.Start(ct, "ShortlinkClient.Get", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	// try to read the namespace from /var/run
	namespace, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "Unable to read current namespace")
	}

	return c.GetNamespaced(ctx, types.NamespacedName{Name: name, Namespace: string(namespace)})
}

// GetNameNamespace returns a Shortlink for a given name in a given namespace
func (c *ShortlinkClient) GetNameNamespace(ct context.Context, name, namespace string) (*v1alpha1.ShortLink, error) {
	ctx, span := c.o11y.Trace.Start(ct, "ShortlinkClient.GetNameNamespace", trace.WithAttributes(attribute.String("name", name), attribute.String("namespace", namespace)))
	defer span.End()

	return c.GetNamespaced(ctx, types.NamespacedName{Name: name, Namespace: namespace})
}

// Get returns a Shortlink
func (c *ShortlinkClient) GetNamespaced(ct context.Context, nameNamespaced types.NamespacedName) (*v1alpha1.ShortLink, error) {
	ctx, span := c.o11y.Trace.Start(
		ct, "ShortlinkClient.GetNamespaced",
		trace.WithAttributes(
			attribute.String("name", nameNamespaced.Name),
			attribute.String("namespace", nameNamespaced.Namespace),
		),
	)
	defer span.End()

	shortlink := &v1alpha1.ShortLink{}

	err := c.client.Get(ctx, nameNamespaced, shortlink)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return shortlink, nil
}

// List returns a list of all Shortlink
func (c *ShortlinkClient) List(ct context.Context) (*v1alpha1.ShortLinkList, error) {
	ctx, span := c.o11y.Trace.Start(ct, "ShortlinkClient.List")
	defer span.End()

	shortlinks := &v1alpha1.ShortLinkList{}

	err := c.client.List(ctx, shortlinks)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return shortlinks, nil
}

// List returns a list of all Shortlink that match the label shortlink with the parameter label
// ToDo: Rewrite and come up with a better way. This only works client-side and is absolutely ugly and inefficient
func (c *ShortlinkClient) Query(ct context.Context, label string) (*v1alpha1.ShortLinkList, error) {
	ctx, span := c.o11y.Trace.Start(ct, "ShortlinkClient.Query", trace.WithAttributes(attribute.String("label", "shortlink"), attribute.String("labelValue", label)))
	defer span.End()

	shortlinks := &v1alpha1.ShortLinkList{}

	// Like `kubectl get shortlink -l shortlink=$shortlink
	shortlinkReq, _ := labels.NewRequirement("shortlink", selection.Equals, []string{label})
	selector := labels.NewSelector()
	selector = selector.Add(*shortlinkReq)

	err := c.client.List(ctx, shortlinks, &client.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return shortlinks, nil
}

func (c *ShortlinkClient) Save(ct context.Context, shortlink *v1alpha1.ShortLink) error {
	ctx, span := c.o11y.Trace.Start(ct, "ShortlinkClient.Save", trace.WithAttributes(attribute.String("shortlink", shortlink.ObjectMeta.Name), attribute.String("namespace", shortlink.ObjectMeta.Namespace)))
	defer span.End()

	err := c.client.Update(ctx, shortlink)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

func (c *ShortlinkClient) SaveStatus(ct context.Context, shortlink *v1alpha1.ShortLink) error {
	ctx, span := c.o11y.Trace.Start(ct, "ShortlinkClient.SaveStatus", trace.WithAttributes(attribute.String("shortlink", shortlink.ObjectMeta.Name), attribute.String("namespace", shortlink.ObjectMeta.Namespace)))
	defer span.End()

	err := c.client.Status().Update(ctx, shortlink)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

func (c *ShortlinkClient) IncrementInvocationCount(ct context.Context, shortlink *v1alpha1.ShortLink) error {
	ctx, span := c.o11y.Trace.Start(ct, "ShortlinkClient.SaveStatus", trace.WithAttributes(attribute.String("shortlink", shortlink.ObjectMeta.Name), attribute.String("namespace", shortlink.ObjectMeta.Namespace)))
	defer span.End()

	shortlink.Status.Count = shortlink.Status.Count + 1

	err := c.client.Status().Update(ctx, shortlink)
	if err != nil {
		span.RecordError(err)
	}

	return err
}
