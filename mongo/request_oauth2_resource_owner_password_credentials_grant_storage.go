package mongo

import (
	// Standard Library Imports
	"context"
	"errors"

	// External Imports
	"github.com/ory/fosite"
)

// Provides a concrete implementation of oauth2.ResourceOwnerPasswordCredentialsGrantStorage
// oauth2.ResourceOwnerPasswordCredentialsGrantStorage also implements
// oauth2.AccessTokenStorage and oauth2.RefreshTokenStorage

// Authenticate confirms whether the specified password matches the stored
// hashed password within a User resource, found by username.
func (r *RequestManager) Authenticate(ctx context.Context, username string, secret string) (string, error) {
	user, err := r.Users.Authenticate(ctx, username, secret)
	if err != nil {
		if errors.Is(err, fosite.ErrNotFound) {
			return "", err
		}
		return "", err
	}

	return user.GetID(), nil
}
