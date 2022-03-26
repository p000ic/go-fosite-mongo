package mongo

import (
	// Standard Library Imports
	"context"

	// External Imports
	"github.com/ory/fosite"
	// Internal Imports
	"github.com/p000ic/go-fosite-mongo"
)

// CreateRefreshTokenSession implements fosite.RefreshTokenStorage.
func (r *RequestManager) CreateRefreshTokenSession(ctx context.Context, signature string, request fosite.Requester) (err error) {
	// Store session request
	_, err = r.Create(ctx, storage.EntityRefreshTokens, toMongo(signature, request))
	if err != nil {
		if err == storage.ErrResourceExists {
			return err
		}

		return err
	}

	return nil
}

// GetRefreshTokenSession implements fosite.RefreshTokenStorage.
func (r *RequestManager) GetRefreshTokenSession(ctx context.Context, signature string, session fosite.Session) (request fosite.Requester, err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var closeSession func()
		ctx, closeSession, err = newSession(ctx, r.DB)
		if err != nil {
			return nil, err
		}
		defer closeSession()
	}
	// Get the stored request
	req, err := r.GetBySignature(ctx, storage.EntityRefreshTokens, signature)
	if err != nil {
		if err == fosite.ErrNotFound {
			return nil, err
		}
		return nil, err
	}

	// Transform to a fosite.Request
	request, err = req.ToRequest(ctx, session, r.Clients)
	if err != nil {
		if err == fosite.ErrNotFound {
			return nil, err
		}
		return nil, err
	}

	return request, nil
}

// DeleteRefreshTokenSession implements fosite.RefreshTokenStorage.
func (r *RequestManager) DeleteRefreshTokenSession(ctx context.Context, signature string) (err error) {
	// Remove session request
	err = r.DeleteBySignature(ctx, storage.EntityRefreshTokens, signature)
	if err != nil {
		if err == fosite.ErrNotFound {
			return err
		}
		return err
	}
	return nil
}
