package mongo

import (
	// Standard Library Imports
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	// External Imports
	"github.com/ory/fosite"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	// Local Imports
	"github.com/p000ic/go-fosite-mongo"
)

func init() {}

var (
	defaultHost         = ""
	defaultDatabaseName = ""
	// defaultPort         = 0
	// defaultUsername     = ""
	// defaultPassword     = ""
	// defaultAuthDB       = ""
)

// Store provides a MongoDB storage driver compatible with fosite's required
// storage interfaces.
type Store struct {
	// Internals
	DB *DB

	// timeout provides a way to configure maximum time before killing an
	// in-flight request.
	timeout time.Duration

	// Public API
	Hasher fosite.Hasher
	storage.Store
}

// DB wraps the mongo database connection and the features that are enabled.
type DB struct {
	*mongo.Database
}

// NewSession creates and returns a new mongo session.
// A deferrable session closer is returned in an attempt to enforce proper
// session handling/closing of sessions to avoid session and memory leaks.
//
// NewSession boilerplate becomes:
// ```
// ctx := context.Background()
//
//	if store.DB.HasSessions {
//	    var sess func()
//	    ctx, sess, err = store.NewSession(nil)
//	    if err != nil {
//	        panic(err)
//	    }
//	    defer sess()
//	}
//
// ```
func (s *Store) NewSession(ctx context.Context) (context.Context, func(), error) {
	return newSession(ctx, s.DB)
}

// newSession creates a new mongo session.
func newSession(ctx context.Context, db *DB) (context.Context, func(), error) {
	session, err := db.Client().StartSession()
	if err != nil {
		return ctx, nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx = SessionToContext(ctx, session)

	return ctx, closeSession(ctx, session), nil
}

// closeSession encapsulates the logic required to close a mongo session.
func closeSession(ctx context.Context, session *mongo.Session) func() {
	return func() {
		session.EndSession(ctx)
	}
}

// Close terminates the mongo connection.
func (s *Store) Close() {
	err := s.DB.Client().Disconnect(context.Background())
	if err != nil {
		return
	}
}

// Config defines the configuration parameters which are used by GetMongoSession.
type Config struct {
	Hostnames        []string    `default:"localhost" envconfig:"CONNECTIONS_MONGO_HOSTNAMES"`
	Port             uint16      `default:"27017"     envconfig:"CONNECTIONS_MONGO_PORT"`
	SSL              bool        `default:"false"     envconfig:"CONNECTIONS_MONGO_SSL"`
	AuthDB           string      `default:"admin"     envconfig:"CONNECTIONS_MONGO_AUTH_DB"`
	Username         string      `default:""          envconfig:"CONNECTIONS_MONGO_USERNAME"`
	Password         string      `default:""          envconfig:"CONNECTIONS_MONGO_PASSWORD"`
	DatabaseName     string      `default:""          envconfig:"CONNECTIONS_MONGO_NAME"`
	Replicaset       string      `default:""          envconfig:"CONNECTIONS_MONGO_REPLICASET"`
	Timeout          uint        `default:"10"        envconfig:"CONNECTIONS_MONGO_TIMEOUT"`
	PoolMinSize      uint64      `default:"0"         envconfig:"CONNECTIONS_MONGO_POOL_MIN_SIZE"`
	PoolMaxSize      uint64      `default:"100"       envconfig:"CONNECTIONS_MONGO_POOL_MAX_SIZE"`
	Compressors      []string    `default:""          envconfig:"CONNECTIONS_MONGO_COMPRESSORS"`
	TokenTTL         uint32      `default:"0"         envconfig:"CONNECTIONS_MONGO_TOKEN_TTL"`
	CollectionPrefix string      `default:""          envconfig:"CONNECTIONS_MONGO_COLLECTION_PREFIX"`
	TLSConfig        *tls.Config `ignored:"true"`
}

// // DefaultConfig returns a configuration for a locally hosted, unauthenticated mongo
// func DefaultConfig() *Config {
// 	cfg := &Config{
// 		Hostnames:    []string{defaultHost},
// 		Port:         uint16(defaultPort),
// 		DatabaseName: defaultDatabaseName,
// 		AuthDB:       defaultAuthDB,
// 		Username:     defaultUsername,
// 		Password:     defaultPassword,
// 	}
// 	return cfg
// }

// ConnectionInfo configures options for establishing a session with a MongoDB cluster.
func ConnectionInfo(cfg *Config) *options.ClientOptions {
	if len(cfg.Hostnames) == 0 {
		cfg.Hostnames = []string{defaultHost}
	}

	if cfg.DatabaseName == "" {
		cfg.DatabaseName = defaultDatabaseName
	}

	clientOpts := options.Client()
	if len(cfg.Hostnames) == 1 && strings.HasPrefix(cfg.Hostnames[0], "mongodb+srv://") {
		// MongoDB SRV records can only be configured with ApplyURI,
		// but we can continue to mung with client options after it's set.
		clientOpts.ApplyURI(cfg.Hostnames[0])
	} else {
		for i := range cfg.Hostnames {
			if cfg.Port != 0 && !strings.Contains(cfg.Hostnames[i], ":") {
				cfg.Hostnames[i] = fmt.Sprintf("%s:%d", cfg.Hostnames[i], cfg.Port)
			}
		}
		clientOpts.SetHosts(cfg.Hostnames)
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 1
	}

	clientOpts.
		SetConnectTimeout(time.Second * time.Duration(cfg.Timeout)).
		SetReadPreference(readpref.SecondaryPreferred()).
		SetMinPoolSize(cfg.PoolMinSize).
		SetMaxPoolSize(cfg.PoolMaxSize).
		SetCompressors(cfg.Compressors).
		SetAppName(cfg.DatabaseName)

	if cfg.Username != "" && cfg.Password != "" {
		auth := options.Credential{
			AuthMechanism: "SCRAM-SHA-1",
			AuthSource:    cfg.AuthDB,
			Username:      cfg.Username,
			Password:      cfg.Password,
		}
		clientOpts.SetAuth(auth)
	}

	if cfg.SSL {
		tlsConfig := cfg.TLSConfig
		if tlsConfig == nil {
			// Inject a default TLS config if the SSL switch is toggled, but a
			// TLS config has not been provided programmatically.
			tlsConfig = &tls.Config{
				InsecureSkipVerify: false,
				MinVersion:         tls.VersionTLS12,
			}
		}

		clientOpts.SetTLSConfig(tlsConfig)
	}

	return clientOpts
}

// Connect returns a connection to a mongo database.
func Connect(cfg *Config) (*mongo.Database, error) {
	ctx := context.Background()
	opts := ConnectionInfo(cfg)

	var startedCommands sync.Map
	cmdMonitor := &event.CommandMonitor{
		Started: func(_ context.Context, evt *event.CommandStartedEvent) {
			startedCommands.Store(evt.RequestID, evt.Command)
		},
		Succeeded: func(_ context.Context, evt *event.CommandSucceededEvent) {
			startedCommands.Delete(evt.RequestID)
		},
		Failed: func(_ context.Context, evt *event.CommandFailedEvent) {
			if cmd, ok := startedCommands.Load(evt.RequestID); ok {
				log.Printf("cmd: %v failure-resp: %v", cmd, evt.Failure)
				startedCommands.Delete(evt.RequestID)
			}
		},
	}
	opts.SetMonitor(cmdMonitor)
	client, err := mongo.Connect(opts)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	// check connection works as mongo-go lazily connects.
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	// log.Printf("mongo-db-connection-successful")
	return client.Database(cfg.DatabaseName), nil
}

// New allows for custom mongo configuration and custom hashers.
func New(cfg *Config, hash fosite.Hasher) (*Store, error) {
	database, err := Connect(cfg)
	if err != nil {
		return nil, err
	}

	// Wrap database with mongo feature detection.
	mongoDB := &DB{
		Database: database,
	}

	if hash == nil {
		// Initialize default fosite Hasher.
		hash = &fosite.BCrypt{Config: &fosite.Config{HashCost: 8}}
	}

	// Build up the mongo endpoints
	mongoDeniedJTIs := &DeniedJtiManager{
		DB: mongoDB,
	}
	mongoClients := &ClientManager{
		DB:     mongoDB,
		Hasher: hash,

		DeniedJTIs: mongoDeniedJTIs,
	}
	mongoUsers := &UserManager{
		DB:     mongoDB,
		Hasher: hash,
	}
	mongoRequests := &RequestManager{
		DB: mongoDB,

		Clients: mongoClients,
		Users:   mongoUsers,
	}

	// attempt to perform index updates in a session.
	ctx, closeSess, err := newSession(context.Background(), mongoDB)
	if err != nil {
		return nil, err
	}
	defer closeSess()

	// Configure DB collections, indices, TTLs e.t.c.
	if err = configureDatabases(ctx, mongoClients, mongoDeniedJTIs, mongoUsers, mongoRequests); err != nil {
		return nil, err
	}
	if cfg.TokenTTL > 0 {
		if err = configureExpiry(ctx, int(cfg.TokenTTL), mongoRequests); err != nil {
			return nil, err
		}
	}

	store := &Store{
		DB:      mongoDB,
		timeout: time.Second * time.Duration(cfg.Timeout),
		Hasher:  hash,
		Store: storage.Store{
			ClientManager:    mongoClients,
			DeniedJTIManager: mongoDeniedJTIs,
			RequestManager:   mongoRequests,
			UserManager:      mongoUsers,
		},
	}
	return store, nil
}

// configureDatabases calls the configuration handler for the provided
// configures.
func configureDatabases(ctx context.Context, cfgs ...storage.Configure) error {
	for _, cfg := range cfgs {
		if err := cfg.Configure(ctx); err != nil {
			return err
		}
	}

	return nil
}

// configureExpiry calls the configuration handler for the provided expires.
// ttl should be a positive integer.
func configureExpiry(ctx context.Context, ttl int, expires ...storage.Expire) error {
	for _, expire := range expires {
		if err := expire.ConfigureExpiryWithTTL(ctx, ttl); err != nil {
			return err
		}
	}

	return nil
}

// NewIndex generates a new index model, ready to be saved in mongo.
//
// Note:
//   - This function assumes you are entering valid index keys and relies on
//     mongo rejecting index operations if a bad index is created.
func NewIndex(name string, keys ...string) (model mongo.IndexModel) {
	idxModel := mongo.IndexModel{
		Keys: generateIndexKeys(keys...),
		Options: options.Index().
			SetName(name).
			SetUnique(false),
	}
	return idxModel
}

// NewUniqueIndex generates a new unique index model, ready to be saved in
// mongo.
func NewUniqueIndex(name string, keys ...string) mongo.IndexModel {
	idxModel := mongo.IndexModel{
		Keys: generateIndexKeys(keys...),
		Options: options.Index().
			SetName(name).
			SetUnique(true),
	}
	return idxModel
}

// NewExpiryIndex generates a new index with a time to live value before the
// record expires in mongodb.
func NewExpiryIndex(name string, key string, expireAfter int) (model mongo.IndexModel) {
	idxModel := mongo.IndexModel{
		Keys: bson.D{{Key: key, Value: int32(1)}},
		Options: options.Index().
			SetName(name).
			SetUnique(false).
			SetExpireAfterSeconds(int32(expireAfter)),
	}
	return idxModel
}

// generateIndexKeys given a number of stringy keys will return a bson
// document containing keys in the structure required by mongo for defining
// index and sort order.
func generateIndexKeys(keys ...string) (indexKeys bson.D) {
	var indexKey bson.E
	for _, key := range keys {
		switch {
		case strings.HasPrefix(key, "-"):
			// Reverse Index
			indexKey.Key = strings.TrimLeft(key, "-")
			indexKey.Value = int32(-1)

		case strings.HasPrefix(key, "#"):
			// Hashed Index
			indexKey.Key = strings.TrimLeft(key, "#")
			indexKey.Value = "hashed"

		default:
			// Forward Index
			indexKey.Key = key
			indexKey.Value = int32(1)
		}

		indexKeys = append(indexKeys, indexKey)
	}

	return
}
