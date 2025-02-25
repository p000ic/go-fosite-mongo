package mongo

import (
	// Standard Library Imports
	"context"

	// External Imports
	"github.com/ory/fosite"
	// Internal Imports
	"github.com/p000ic/go-fosite-mongo"
)

// CreateOpenIDConnectSession creates an open id connect session resource for a
// given authorize code. This is relevant for explicit open id connect flow.
func (r *RequestManager) CreateOpenIDConnectSession(ctx context.Context, authorizeCode string, request fosite.Requester) (err error) {
	// Store session request
	_, err = r.Create(ctx, storage.EntityOpenIDSessions, toMongo(authorizeCode, request))
	if err != nil {
		if err == storage.ErrResourceExists {
			return err
		}

		return err
	}

	return err
}

// GetOpenIDConnectSession gets a session resource based off the Authorize Code
// and returns a fosite.Requester, or an error.
func (r *RequestManager) GetOpenIDConnectSession(ctx context.Context, authorizeCode string, requester fosite.Requester) (request fosite.Requester, err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var sess func()
		ctx, sess, err = newSession(ctx, r.DB)
		if err != nil {
			return nil, err
		}
		defer sess()
	}

	// Get the stored request
	req, err := r.GetBySignature(ctx, storage.EntityOpenIDSessions, authorizeCode)
	if err != nil {
		if err == fosite.ErrNotFound {
			return nil, err
		}
		return nil, err
	}

	// Transform to a fosite.Request
	session := requester.GetSession()
	if session == nil {
		return nil, fosite.ErrNotFound
	}

	request, err = req.ToRequest(ctx, session, r.Clients)
	if err != nil {
		if err == fosite.ErrNotFound {
			return nil, err
		}
		return nil, err
	}

	return request, err
}

// DeleteOpenIDConnectSession removes an open id connect session from mongo.
func (r *RequestManager) DeleteOpenIDConnectSession(ctx context.Context, authorizeCode string) (err error) {
	// Remove session request
	err = r.DeleteBySignature(ctx, storage.EntityOpenIDSessions, authorizeCode)
	if err != nil {
		if err == fosite.ErrNotFound {
			return err
		}
		return err
	}
	return nil
}
