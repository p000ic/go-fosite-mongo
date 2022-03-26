package mongo

import (
	// Standard Library Imports
	"context"
	"encoding/json"
	"gopkg.in/square/go-jose.v2"
	"sync"
	"time"

	// External Imports
	"github.com/google/uuid"
	ot "github.com/opentracing/opentracing-go"
	"github.com/ory/fosite"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	// Internal Imports
	"github.com/p000ic/go-fosite-mongo"
)

type IssuerPublicKeys struct {
	Issuer    string
	KeysBySub map[string]SubjectPublicKeys
}

type SubjectPublicKeys struct {
	Subject string
	Keys    map[string]PublicKeyScopes
}

type PublicKeyScopes struct {
	Key    *jose.JSONWebKey
	Scopes []string
}

// RequestManager manages the main Mongo Session for a Request.
type RequestManager struct {
	// DB contains the Mongo connection that holds the base session that can be
	// copied and closed.
	DB *DB

	// Clients provides access to Client entities in order to create, read,
	// update and delete resources from the clients collection.
	// A client is required when cross referencing scope access rights.
	Clients storage.ClientStore

	// Users provides access to User entities in order to create, read, update
	// and delete resources from the user collection.
	// Users are required when the Password Credentials Grant, is implemented
	// in order to find and authenticate users.
	Users storage.UserStorer

	// Public keys to check signature in auth grant jwt assertion.
	IssuerPublicKeys map[string]IssuerPublicKeys

	clientsMutex          sync.RWMutex
	authorizeCodesMutex   sync.RWMutex
	idSessionsMutex       sync.RWMutex
	accessTokensMutex     sync.RWMutex
	refreshTokensMutex    sync.RWMutex
	pkcesMutex            sync.RWMutex
	usersMutex            sync.RWMutex
	issuerPublicKeysMutex sync.RWMutex
}

// Configure implements storage.Configure.
func (r *RequestManager) Configure(ctx context.Context) (err error) {
	// In terms of the underlying entity for session data, the model is the
	// same across the following entities. I have decided to logically break
	// them into separate collections rather than have a 'SessionType'.
	collections := []string{
		storage.EntityAccessTokens,
		storage.EntityAuthorizationCodes,
		storage.EntityOpenIDSessions,
		storage.EntityPKCESessions,
		storage.EntityRefreshTokens,
	}

	for _, entityName := range collections {
		// Build Indices
		indices := []mongo.IndexModel{
			NewUniqueIndex(IdxSessionID, "id"),
			NewIndex(IdxCompoundRequester, "clientId", "userId"),
		}

		// Compute Signature Index
		signatureIndex := NewUniqueIndex(IdxSignatureID, "signature")
		if entityName == storage.EntityAccessTokens {
			// Access Tokens generate a very large signature, which leads to
			// the index size blowing out. Instead, we can make use of Mongo's
			// hashed indices to massively reduce the size of the index.
			//
			// Note:
			// - Hashed Indices don't currently support a unique constraint.
			signatureIndex = NewIndex(IdxSignatureID+"Hashed", "#signature")
		}
		indices = append(indices, signatureIndex)

		log := logger.WithFields(logrus.Fields{
			"package":    "mongo",
			"collection": entityName,
			"method":     "Configure",
		})

		collection := r.DB.Collection(entityName)
		_, err = collection.Indexes().CreateMany(ctx, indices)
		if err != nil {
			log.WithError(err).Error(logError)
			return err
		}
	}

	return nil
}

// ConfigureExpiryWithTTL implements storage.Expire.
func (r *RequestManager) ConfigureExpiryWithTTL(ctx context.Context, ttl int) error {
	collections := []string{
		storage.EntityAccessTokens,
		storage.EntityAuthorizationCodes,
		storage.EntityOpenIDSessions,
		storage.EntityPKCESessions,
		storage.EntityRefreshTokens,
	}

	for _, entityName := range collections {
		log := logger.WithFields(logrus.Fields{
			"package":    "mongo",
			"collection": entityName,
			"method":     "ConfigureExpiryWithTTL",
		})

		index := NewExpiryIndex(IdxExpiry+"RequestedAt", "requestedAt", ttl)
		collection := r.DB.Collection(entityName)
		_, err := collection.Indexes().CreateOne(ctx, index)
		if err != nil {
			log.WithError(err).Error(logError)
			return err
		}
	}

	return nil
}

// getConcrete returns a Request resource.
func (r *RequestManager) getConcrete(ctx context.Context, entityName string, requestID string) (result storage.Request, err error) {
	log := logger.WithFields(logrus.Fields{
		"package":    "mongo",
		"collection": entityName,
		"method":     "getConcrete",
		"id":         requestID,
	})

	// Build Query
	query := bson.M{
		"id": requestID,
	}

	// Trace how long the Mongo operation takes to complete.
	span, _ := traceMongoCall(ctx, dbTrace{
		Manager: "RequestManager",
		Method:  "getConcrete",
		Query:   query,
	})
	defer span.Finish()

	var request storage.Request
	collection := r.DB.Collection(entityName)
	err = collection.FindOne(ctx, query).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.WithError(err).Debug(logNotFound)
			return result, fosite.ErrNotFound
		}

		// Log to StdOut
		log.WithError(err).Error(logError)
		// Log to OpenTracing
		otLogErr(span, err)
		return result, err
	}

	return request, nil
}

// List returns a list of Request resources that match the provided inputs.
func (r *RequestManager) List(ctx context.Context, entityName string, filter storage.ListRequestsRequest) (results []storage.Request, err error) {
	// Initialize contextual method logger
	log := logger.WithFields(logrus.Fields{
		"package":    "mongo",
		"collection": entityName,
		"method":     "List",
	})

	// Build Query
	query := bson.M{}
	if filter.ClientID != "" {
		query["clientId"] = filter.ClientID
	}
	if filter.UserID != "" {
		query["userId"] = filter.UserID
	}
	if len(filter.ScopesIntersection) > 0 {
		query["scopes"] = bson.M{"$all": filter.ScopesIntersection}
	}
	if len(filter.ScopesUnion) > 0 {
		query["scopes"] = bson.M{"$in": filter.ScopesUnion}
	}
	if len(filter.GrantedScopesIntersection) > 0 {
		query["scopes"] = bson.M{"$all": filter.GrantedScopesIntersection}
	}
	if len(filter.GrantedScopesUnion) > 0 {
		query["scopes"] = bson.M{"$in": filter.GrantedScopesUnion}
	}

	// Trace how long the Mongo operation takes to complete.
	span, _ := traceMongoCall(ctx, dbTrace{
		Manager: "RequestManager",
		Method:  "List",
		Query:   query,
	})
	defer span.Finish()

	collection := r.DB.Collection(entityName)
	cursor, err := collection.Find(ctx, query)
	if err != nil {
		// Log to StdOut
		log.WithError(err).Error(logError)
		// Log to OpenTracing
		otLogErr(span, err)
		return results, err
	}

	var requests []storage.Request
	err = cursor.All(ctx, &requests)
	if err != nil {
		// Log to StdOut
		log.WithError(err).Error(logError)
		// Log to OpenTracing
		otLogErr(span, err)
		return results, err
	}

	return requests, nil
}

// Create creates the new Request resource and returns the newly created Request
// resource.
func (r *RequestManager) Create(ctx context.Context, entityName string, request storage.Request) (result storage.Request, err error) {
	// Initialize contextual method logger
	log := logger.WithFields(logrus.Fields{
		"package":    "mongo",
		"collection": entityName,
		"method":     "Create",
	})

	// Enable developers to provide their own IDs
	if request.ID == "" {
		request.ID = uuid.NewString()
	}
	if request.CreateTime == 0 {
		request.CreateTime = time.Now().Unix()
	}
	if request.RequestedAt.IsZero() {
		request.RequestedAt = time.Now()
	}

	// Trace how long the Mongo operation takes to complete.
	span, _ := traceMongoCall(ctx, dbTrace{
		Manager: "RequestManager",
		Method:  "Create",
	})
	defer span.Finish()

	// Create resource
	collection := r.DB.Collection(entityName)
	_, err = collection.InsertOne(ctx, request)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// Log to StdOut
			log.WithError(err).Debug(logConflict)
			// Log to OpenTracing
			otLogErr(span, err)
			return result, storage.ErrResourceExists
		}

		// Log to StdOut
		log.WithError(err).Error(logError)
		// Log to OpenTracing
		otLogQuery(span, request)
		otLogErr(span, err)
		return result, err
	}

	return request, nil
}

// Get returns the specified Request resource.
func (r *RequestManager) Get(ctx context.Context, entityName string, requestID string) (result storage.Request, err error) {
	return r.getConcrete(ctx, entityName, requestID)
}

// GetBySignature returns a Request resource, if the presented signature returns
// a match.
func (r *RequestManager) GetBySignature(ctx context.Context, entityName string, signature string) (result storage.Request, err error) {
	log := logger.WithFields(logrus.Fields{
		"package":    "mongo",
		"collection": entityName,
		"method":     "GetBySignature",
	})

	// Build Query
	query := bson.M{
		"signature": signature,
	}

	// Trace how long the Mongo operation takes to complete.
	span, _ := traceMongoCall(ctx, dbTrace{
		Manager: "RequestManager",
		Method:  "GetBySignature",
		Query:   query,
	})
	defer span.Finish()

	var request storage.Request
	collection := r.DB.Collection(entityName)
	err = collection.FindOne(ctx, query).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.WithError(err).Debug(logNotFound)
			return result, fosite.ErrNotFound
		}

		// Log to StdOut
		log.WithError(err).Error(logError)
		// Log to OpenTracing
		otLogErr(span, err)
		return result, err
	}

	return request, nil
}

// Update updates the Request resource and attributes and returns the updated
// Request resource.
func (r *RequestManager) Update(ctx context.Context, entityName string, requestID string, updatedRequest storage.Request) (result storage.Request, err error) {
	// Initialize contextual method logger
	log := logger.WithFields(logrus.Fields{
		"package":    "mongo",
		"collection": entityName,
		"method":     "Update",
		"id":         requestID,
	})

	// Deny updating the entity Id
	updatedRequest.ID = requestID
	// Update modified time
	updatedRequest.UpdateTime = time.Now().Unix()

	// Build Query
	selector := bson.M{
		"id": requestID,
	}

	// Trace how long the Mongo operation takes to complete.
	span, _ := traceMongoCall(ctx, dbTrace{
		Manager:  "RequestManager",
		Method:   "Update",
		Selector: selector,
	})
	defer span.Finish()

	collection := r.DB.Collection(entityName)
	res, err := collection.ReplaceOne(ctx, selector, updatedRequest)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// Log to StdOut
			log.WithError(err).Debug(logConflict)
			// Log to OpenTracing
			otLogErr(span, err)
			return result, storage.ErrResourceExists
		}

		// Log to StdOut
		log.WithError(err).Error(logError)
		// Log to OpenTracing
		otLogQuery(span, updatedRequest)
		otLogErr(span, err)
		return result, err
	}

	if res.MatchedCount == 0 {
		// Log to StdOut
		log.WithError(err).Debug(logNotFound)
		// Log to OpenTracing
		otLogErr(span, err)
		return result, fosite.ErrNotFound
	}

	return updatedRequest, nil
}

// Delete deletes the specified Request resource.
func (r *RequestManager) Delete(ctx context.Context, entityName string, requestID string) (err error) {
	// Initialize contextual method logger
	log := logger.WithFields(logrus.Fields{
		"package":    "mongo",
		"collection": entityName,
		"method":     "Delete",
		"id":         requestID,
	})

	// Build Query
	query := bson.M{
		"id": requestID,
	}

	// Trace how long the Mongo operation takes to complete.
	span, _ := traceMongoCall(ctx, dbTrace{
		Manager: "RequestManager",
		Method:  "Delete",
		Query:   query,
	})
	defer span.Finish()

	collection := r.DB.Collection(entityName)
	res, err := collection.DeleteOne(ctx, query)
	if err != nil {
		// Log to StdOut
		log.WithError(err).Error(logError)
		// Log to OpenTracing
		otLogErr(span, err)
		return err
	}

	if res.DeletedCount == 0 {
		// Log to StdOut
		log.WithError(err).Debug(logNotFound)
		// Log to OpenTracing
		otLogErr(span, err)
		return fosite.ErrNotFound
	}

	return nil
}

// DeleteBySignature deletes the specified request resource, if the presented
// signature returns a match.
func (r *RequestManager) DeleteBySignature(ctx context.Context, entityName string, signature string) (err error) {
	// Initialize contextual method logger
	log := logger.WithFields(logrus.Fields{
		"package":    "mongo",
		"collection": entityName,
		"method":     "DeleteBySignature",
	})

	// Build Query
	query := bson.M{
		"signature": signature,
	}

	// Trace how long the Mongo operation takes to complete.
	span, _ := traceMongoCall(ctx, dbTrace{
		Manager: "RequestManager",
		Method:  "DeleteBySignature",
		Query:   query,
	})
	defer span.Finish()

	collection := r.DB.Collection(entityName)
	res, err := collection.DeleteOne(ctx, query)
	if err != nil {
		// Log to StdOut
		log.WithError(err).Error(logError)
		// Log to OpenTracing
		otLogErr(span, err)
		return err
	}

	if res.DeletedCount == 0 {
		// Log to StdOut
		log.WithError(err).Debug(logNotFound)
		// Log to OpenTracing
		otLogErr(span, err)
		return fosite.ErrNotFound
	}

	return nil
}

// RevokeRefreshToken deletes the refresh token session.
func (r *RequestManager) RevokeRefreshToken(ctx context.Context, requestID string) (err error) {
	return r.revokeToken(ctx, storage.EntityRefreshTokens, requestID)
}

// RevokeAccessToken deletes the access token session.
func (r *RequestManager) RevokeAccessToken(ctx context.Context, requestID string) (err error) {
	return r.revokeToken(ctx, storage.EntityAccessTokens, requestID)
}

func (r *RequestManager) RevokeRefreshTokenMaybeGracePeriod(ctx context.Context, requestID string, signature string) error {
	// no configuration option is available; grace period is not available with memory store
	return r.RevokeRefreshToken(ctx, requestID)
}

func (r *RequestManager) GetPublicKey(ctx context.Context, issuer string, subject string, keyId string) (*jose.JSONWebKey, error) {
	r.issuerPublicKeysMutex.RLock()
	defer r.issuerPublicKeysMutex.RUnlock()

	if issuerKeys, ok := r.IssuerPublicKeys[issuer]; ok {
		if subKeys, ok := issuerKeys.KeysBySub[subject]; ok {
			if keyScopes, ok := subKeys.Keys[keyId]; ok {
				return keyScopes.Key, nil
			}
		}
	}

	return nil, fosite.ErrNotFound
}

func (r *RequestManager) GetPublicKeys(ctx context.Context, issuer string, subject string) (*jose.JSONWebKeySet, error) {
	r.issuerPublicKeysMutex.RLock()
	defer r.issuerPublicKeysMutex.RUnlock()
	if issuerKeys, ok := r.IssuerPublicKeys[issuer]; ok {
		if subKeys, ok := issuerKeys.KeysBySub[subject]; ok {
			if len(subKeys.Keys) == 0 {
				return nil, fosite.ErrNotFound
			}

			keys := make([]jose.JSONWebKey, 0, len(subKeys.Keys))
			for _, keyScopes := range subKeys.Keys {
				keys = append(keys, *keyScopes.Key)
			}

			return &jose.JSONWebKeySet{Keys: keys}, nil
		}
	}
	return nil, fosite.ErrNotFound
}

func (r *RequestManager) GetPublicKeyScopes(ctx context.Context, issuer string, subject string, keyId string) ([]string, error) {
	r.issuerPublicKeysMutex.RLock()
	defer r.issuerPublicKeysMutex.RUnlock()
	if issuerKeys, ok := r.IssuerPublicKeys[issuer]; ok {
		if subKeys, ok := issuerKeys.KeysBySub[subject]; ok {
			if keyScopes, ok := subKeys.Keys[keyId]; ok {
				return keyScopes.Scopes, nil
			}
		}
	}
	return nil, fosite.ErrNotFound
}

// revokeToken deletes a token based on the provided request id.
func (r *RequestManager) revokeToken(ctx context.Context, entityName string, requestID string) (err error) {
	// Initialize contextual method logger
	log := logger.WithFields(logrus.Fields{
		"package":    "mongo",
		"collection": entityName,
		"method":     "revokeToken",
		"id":         requestID,
	})

	// Trace how long the Mongo operation takes to complete.
	span, ctx := traceMongoCall(ctx, dbTrace{
		Manager: "RequestManager",
		Method:  "revokeToken",
		Query:   requestID,
		CustomTags: []ot.Tag{{
			Key:   "collection",
			Value: entityName,
		}},
	})
	defer span.Finish()

	err = r.Delete(ctx, entityName, requestID)
	if err != nil && err != fosite.ErrNotFound {
		// Note: If the token is not found, we can declare it revoked.

		// Log to StdOut
		log.WithError(err).Error(logError)
		// Log to OpenTracing
		otLogErr(span, err)
		return err
	}

	return nil
}

// toMongo transforms a fosite.Request to a storage.Request
// Signature is a hash that relates to the underlying request method and may not
// be a strict 'signature', for example, authorization code grant passes in an
// authorization code.
func toMongo(signature string, r fosite.Requester) storage.Request {
	session, _ := json.Marshal(r.GetSession())
	return storage.Request{
		ID:                r.GetID(),
		RequestedAt:       r.GetRequestedAt(),
		Signature:         signature,
		ClientID:          r.GetClient().GetID(),
		UserID:            r.GetSession().GetSubject(),
		RequestedScope:    r.GetRequestedScopes(),
		GrantedScope:      r.GetGrantedScopes(),
		RequestedAudience: r.GetRequestedAudience(),
		GrantedAudience:   r.GetGrantedAudience(),
		Form:              r.GetRequestForm(),
		Active:            true,
		Session:           session,
	}
}
