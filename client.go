package triage

import (
	"context"

	"github.com/google/go-github/v28/github"
)

// clientKey is a private context key.
type clientKey struct{}

// NewClientContext returns a new context with client.
func NewClientContext(ctx context.Context, v *github.Client) context.Context {
	return context.WithValue(ctx, clientKey{}, v)
}

// ClientFromContext returns client from context.
func ClientFromContext(ctx context.Context) (*github.Client, bool) {
	v, ok := ctx.Value(clientKey{}).(*github.Client)
	return v, ok
}

// MustClientFromContext returns client from context.
func MustClientFromContext(ctx context.Context) *github.Client {
	v, ok := ctx.Value(clientKey{}).(*github.Client)
	if !ok {
		panic("missing github client in context")
	}
	return v
}
