package registry

import "context"

// Option is a functional option that configures the registry
type Option func(reg *registry)

// WithContext is a functional option that configures a context for the registry
func WithContext(ctx context.Context) Option {
	return func(reg *registry) {
		reg.ctx = ctx
	}
}
