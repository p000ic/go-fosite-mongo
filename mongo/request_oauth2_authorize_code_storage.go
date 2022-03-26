package mongo

import (
	// Standard Library Imports
	"context"
	"time"

	// External Imports
	"github.com/ory/fosite"
	// Internal Imports
	"github.com/p000ic/go-fosite-mongo"
)

// CreateAuthorizeCodeSession stores the authorization request for a given
// authorization code.
func (r *RequestManager) CreateAuthorizeCodeSession(ctx context.Context, code string, request fosite.Requester) (err error) {
	// Store session request
	_, err = r.Create(ctx, storage.EntityAuthorizationCodes, toMongo(code, request))
	if err != nil {
		if err == storage.ErrResourceExists {
			return err
		}
		return err
	}

	return err
}

// GetAuthorizeCodeSession hydrates the session based on the given code and
// returns the authorization request.
func (r *RequestManager) GetAuthorizeCodeSession(ctx context.Context, code string, session fosite.Session) (request fosite.Requester, err error) {
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
	req, err := r.GetBySignature(ctx, storage.EntityAuthorizationCodes, code)
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
	if !req.Active {
		// If the authorization code has been invalidated with
		// `InvalidateAuthorizeCodeSession`, this method should return the
		// ErrInvalidatedAuthorizeCode error.
		// Make sure to also return the fosite.Requester value when returning
		// the ErrInvalidatedAuthorizeCode error!
		return request, fosite.ErrInvalidatedAuthorizeCode
	}
	return request, err
}

// InvalidateAuthorizeCodeSession is called when an authorize code is being
// used. The state of the authorization code should be set to invalid and
// consecutive requests to GetAuthorizeCodeSession should return the
// ErrInvalidatedAuthorizeCode error.
func (r *RequestManager) InvalidateAuthorizeCodeSession(ctx context.Context, code string) (err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var closeSession func()
		ctx, closeSession, err = newSession(ctx, r.DB)
		if err != nil {
			return err
		}
		defer closeSession()
	}
	// Get the stored request
	req, err := r.GetBySignature(ctx, storage.EntityAuthorizationCodes, code)
	if err != nil {
		if err == fosite.ErrNotFound {
			return err
		}
		// Log to StdOut
		return err
	}

	// InvalidateAuthorizeCodeSession
	req.UpdateTime = time.Now().Unix()
	req.Active = false

	// Push the update back
	req, err = r.Update(ctx, storage.EntityAuthorizationCodes, req.ID, req)
	if err != nil {
		if err == fosite.ErrNotFound {
			return err
		}
		return err
	}

	return nil
}
