package mongo

import (
	// Standard Library Imports
	"context"
	"errors"

	// External Imports
	"github.com/ory/fosite"

	// Internal Imports
	"github.com/p000ic/go-fosite-mongo"
)

// CreateAccessTokenSession creates a new session for an Access Token
func (r *RequestManager) CreateAccessTokenSession(ctx context.Context, signature string, request fosite.Requester) (err error) {
	// Store session request
	_, err = r.Create(ctx, storage.EntityAccessTokens, toMongo(signature, request))
	if err != nil {
		if errors.Is(err, storage.ErrResourceExists) {
			return err
		}
		return err
	}
	return err
}

// GetAccessTokenSession returns a session if it can be found by signature
func (r *RequestManager) GetAccessTokenSession(ctx context.Context, signature string, session fosite.Session) (request fosite.Requester, err error) {
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
	req, err := r.GetBySignature(ctx, storage.EntityAccessTokens, signature)
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

	return request, err
}

// DeleteAccessTokenSession removes an Access Token's session
func (r *RequestManager) DeleteAccessTokenSession(ctx context.Context, signature string) (err error) {
	// Remove session request
	err = r.DeleteBySignature(ctx, storage.EntityAccessTokens, signature)
	if err != nil {
		if err == fosite.ErrNotFound {
			return err
		}
		return err
	}

	return nil
}
