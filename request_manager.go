package storage

import (
	// Standard Library Imports
	"context"
	// External Imports
	"github.com/go-jose/go-jose/v3"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/handler/pkce"
)

// RequestManager provides an interface in order to build a compliant Fosite
// storage backend.
type RequestManager interface {
	Configure
	RequestStore
}

// RequestStore implements all fosite interfaces required to be a storage
// driver.
type RequestStore interface {
	// CoreStorage OAuth2 storage interfaces.
	oauth2.CoreStorage

	// OpenIDConnectRequestStorage OpenID storage interfaces.
	openid.OpenIDConnectRequestStorage

	// PKCERequestStorage Proof Key for Code Exchange storage interfaces.
	pkce.PKCERequestStorage

	GetPublicKey(ctx context.Context, issuer string, subject string, keyId string) (*jose.JSONWebKey, error)
	GetPublicKeys(ctx context.Context, issuer string, subject string) (*jose.JSONWebKeySet, error)
	GetPublicKeyScopes(ctx context.Context, issuer string, subject string, keyId string) ([]string, error)

	// RevokeRefreshToken Implements the rest of oauth2.TokenRevocationStorage
	RevokeRefreshToken(ctx context.Context, requestID string) error
	RevokeAccessToken(ctx context.Context, requestID string) error
	RevokeRefreshTokenMaybeGracePeriod(ctx context.Context, requestID string, signature string) error

	// Authenticate Implements the rest of oauth2.ResourceOwnerPasswordCredentialsGrantStorage
	Authenticate(ctx context.Context, username string, secret string) error

	// List Standard CRUD Storage API
	List(ctx context.Context, entityName string, filter ListRequestsRequest) ([]Request, error)
	Create(ctx context.Context, entityName string, request Request) (Request, error)
	Get(ctx context.Context, entityName string, requestID string) (Request, error)
	Update(ctx context.Context, entityName string, requestID string, request Request) (Request, error)
	Delete(ctx context.Context, entityName string, requestID string) error
	DeleteBySignature(ctx context.Context, entityName string, signature string) error
}

// ListRequestsRequest enables filtering stored Request entities.
type ListRequestsRequest struct {
	// ClientID enables filtering requests based on Client ID
	ClientID string `json:"client_id" xml:"client_id"`
	// UserID enables filtering requests based on User ID
	UserID string `json:"user_id" xml:"user_id"`
	// ScopesIntersection filters clients that have all of the listed scopes.
	// ScopesIntersection performs an AND operation.
	// If ScopesUnion is provided, a union operation will be performed as it
	// returns the wider selection.
	ScopesIntersection []string `json:"scopes_intersection" xml:"scopes_intersection"`
	// ScopesUnion filters users that have at least one of of the listed scopes.
	// ScopesUnion performs an OR operation.
	ScopesUnion []string `json:"scopes_union" xml:"scopes_union"`
	// GrantedScopesIntersection enables filtering requests based on GrantedScopes
	// GrantedScopesIntersection performs an AND operation.
	// If GrantedScopesIntersection is provided, a union operation will be
	// performed as it returns the wider selection.
	GrantedScopesIntersection []string `json:"granted_scopes_intersection" xml:"granted_scopes_intersection"`
	// GrantedScopesUnion enables filtering requests based on GrantedScopes
	// GrantedScopesUnion performs an OR operation.
	GrantedScopesUnion []string `json:"granted_scopes_union" xml:"granted_scopes_union"`
}
