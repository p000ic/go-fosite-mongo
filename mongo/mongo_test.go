package mongo

import (
	// Standard Library Imports
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMain(m *testing.M) {
	// If needed, enable logging when debugging for tests
	// mongo.SetLogger(logrus.New())
	// mongo.SetDebug(true)

	exitCode := m.Run()
	os.Exit(exitCode)
}

func AssertError(t *testing.T, got interface{}, want interface{}, msg string) {
	t.Errorf(fmt.Sprintf("Error: %s\n	 got: %#+v\n	want: %#+v", msg, got, want))
}

func AssertFatal(t *testing.T, got interface{}, want interface{}, msg string) {
	t.Fatalf(fmt.Sprintf("Fatal: %s\n	 got: %#+v\n	want: %#+v", msg, got, want))
}

func setup(t *testing.T) (*Store, context.Context, func()) {
	cfg := &Config{}
	cfg.Hostnames = []string{"localhost"}
	cfg.Port = 27017
	cfg.DatabaseName = "oauth2"
	cfg.Username = "test"
	cfg.Password = "test"
	cfg.AuthDB = "admin"

	store, err := New(cfg, nil)
	if err != nil {
		AssertFatal(t, err, nil, "mongo connection error")
	}

	// Build a context with a mongo session ready to use for testing
	ctx := context.Background()
	var sess func()
	ctx, sess, err = store.NewSession(ctx)
	if err != nil {
		AssertFatal(t, err, nil, "error getting mongo session")
	}

	teardown := func() {
		// Drop the database.
		err = store.DB.Drop(ctx)
		if err != nil {
			t.Errorf("error dropping database on cleanup: %s", err)
			return
		}

		// Close the inner (test) session if it exists.
		sess()

		// Close the database connection.
		store.Close()
	}

	return store, ctx, teardown
}

// TestNewStore tests the NewDefaultStore function.
func TestNewStore(t *testing.T) {
	cfg := &Config{}
	cfg.Hostnames = []string{"localhost"}
	cfg.Port = 27017
	cfg.DatabaseName = "oauth2"
	cfg.Username = "test"
	cfg.Password = "test"
	cfg.AuthDB = "admin"

	convey.Convey("Default store", t, func() {
		store, err := New(cfg, nil)
		convey.So(err, convey.ShouldBeNil)
		convey.So(store.DB, convey.ShouldNotBeNil)

		convey.Convey("Store should be functional", func() {
			collNames, err := store.DB.ListCollectionNames(context.Background(), bson.D{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(collNames, convey.ShouldNotBeEmpty)
			if err != nil {
				t.Errorf("Error listing collections: %s", err)
				return
			}
			t.Logf("Collections: %v", collNames)
		})

		convey.Convey("Close the store", func() {
			store.Close()
		})
	})
}
