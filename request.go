package storage

import (
	// Standard Library Imports
	"context"
	"encoding/json"
	"log"
	"net/url"
	"time"

	// External Imports
	"github.com/google/uuid"
	"github.com/ory/fosite"
	"github.com/pkg/errors"
)

// Request is a concrete implementation of a fosite.Requester, extended to
// support the required data for OAuth2 and OpenID.
type Request struct {
	// ID contains the unique request identifier.
	ID string `bson:"id" json:"id" xml:"id"`
	// CreateTime is when the resource was created in seconds from the epoch.
	CreateTime int64 `bson:"created_at" json:"createTime" xml:"createTime"`
	// UpdateTime is the last time the resource was modified in seconds from
	// the epoch.
	UpdateTime int64 `bson:"updated_at" json:"updateTime" xml:"updateTime"`
	// RequestedAt is the time the request was made.
	RequestedAt time.Time `bson:"requested_at" json:"requestedAt" xml:"requestedAt"`
	// Signature contains a unique session signature.
	Signature string `bson:"signature" json:"signature" xml:"signature"`
	// ClientID contains a link to the Client that was used to authenticate
	// this session.
	ClientID string `bson:"client_id" json:"clientId" xml:"clientId"`
	// UserID contains the subject's unique ID which links back to a stored
	// user account.
	UserID string `bson:"user_id" json:"userId" xml:"userId"`
	// Scopes contains the scopes that the user requested.
	RequestedScope fosite.Arguments `bson:"scopes" json:"scopes" xml:"scopes"`
	// GrantedScope contains the list of scopes that the user was actually
	// granted.
	GrantedScope fosite.Arguments `bson:"granted_scopes" json:"grantedScopes" xml:"grantedScopes"`
	// RequestedAudience contains the audience the user requested.
	RequestedAudience fosite.Arguments `bson:"requested_audience" json:"requestedAudience" xml:"requestedAudience"`
	// GrantedAudience contains the list of audiences the user was actually
	// granted.
	GrantedAudience fosite.Arguments `bson:"granted_audience" json:"grantedAudience" xml:"grantedAudience"`
	// Form contains the url values that were passed in to authenticate the
	// user's client session.
	Form url.Values `bson:"form_data" json:"formData" xml:"formData"`
	// Active is specifically used for Authorize Code flow revocation.
	Active bool `bson:"active" json:"active" xml:"active"`
	// Session contains the session data. The underlying structure differs
	// based on OAuth strategy, so we need to store it as binary-encoded JSON.
	// Otherwise, it can be stored but not unmarshalled back into a
	// fosite.Session.
	Session []byte `bson:"session_data" json:"sessionData" xml:"sessionData"`
}

// NewRequest returns a new Mongo Store request object.
func NewRequest() Request {
	return Request{
		ID:             uuid.NewString(),
		RequestedAt:    time.Now(),
		Signature:      "",
		ClientID:       "",
		UserID:         "",
		RequestedScope: fosite.Arguments{},
		GrantedScope:   fosite.Arguments{},
		Form:           make(url.Values),
		Active:         true,
		Session:        nil,
	}
}

// ToRequest transforms a mongo request to a fosite.Request
func (r *Request) ToRequest(ctx context.Context, session fosite.Session, cm ClientStore) (*fosite.Request, error) {
	if session != nil {
		if err := json.Unmarshal(r.Session, session); err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		log.Printf("Got an empty session in toRequest")
	}

	client, err := cm.GetClient(ctx, r.ClientID)
	if err != nil {
		return nil, err
	}

	req := &fosite.Request{
		Client:            client,
		Session:           session,
		ID:                r.ID,
		RequestedAt:       r.RequestedAt,
		RequestedScope:    r.RequestedScope,
		GrantedScope:      r.GrantedScope,
		Form:              r.Form,
		RequestedAudience: r.RequestedAudience,
		GrantedAudience:   r.GrantedAudience,
	}
	return req, nil
}
