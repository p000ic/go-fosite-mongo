package mongo

import (
	// Standard Library Imports
	"testing"

	// External Imports
	"github.com/ory/fosite"

	// Internal Imports
	"github.com/p000ic/go-fosite-mongo"
)

func TestClientMongoManagerImplementsStorageConfigurer(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(storage.Configure); !ok {
		t.Error("ClientManager does not implement interface storage.Configure")
	}
}

func TestClientMongoManagerImplementsStorageAuthClientMigrator(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(storage.AuthClientMigrator); !ok {
		t.Error("ClientManager does not implement interface storage.AuthClientMigrator")
	}
}

func TestClientMongoManagerImplementsFositeClientManager(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(fosite.ClientManager); !ok {
		t.Error("ClientManager does not implement interface fosite.ClientManager")
	}
}

func TestClientMongoManagerImplementsFositeStorage(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(fosite.Storage); !ok {
		t.Error("ClientManager does not implement interface fosite.Storage")
	}
}

func TestClientMongoManagerImplementsStorageClientStorer(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(storage.ClientStore); !ok {
		t.Error("ClientManager does not implement interface storage.ClientStore")
	}
}

func TestClientMongoManagerImplementsStorageClientManager(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(storage.ClientManager); !ok {
		t.Error("ClientManager does not implement interface storage.ClientManager")
	}
}
