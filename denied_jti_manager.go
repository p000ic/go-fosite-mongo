package storage

import (
	// Standard Library Imports
	"context"
)

// DeniedJTIManager provides a generic interface to clients in order to build a
// Datastore backend.
type DeniedJTIManager interface {
	Configure
	DeniedJTIStore
}

// DeniedJTIStore enables storing denied JWT Tokens, by ID.
type DeniedJTIStore interface {
	// Create Standard CRUD Storage API
	Create(ctx context.Context, deniedJti DeniedJTI) (DeniedJTI, error)
	Get(ctx context.Context, jti string) (DeniedJTI, error)
	Delete(ctx context.Context, jti string) error
	// DeleteBefore removes all denied JTIs before the given unix time.
	DeleteBefore(ctx context.Context, expBefore int64) error

	// IsJWTUsed(ctx context.Context, jti string) (bool, error)
	// MarkJWTUsedForTime(ctx context.Context, jti string, exp time.Time) error
	// ClientAssertionJWTValid(_ context.Context, jti string) error
	// SetClientAssertionJWT(_ context.Context, jti string, exp time.Time) error
}
