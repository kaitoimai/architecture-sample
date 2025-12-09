package auth

import "context"

// contextKey is the type used for context keys to avoid collisions
type contextKey string

const (
	// claimsKey is the context key for storing JWT claims
	claimsKey contextKey = "jwt_claims"
)

// NewContext returns a new context with the given claims
func NewContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// FromContext extracts claims from the context
// Returns nil if no claims are found
func FromContext(ctx context.Context) *Claims {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// MustFromContext extracts claims from the context
// Panics if no claims are found (use only when claims are guaranteed to exist)
func MustFromContext(ctx context.Context) *Claims {
	claims := FromContext(ctx)
	if claims == nil {
		panic("claims not found in context")
	}
	return claims
}
