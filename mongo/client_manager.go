package mongo

import (
	// Standard Library imports
	"context"
	"errors"
	"time"

	// External Imports
	"github.com/google/uuid"
	"github.com/ory/fosite"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	// Internal Imports
	"github.com/p000ic/go-fosite-mongo"
)

// ClientManager provides a fosite storage implementation for Clients.
//
// Implements:
// - fosite.Storage
// - fosite.ClientManager
// - storage.AuthClientMigrator
// - storage.ClientManager
// - storage.ClientStore
type ClientManager struct {
	DB     *DB
	Hasher fosite.Hasher

	DeniedJTIs storage.DeniedJTIStore
}

// Configure sets up the Mongo collection for OAuth 2.0 client resources.
func (c *ClientManager) Configure(ctx context.Context) (err error) {
	// Build Index
	// indices := []mongo.IndexModel{
	// 	NewUniqueIndex(IdxClientID, "id"),
	// }
	// collection := c.DB.Collection(storage.EntityClients)
	// _, err = collection.Indexes().CreateMany(ctx, indices)
	// if err != nil {
	// 	return err
	// }
	return nil
}

// getConcrete returns an OAuth 2.0 Client resource.
func (c *ClientManager) getConcrete(ctx context.Context, clientID string) (result storage.Client, err error) {
	// Build Query
	query := bson.M{
		"id": clientID,
	}
	var storageClient storage.Client
	collection := c.DB.Collection(storage.EntityClients)
	err = collection.FindOne(ctx, query).Decode(&storageClient)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return result, fosite.ErrNotFound
		}
		return result, err
	}

	return storageClient, nil
}

// List filters resources to return a list of OAuth 2.0 client resources.
func (c *ClientManager) List(ctx context.Context, filter storage.ListClientsRequest) (results []storage.Client, err error) {
	// Build Query
	query := bson.M{}
	if filter.AllowedTenantAccess != "" {
		query["allowed_tenant_access"] = filter.AllowedTenantAccess
	}
	if filter.AllowedRegion != "" {
		query["allowed_regions"] = filter.AllowedRegion
	}
	if filter.RedirectURI != "" {
		query["redirect_uris"] = filter.RedirectURI
	}
	if filter.GrantType != "" {
		query["grant_types"] = filter.GrantType
	}
	if filter.ResponseType != "" {
		query["response_types"] = filter.ResponseType
	}
	if len(filter.ScopesIntersection) > 0 {
		query["scopes"] = bson.M{"$all": filter.ScopesIntersection}
	}
	if len(filter.ScopesUnion) > 0 {
		query["scopes"] = bson.M{"$in": filter.ScopesUnion}
	}
	if filter.Contact != "" {
		query["contacts"] = filter.Contact
	}
	if filter.Public {
		query["public"] = filter.Public
	}
	if filter.Disabled {
		query["disabled"] = filter.Disabled
	}
	if filter.Published {
		query["published"] = filter.Published
	}
	collection := c.DB.Collection(storage.EntityClients)
	cursor, err := collection.Find(ctx, query)
	if err != nil {
		return results, err
	}

	var clients []storage.Client
	err = cursor.All(ctx, &clients)
	if err != nil {
		return results, err
	}

	return clients, nil
}

// Create stores a new OAuth2.0 Client resource.
func (c *ClientManager) Create(ctx context.Context, client storage.Client) (result storage.Client, err error) {
	// Enable developers to provide their own IDs
	if client.ID == "" {
		client.ID = uuid.NewString()
	}
	if client.CreateTime == 0 {
		client.CreateTime = time.Now().Unix()
	}

	// Hash incoming secret
	hash, err := c.Hasher.Hash(ctx, []byte(client.Secret))
	if err != nil {
		return result, err
	}
	client.Secret = string(hash)

	// Create resource
	collection := c.DB.Collection(storage.EntityClients)
	_, err = collection.InsertOne(ctx, client)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return result, storage.ErrResourceExists
		}
		return result, err
	}

	return client, nil
}

// Get finds and returns an OAuth 2.0 client resource.
func (c *ClientManager) Get(ctx context.Context, clientID string) (result storage.Client, err error) {
	return c.getConcrete(ctx, clientID)
}

// GetClient finds and returns an OAuth 2.0 client resource.
//
// GetClient implements:
// - fosite.Storage
// - fosite.ClientManager
func (c *ClientManager) GetClient(ctx context.Context, clientID string) (fosite.Client, error) {
	client, err := c.getConcrete(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

// ClientAssertionJWTValid returns an error if the JTI is known or the DB check
// failed and nil if the JTI is not known.
func (c *ClientManager) ClientAssertionJWTValid(ctx context.Context, jti string) error {
	deniedJti, err := c.DeniedJTIs.Get(ctx, jti)
	if err != nil {
		switch err {
		case fosite.ErrNotFound:
			// the jti is not known => valid
			return nil

		default:
			// Unknown error...
			return err
		}
	}

	if time.Unix(deniedJti.Expiry, 0).After(time.Now()) {
		// the jti is not expired yet => invalid
		return fosite.ErrJTIKnown
	}

	return nil
}

// SetClientAssertionJWT marks a JTI as known for the given expiry time.
// Before inserting the new JTI, it will clean up any existing JTIs that have
// expired as those tokens can not be replayed due to the expiry.
func (c *ClientManager) SetClientAssertionJWT(ctx context.Context, jti string, exp time.Time) (err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var closeSession func()
		ctx, closeSession, err = newSession(ctx, c.DB)
		if err != nil {
			return err
		}
		defer closeSession()
	}

	// delete expired JTIs
	err = c.DeniedJTIs.DeleteBefore(ctx, time.Now().Unix())
	if err != nil {
		switch err {
		case fosite.ErrNotFound:
			return
		default:
			return err
		}
	}

	_, err = c.DeniedJTIs.Create(ctx, storage.NewDeniedJTI(jti, exp))
	if err != nil {
		switch err {
		case storage.ErrResourceExists:
			// found a DeniedJTIs
			return fosite.ErrJTIKnown
		default:
			return err
		}
	}

	return nil
}

// Update updates an OAuth 2.0 client resource.
func (c *ClientManager) Update(ctx context.Context, clientID string, updatedClient storage.Client) (result storage.Client, err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var closeSession func()
		ctx, closeSession, err = newSession(ctx, c.DB)
		if err != nil {
			return result, err
		}
		defer closeSession()
	}

	currentResource, err := c.getConcrete(ctx, clientID)
	if err != nil {
		if errors.Is(err, fosite.ErrNotFound) {
			return result, err
		}
		return result, err
	}

	// Deny updating the entity Id
	updatedClient.ID = clientID
	// Update modified time
	updatedClient.UpdateTime = time.Now().Unix()

	if currentResource.Secret == updatedClient.Secret || updatedClient.Secret == "" {
		// If the password/hash is blank or hash matches, set using old hash.
		updatedClient.Secret = currentResource.Secret
	} else {
		// newHash, err := c.Hasher.Hash(ctx, []byte(updatedClient.Secret))
		// if err != nil {
		// 	return result, err
		// }
		// updatedClient.Secret = string(newHash)
	}

	// Build Query
	selector := bson.M{
		"id": clientID,
	}

	collection := c.DB.Collection(storage.EntityClients)
	res, err := collection.ReplaceOne(ctx, selector, updatedClient)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return result, storage.ErrResourceExists
		}
		return result, err
	}

	if res.MatchedCount == 0 {
		return result, fosite.ErrNotFound
	}

	return updatedClient, nil
}

// Migrate is provided solely for the case where you want to migrate clients and
// upgrade their password using the AuthClientMigrator interface.
// This performs an upsert, either creating or overwriting the record with the
// newly provided full record. Use with caution, be secure, don't be dumb.
func (c *ClientManager) Migrate(ctx context.Context, migratedClient storage.Client) (result storage.Client, err error) {
	// Generate a unique ID if not supplied
	if migratedClient.ID == "" {
		migratedClient.ID = uuid.NewString()
	}
	// Update create time
	if migratedClient.CreateTime == 0 {
		migratedClient.CreateTime = time.Now().Unix()
	} else {
		// Update modified time
		migratedClient.UpdateTime = time.Now().Unix()
	}

	// Build Query
	selector := bson.M{
		"id": migratedClient.ID,
	}

	collection := c.DB.Collection(storage.EntityClients)
	opts := options.Replace().SetUpsert(true)
	res, err := collection.ReplaceOne(ctx, selector, migratedClient, opts)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return result, storage.ErrResourceExists
		}
		return result, err
	}

	if res.MatchedCount == 0 {
		return result, fosite.ErrNotFound
	}

	return migratedClient, nil
}

// Delete removes an OAuth 2.0 Client resource.
func (c *ClientManager) Delete(ctx context.Context, clientID string) (err error) {
	// Build Query
	query := bson.M{
		"id": clientID,
	}
	collection := c.DB.Collection(storage.EntityClients)
	res, err := collection.DeleteOne(ctx, query)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return fosite.ErrNotFound
	}

	return nil
}

// Authenticate verifies the identity of a client resource.
func (c *ClientManager) Authenticate(ctx context.Context, clientID string, secret string) (result storage.Client, err error) {
	client, err := c.getConcrete(ctx, clientID)
	if err != nil {
		if errors.Is(err, fosite.ErrNotFound) {
			return result, err
		}
		return result, err
	}

	if client.Public {
		// The client doesn't have a secret, therefore is authenticated
		// implicitly.
		return client, nil
	}

	if client.Disabled {
		return result, fosite.ErrAccessDenied
	}

	err = c.Hasher.Compare(ctx, client.GetHashedSecret(), []byte(secret))
	if err != nil {
		return result, err
	}

	return client, nil
}

// AuthenticateMigration is provided to authenticate clients that have been
// migrated from a system that may use a different underlying hashing
// mechanism.
// It authenticates a Client first by using the provided AuthClientFunc which,
// if fails, will otherwise try to authenticate using the configured
// fosite.hasher.
func (c *ClientManager) AuthenticateMigration(ctx context.Context, currentAuth storage.AuthClientFunc, clientID string, secret string) (result storage.Client, err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var closeSession func()
		ctx, closeSession, err = newSession(ctx, c.DB)
		if err != nil {
			return result, err
		}
		defer closeSession()
	}

	// Authenticate with old Hasher
	client, authenticated := currentAuth(ctx)

	// Check for client not found
	if client.IsEmpty() && !authenticated {
		return result, fosite.ErrNotFound
	}

	if client.Public {
		// The client doesn't have a secret, therefore is authenticated
		// implicitly.
		return client, nil
	}

	if client.Disabled {
		return result, fosite.ErrAccessDenied
	}

	if !authenticated {
		// If client isn't authenticated, try authenticating with new Hasher.
		err := c.Hasher.Compare(ctx, client.GetHashedSecret(), []byte(secret))
		if err != nil {
			return result, err
		}
		return client, nil
	}

	// If the client is found and authenticated, create a new hash using the new
	// Hasher, update the database record and return the record with no error.
	newHash, err := c.Hasher.Hash(ctx, []byte(secret))
	if err != nil {
		return result, err
	}

	// Save the new hash
	client.UpdateTime = time.Now().Unix()
	client.Secret = string(newHash)

	return c.Update(ctx, clientID, client)
}

// GrantScopes grants the provided scopes to the specified Client resource.
func (c *ClientManager) GrantScopes(ctx context.Context, clientID string, scopes []string) (result storage.Client, err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var closeSession func()
		ctx, closeSession, err = newSession(ctx, c.DB)
		if err != nil {
			return result, err
		}
		defer closeSession()
	}
	client, err := c.getConcrete(ctx, clientID)
	if err != nil {
		if errors.Is(err, fosite.ErrNotFound) {
			return result, err
		}
		return result, err
	}

	client.UpdateTime = time.Now().Unix()
	client.EnableScopeAccess(scopes...)

	return c.Update(ctx, client.ID, client)
}

// RemoveScopes revokes the provided scopes from the specified Client resource.
func (c *ClientManager) RemoveScopes(ctx context.Context, clientID string, scopes []string) (result storage.Client, err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var closeSession func()
		ctx, closeSession, err = newSession(ctx, c.DB)
		if err != nil {
			return result, err
		}
		defer closeSession()
	}

	client, err := c.getConcrete(ctx, clientID)
	if err != nil {
		if errors.Is(err, fosite.ErrNotFound) {
			return result, err
		}
		return result, err
	}

	client.UpdateTime = time.Now().Unix()
	client.DisableScopeAccess(scopes...)

	return c.Update(ctx, client.ID, client)
}

func (c *ClientManager) IsJWTUsed(ctx context.Context, jti string) (bool, error) {
	err := c.ClientAssertionJWTValid(ctx, jti)
	if err != nil {
		return true, nil
	}

	return false, nil
}

func (c *ClientManager) MarkJWTUsedForTime(ctx context.Context, jti string, exp time.Time) error {
	return c.SetClientAssertionJWT(ctx, jti, exp)
}
