package client

import (
	"context"

	"github.com/cedi/urlshortener/api/v1alpha1"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slices"
)

type ShortlinkClientAuth struct {
	log    *logr.Logger
	tracer trace.Tracer
	client *ShortlinkClient
}

func NewAuthenticatedShortlinkClient(log *logr.Logger, tracer trace.Tracer, client *ShortlinkClient) *ShortlinkClientAuth {
	return &ShortlinkClientAuth{
		log:    log,
		tracer: tracer,
		client: client,
	}
}

func (c *ShortlinkClientAuth) List(ct context.Context, username string) (*v1alpha1.ShortLinkList, error) {
	ctx, span := c.tracer.Start(ct, "ShortlinkClientAuth.List")
	defer span.End()

	list, err := c.client.List(ctx)
	if err != nil {
		return nil, err
	}

	userShortlinkList := v1alpha1.ShortLinkList{
		TypeMeta: list.TypeMeta,
		ListMeta: list.ListMeta,
		Items:    make([]v1alpha1.ShortLink, 0),
	}

	for _, shortLink := range list.Items {
		if shortLink.IsOwnedBy(username) {
			userShortlinkList.Items = append(userShortlinkList.Items, shortLink)
		}
	}

	return &userShortlinkList, nil
}

func (c *ShortlinkClientAuth) Get(ct context.Context, username string, name string) (*v1alpha1.ShortLink, error) {
	ctx, span := c.tracer.Start(ct, "ShortlinkClientAuth.Get")
	defer span.End()

	shortLink, err := c.client.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	if !shortLink.IsOwnedBy(username) {
		return nil, nil
	}

	return shortLink, nil
}

func (c *ShortlinkClientAuth) Create(ct context.Context, username string, shortLink *v1alpha1.ShortLink) error {
	ctx, span := c.tracer.Start(ct, "ShortlinkClientAuth.Create")
	defer span.End()

	shortLink.Spec.Owner = username

	return c.client.Create(ctx, shortLink)
}

func (c *ShortlinkClientAuth) Update(ct context.Context, username string, shortLink *v1alpha1.ShortLink) error {
	ctx, span := c.tracer.Start(ct, "ShortlinkClientAuth.Update")
	defer span.End()

	// When someone updates a shortlink and removes himself as the owner
	// add him to the CoOwner
	if shortLink.Spec.Owner != username {
		if !slices.Contains(shortLink.Spec.CoOwners, username) {
			shortLink.Spec.CoOwners = append(shortLink.Spec.CoOwners, username)
		}
	}

	if err := c.client.Update(ctx, shortLink); err != nil {
		return err
	}

	shortLink.Status.ChangedBy = username
	return c.client.UpdateStatus(ctx, shortLink)
}

func (c *ShortlinkClientAuth) Delete(ct context.Context, username string, shortLink *v1alpha1.ShortLink) error {
	ctx, span := c.tracer.Start(ct, "ShortlinkClientAuth.Update")
	defer span.End()

	if !shortLink.IsOwnedBy(username) {
		return nil
	}

	return c.client.Delete(ctx, shortLink)
}
