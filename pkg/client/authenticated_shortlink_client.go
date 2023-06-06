package client

import (
	"context"

	"github.com/cedi/urlshortener/api/v1alpha1"
	"github.com/cedi/urlshortener/pkg/model"
	"go.uber.org/zap"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ShortlinkClientAuth struct {
	zapLog *zap.Logger
	tracer trace.Tracer
	client *ShortlinkClient
}

func NewAuthenticatedShortlinkClient(zapLog *zap.Logger, tracer trace.Tracer, client *ShortlinkClient) *ShortlinkClientAuth {
	return &ShortlinkClientAuth{
		zapLog: zapLog,
		tracer: tracer,
		client: client,
	}
}

func (c *ShortlinkClientAuth) List(ct context.Context, username string) (*v1alpha1.ShortLinkList, error) {
	ctx, span := c.tracer.Start(ct, "ShortlinkClientAuth.List")
	defer span.End()

	span.SetAttributes(attribute.String("username", username))

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

	span.SetAttributes(attribute.String("username", username))

	shortLink, err := c.client.Get(ctx, name)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get shortlink")
	}

	if !shortLink.IsOwnedBy(username) {
		return nil, model.NewNotAllowedError(username, "delete", shortLink.Name)
	}

	return shortLink, nil
}

func (c *ShortlinkClientAuth) Create(ct context.Context, username string, shortLink *v1alpha1.ShortLink) error {
	ctx, span := c.tracer.Start(ct, "ShortlinkClientAuth.Create")
	defer span.End()

	span.SetAttributes(attribute.String("username", username))

	shortLink.Spec.Owner = username
	return c.client.Create(ctx, shortLink)
}

func (c *ShortlinkClientAuth) Update(ct context.Context, username string, shortLink *v1alpha1.ShortLink) error {
	ctx, span := c.tracer.Start(ct, "ShortlinkClientAuth.Update")
	defer span.End()

	span.SetAttributes(attribute.String("username", username))

	if !shortLink.IsOwnedBy(username) {
		return model.NewNotAllowedError(username, "delete", shortLink.Name)
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

	span.SetAttributes(attribute.String("username", username))

	if !shortLink.IsOwnedBy(username) {
		return model.NewNotAllowedError(username, "delete", shortLink.Name)
	}

	return c.client.Delete(ctx, shortLink)
}
