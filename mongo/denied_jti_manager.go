package mongo

import (
	// Standard Library Imports
	"context"
	"sync"
	"time"

	// External Imports
	"github.com/ory/fosite"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	// Internal Imports
	"github.com/p000ic/go-fosite-mongo"
)

// DeniedJtiManager provides a mongo backed implementation for denying JSON Web
// Tokens (JWTs) by ID.
type DeniedJtiManager struct {
	DB *DB

	BlacklistedJTIs        map[string]time.Time
	AccessTokenRequestIDs  map[string]string
	RefreshTokenRequestIDs map[string]string

	blacklistedJTIsMutex        sync.RWMutex
	accessTokenRequestIDsMutex  sync.RWMutex
	refreshTokenRequestIDsMutex sync.RWMutex
}

// Configure implements storage.Configure.
func (d *DeniedJtiManager) Configure(ctx context.Context) (err error) {
	// indices := []mongo.IndexModel{
	// 	NewUniqueIndex(IdxSignatureID, "signature"),
	// 	NewIndex(IdxExpires, "exp"),
	// }
	// collection := d.DB.Collection(storage.EntityJtiDenylist)
	// _, err = collection.Indexes().CreateMany(ctx, indices)
	// if err != nil {
	// 	return err
	// }
	return nil
}

// getConcrete returns a denied jti resource.
func (d *DeniedJtiManager) getConcrete(ctx context.Context, signature string) (result storage.DeniedJTI, err error) {
	// Build Query
	query := bson.M{
		"signature": signature,
	}
	var user storage.DeniedJTI
	collection := d.DB.Collection(storage.EntityJtiDenylist)
	err = collection.FindOne(ctx, query).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return result, fosite.ErrNotFound
		}
		return result, err
	}

	return user, nil
}

// Create creates a new User resource and returns the newly created User
// resource.
func (d *DeniedJtiManager) Create(ctx context.Context, deniedJTI storage.DeniedJTI) (result storage.DeniedJTI, err error) {
	// Create resource
	collection := d.DB.Collection(storage.EntityJtiDenylist)
	_, err = collection.InsertOne(ctx, deniedJTI)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return result, storage.ErrResourceExists
		}
		return result, err
	}

	return deniedJTI, nil
}

// Get returns the specified User resource.
func (d *DeniedJtiManager) Get(ctx context.Context, signature string) (result storage.DeniedJTI, err error) {
	return d.getConcrete(ctx, signature)
}

func (d *DeniedJtiManager) Delete(ctx context.Context, jti string) (err error) {
	// Build Query
	query := bson.M{
		"signature": storage.SignatureFromJTI(jti),
	}

	collection := d.DB.Collection(storage.EntityJtiDenylist)
	res, err := collection.DeleteOne(ctx, query)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return fosite.ErrNotFound
	}

	return nil
}

// DeleteBefore DeleteExpired removes all JTIs before the given time. Returns not found if
// no tokens were found before the given time.
func (d *DeniedJtiManager) DeleteBefore(ctx context.Context, expBefore int64) (err error) {
	// Build Query
	query := bson.M{
		"exp": bson.M{
			"$lt": time.Now().Unix(),
		},
	}

	collection := d.DB.Collection(storage.EntityJtiDenylist)
	res, err := collection.DeleteMany(ctx, query)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return fosite.ErrNotFound
	}

	return nil
}

// func (d *DeniedJtiManager) IsJWTUsed(ctx context.Context, jti string) (bool, error) {
//	err := d.ClientAssertionJWTValid(ctx, jti)
//	if err != nil {
//		return true, nil
//	}
//
//	return false, nil
// }
//
// func (d *DeniedJtiManager) MarkJWTUsedForTime(ctx context.Context, jti string, exp time.Time) error {
//	return d.SetClientAssertionJWT(ctx, jti, exp)
// }

func (d *DeniedJtiManager) ClientAssertionJWTValid(_ context.Context, jti string) error {
	d.blacklistedJTIsMutex.RLock()
	defer d.blacklistedJTIsMutex.RUnlock()

	if exp, exists := d.BlacklistedJTIs[jti]; exists && exp.After(time.Now()) {
		return fosite.ErrJTIKnown
	}

	return nil
}

func (d *DeniedJtiManager) SetClientAssertionJWT(_ context.Context, jti string, exp time.Time) error {
	d.blacklistedJTIsMutex.Lock()
	defer d.blacklistedJTIsMutex.Unlock()

	// delete expired jtis
	for j, e := range d.BlacklistedJTIs {
		if e.Before(time.Now()) {
			delete(d.BlacklistedJTIs, j)
		}
	}

	if _, exists := d.BlacklistedJTIs[jti]; exists {
		return fosite.ErrJTIKnown
	}

	d.BlacklistedJTIs[jti] = exp
	return nil
}
