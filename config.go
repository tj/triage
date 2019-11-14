package triage

import (
	"context"

	"github.com/tj/go-termd"
)

// Config .
type Config struct {
	Theme struct {
		Code *termd.SyntaxTheme `json:"code"`
	} `json:"theme"`
}

// configKey is a private context key.
type configKey struct{}

// NewConfigContext returns a new context with config.
func NewConfigContext(ctx context.Context, v *Config) context.Context {
	return context.WithValue(ctx, configKey{}, v)
}

// ConfigFromContext returns config from context.
func ConfigFromContext(ctx context.Context) (*Config, bool) {
	v, ok := ctx.Value(configKey{}).(*Config)
	return v, ok
}

// MustConfigFromContext returns config from context.
func MustConfigFromContext(ctx context.Context) *Config {
	v, ok := ctx.Value(configKey{}).(*Config)
	if !ok {
		panic("missing config in context")
	}
	return v
}
