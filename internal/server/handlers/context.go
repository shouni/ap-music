package handlers

import "context"

type csrfTokenContextKey struct{}

// WithCSRFToken stores the CSRF token that should be exposed to templates.
func WithCSRFToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfTokenContextKey{}, token)
}

// csrfTokenFromContext returns the CSRF token stored in the context.
func csrfTokenFromContext(ctx context.Context) string {
	token, ok := ctx.Value(csrfTokenContextKey{}).(string)
	if !ok {
		return ""
	}
	return token
}
