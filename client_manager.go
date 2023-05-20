package storage

import (
	// Standard Library Imports
	"context"
	"time"

	// External Imports
	"github.com/ory/fosite"
)

// ClientManager provides a generic interface to clients in order to build a
// Datastore backend.
type ClientManager interface {
	Configure
	ClientStore
	AuthClientMigrator
}

// ClientStore conforms to fosite.Storage and provides methods
type ClientStore interface {
	// Storage fosite.Storage provides get client.
	fosite.Storage

	List(ctx context.Context, filter ListClientsRequest) ([]Client, error)
	Create(ctx context.Context, client Client) (Client, error)
	Get(ctx context.Context, clientID string) (Client, error)
	Update(ctx context.Context, clientID string, client Client) (Client, error)
	Delete(ctx context.Context, clientID string) error
	Authenticate(ctx context.Context, clientID string, secret string) (Client, error)
	GrantScopes(ctx context.Context, clientID string, scopes []string) (Client, error)
	RemoveScopes(ctx context.Context, clientID string, scopes []string) (Client, error)

	IsJWTUsed(ctx context.Context, jti string) (bool, error)
	MarkJWTUsedForTime(ctx context.Context, jti string, exp time.Time) error
	ClientAssertionJWTValid(_ context.Context, jti string) error
	SetClientAssertionJWT(_ context.Context, jti string, exp time.Time) error
}

// ListClientsRequest enables listing and filtering client records.
type ListClientsRequest struct {
	// AllowedTenantAccess filters clients based on an Allowed Tenant Access.
	AllowedTenantAccess string `json:"allowed_tenant_access" xml:"allowed_tenant_access"`
	// AllowedRegion filters clients based on an Allowed Region.
	AllowedRegion string `json:"allowed_region" xml:"allowed_region"`
	// RedirectURI filters clients based on redirectURI.
	RedirectURI string `json:"redirect_uri" xml:"redirect_uri"`
	// GrantType filters clients based on GrantType.
	GrantType string `json:"grant_type" xml:"grant_type"`
	// ResponseType filters clients based on ResponseType.
	ResponseType string `json:"response_type" xml:"response_type"`
	// ScopesIntersection filters clients that have at least the listed scopes.
	// ScopesIntersection performs an AND operation.
	// For example:
	// - given ["cats"] the client must have "cats" in their scopes.
	// - given ["cats, dogs"] the client must have "cats" AND "dogs in their
	//   scopes.
	//
	// If ScopesUnion is provided, a union operation will be performed as it
	// returns the wider selection.
	ScopesIntersection []string `json:"scopes_intersection" xml:"scopes_intersection"`
	// ScopesUnion filters users that have at least one of the listed scopes.
	// ScopesUnion performs an OR operation.
	// For example:
	// - given ["cats"] the client must have "cats" in their scopes.
	// - given ["cats, dogs"] the client must have "cats" OR "dogs in their
	//   scopes.
	ScopesUnion []string `json:"scopes_union" xml:"scopes_union"`
	// Contact filters clients based on Contact.
	Contact string `json:"contact" xml:"contact"`
	// Public filters clients based on Public status.
	Public bool `json:"public" xml:"public"`
	// Disabled filters clients based on denied access.
	Disabled bool `json:"disabled" xml:"disabled"`
	// Published filters clients based on published status.
	Published bool `json:"published" xml:"published"`
}
