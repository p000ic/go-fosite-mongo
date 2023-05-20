package mongo

import (
	// Standard Library Imports
	"context"
	"time"

	// External Imports
	"github.com/google/uuid"
	"github.com/ory/fosite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	// Internal Imports
	"github.com/p000ic/go-fosite-mongo"
)

// UserManager provides a mongo backed implementation for user resources.
//
// Implements:
// - storage.Configure
// - storage.AuthUserMigrator
// - storage.UserStorer
// - storage.UserManager
type UserManager struct {
	DB     *DB
	Hasher fosite.Hasher
}

// Configure implements storage.Configure.
func (u *UserManager) Configure(ctx context.Context) (err error) {
	indices := []mongo.IndexModel{
		NewUniqueIndex(IdxUserID, "id"),
		NewUniqueIndex(IdxUsername, "username"),
	}

	collection := u.DB.Collection(storage.EntityUsers)
	_, err = collection.Indexes().CreateMany(ctx, indices)
	if err != nil {
		return err
	}

	return nil
}

// getConcrete returns an OAuth 2.0 User resource.
func (u *UserManager) getConcrete(ctx context.Context, userID string) (result storage.User, err error) {
	// Build Query
	query := bson.M{
		"id": userID,
	}
	var user storage.User
	collection := u.DB.Collection(storage.EntityUsers)
	err = collection.FindOne(ctx, query).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return result, fosite.ErrNotFound
		}
		return result, err
	}

	return user, nil
}

// List returns a list of User resources that match the provided inputs.
func (u *UserManager) List(ctx context.Context, filter storage.ListUsersRequest) (results []storage.User, err error) {
	// Build Query
	query := bson.M{}
	if filter.AllowedTenantAccess != "" {
		query["allowed_tenant_access"] = filter.AllowedTenantAccess
	}
	if filter.AllowedPersonAccess != "" {
		query["allowed_person_access"] = filter.AllowedPersonAccess
	}
	if filter.PersonID != "" {
		query["person_id"] = filter.PersonID
	}
	if filter.Username != "" {
		query["username"] = filter.Username
	}
	if len(filter.ScopesIntersection) > 0 {
		query["scopes"] = bson.M{"$all": filter.ScopesIntersection}
	}
	if len(filter.ScopesUnion) > 0 {
		query["scopes"] = bson.M{"$in": filter.ScopesUnion}
	}
	if filter.FirstName != "" {
		query["first_name"] = filter.FirstName
	}
	if filter.LastName != "" {
		query["last_name"] = filter.LastName
	}
	if filter.Disabled {
		query["disabled"] = filter.Disabled
	}

	collection := u.DB.Collection(storage.EntityUsers)
	cursor, err := collection.Find(ctx, query)
	if err != nil {
		return results, err
	}

	var users []storage.User
	err = cursor.All(ctx, &users)
	if err != nil {
		return results, err
	}

	return users, nil
}

// Create creates a new User resource and returns the newly created User
// resource.
func (u *UserManager) Create(ctx context.Context, user storage.User) (result storage.User, err error) {
	// Enable developers to provide their own IDs
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	if user.CreateTime == 0 {
		user.CreateTime = time.Now().Unix()
	}

	// Hash incoming secret
	hash, err := u.Hasher.Hash(ctx, []byte(user.Password))
	if err != nil {
		return result, err
	}
	user.Password = string(hash)

	// Create resource
	collection := u.DB.Collection(storage.EntityUsers)
	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return result, storage.ErrResourceExists
		}
		return result, err
	}

	return user, nil
}

// Get returns the specified User resource.
func (u *UserManager) Get(ctx context.Context, userID string) (result storage.User, err error) {
	return u.getConcrete(ctx, userID)
}

// GetByUsername returns a user resource if found by username.
func (u *UserManager) GetByUsername(ctx context.Context, username string) (result storage.User, err error) {
	// Build Query
	query := bson.M{
		"username": username,
	}
	var user storage.User
	collection := u.DB.Collection(storage.EntityUsers)
	err = collection.FindOne(ctx, query).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return result, fosite.ErrNotFound
		}

		return result, err
	}

	return user, nil
}

// Update updates the User resource and attributes and returns the updated
// User resource.
func (u *UserManager) Update(ctx context.Context, userID string, updatedUser storage.User) (result storage.User, err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var closeSession func()
		ctx, closeSession, err = newSession(ctx, u.DB)
		if err != nil {
			return result, err
		}
		defer closeSession()
	}

	currentResource, err := u.getConcrete(ctx, userID)
	if err != nil {
		if err == fosite.ErrNotFound {
			return result, err
		}

		return result, err
	}

	// Deny updating the entity Id
	updatedUser.ID = userID
	// Update modified time
	updatedUser.UpdateTime = time.Now().Unix()

	if currentResource.Password == updatedUser.Password || updatedUser.Password == "" {
		// If the password/hash is blank or hash matches, set using old hash.
		updatedUser.Password = currentResource.Password
	} else {
		newHash, err := u.Hasher.Hash(ctx, []byte(updatedUser.Password))
		if err != nil {
			return result, err
		}
		updatedUser.Password = string(newHash)
	}
	// Build Query
	selector := bson.M{
		"id": userID,
	}

	collection := u.DB.Collection(storage.EntityUsers)
	res, err := collection.ReplaceOne(ctx, selector, updatedUser)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return result, storage.ErrResourceExists
		}
		return result, err
	}

	if res.MatchedCount == 0 {
		return result, fosite.ErrNotFound
	}

	return updatedUser, nil
}

// Migrate is provided solely for the case where you want to migrate users and
// upgrade their password using the AuthUserMigrator interface.
// This performs an upsert, either creating or overwriting the record with the
// newly provided full record. Use with caution, be secure, don't be dumb.
func (u *UserManager) Migrate(ctx context.Context, migratedUser storage.User) (result storage.User, err error) {
	// Generate a unique ID if not supplied
	if migratedUser.ID == "" {
		migratedUser.ID = uuid.NewString()
	}
	// Update create time
	if migratedUser.CreateTime == 0 {
		migratedUser.CreateTime = time.Now().Unix()
	}
	// Update modified time
	migratedUser.UpdateTime = time.Now().Unix()

	// Build Query
	selector := bson.M{
		"id": migratedUser.ID,
	}

	collection := u.DB.Collection(storage.EntityUsers)
	opts := options.Replace().SetUpsert(true)
	_, err = collection.ReplaceOne(ctx, selector, migratedUser, opts)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return result, storage.ErrResourceExists
		}
		return result, err
	}

	return migratedUser, nil
}

// Delete deletes the specified User resource.
func (u *UserManager) Delete(ctx context.Context, userID string) (err error) {
	// Build Query
	query := bson.M{
		"id": userID,
	}

	collection := u.DB.Collection(storage.EntityUsers)
	res, err := collection.DeleteOne(ctx, query)
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return fosite.ErrNotFound
	}
	return nil
}

// Authenticate confirms whether the specified password matches the stored
// hashed password within the User resource.
// The User resource returned is matched by username.
func (u *UserManager) Authenticate(ctx context.Context, username string, password string) (result storage.User, err error) {
	return u.AuthenticateByUsername(ctx, username, password)
}

// AuthenticateByID confirms whether the specified password matches the stored
// hashed password within the User resource.
// The User resource returned is matched by User ID.
func (u *UserManager) AuthenticateByID(ctx context.Context, userID string, password string) (result storage.User, err error) {
	user, err := u.getConcrete(ctx, userID)
	if err != nil {
		return result, err
	}

	if user.Disabled {
		return result, fosite.ErrAccessDenied
	}

	err = u.Hasher.Compare(ctx, []byte(user.Password), []byte(password))
	if err != nil {
		return result, err
	}

	return user, nil
}

// AuthenticateByUsername confirms whether the specified password matches the
// stored hashed password within the User resource.
// The User resource returned is matched by username.
func (u *UserManager) AuthenticateByUsername(ctx context.Context, username string, password string) (result storage.User, err error) {
	user, err := u.GetByUsername(ctx, username)
	if err != nil {
		return result, err
	}

	if user.Disabled {
		return result, fosite.ErrAccessDenied
	}

	err = u.Hasher.Compare(ctx, []byte(user.Password), []byte(password))
	if err != nil {
		return result, err
	}

	return user, nil
}

// AuthenticateMigration enables developers to supply your own
// authentication function, which in turn, if true, will migrate the secret
// to the Hasher implemented within fosite.
func (u *UserManager) AuthenticateMigration(ctx context.Context, currentAuth storage.AuthUserFunc, userID string, password string) (result storage.User, err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var closeSession func()
		ctx, closeSession, err = newSession(ctx, u.DB)
		if err != nil {
			return result, err
		}
		defer closeSession()
	}
	// Authenticate with old Hasher
	user, authenticated := currentAuth(ctx)

	// Check for user not found
	if user.IsEmpty() && !authenticated {
		return result, fosite.ErrNotFound
	}

	if user.Disabled {
		return result, fosite.ErrAccessDenied
	}

	if !authenticated {
		// If user isn't authenticated, try authenticating with new Hasher.
		err := u.Hasher.Compare(ctx, user.GetHashedSecret(), []byte(password))
		if err != nil {
			return result, err
		}
		return user, nil
	}

	// If the user is found and authenticated, create a new hash using the new
	// Hasher, update the database record and return the record with no error.
	newHash, err := u.Hasher.Hash(ctx, []byte(password))
	if err != nil {
		return result, err
	}

	// Save the new hash
	user.UpdateTime = time.Now().Unix()
	user.Password = string(newHash)

	return u.Update(ctx, userID, user)
}

// GrantScopes grants the provided scopes to the specified User resource.
func (u *UserManager) GrantScopes(ctx context.Context, userID string, scopes []string) (result storage.User, err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var closeSession func()
		ctx, closeSession, err = newSession(ctx, u.DB)
		if err != nil {
			return result, err
		}
		defer closeSession()
	}

	user, err := u.getConcrete(ctx, userID)
	if err != nil {
		if err == fosite.ErrNotFound {
			return result, err
		}

		return result, err
	}

	// Enable access to the provided scopes...
	user.UpdateTime = time.Now().Unix()
	user.EnableScopeAccess(scopes...)

	return u.Update(ctx, user.ID, user)
}

// RemoveScopes revokes the provided scopes from the specified User Resource.
func (u *UserManager) RemoveScopes(ctx context.Context, userID string, scopes []string) (result storage.User, err error) {
	// Copy a new DB session if none specified
	_, ok := ContextToSession(ctx)
	if !ok {
		var closeSession func()
		ctx, closeSession, err = newSession(ctx, u.DB)
		if err != nil {
			return result, err
		}
		defer closeSession()
	}
	user, err := u.getConcrete(ctx, userID)
	if err != nil {
		if err == fosite.ErrNotFound {
			return result, err
		}
		return result, err
	}

	// Disable access to the provided scopes...
	user.UpdateTime = time.Now().Unix()
	user.DisableScopeAccess(scopes...)
	return u.Update(ctx, user.ID, user)
}
