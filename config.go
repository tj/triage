package triage

import (
	"context"

	"github.com/tj/go-termd"
)

// Priority is user configurable priority name and label.
type Priority struct {
	// Name of the priority.
	Name string `json:"name"`

	// Label is the GitHub label name.
	Label string `json:"label"`

	// Color is the GitHub label color, for example "#532BE3".
	Color string `json:"color"`
}

// Config is the user configuration.
type Config struct {
	// Priorities is a set of priorities used in assigning. By default
	// low, medium, and high are provided.
	Priorities []Priority

	// Theme is style related configuration.
	Theme struct {
		// Code is the syntax theme used for highlighting blocks of code.
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
