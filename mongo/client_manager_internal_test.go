package mongo

import (
	// Standard Library Imports
	"testing"

	// External Imports
	"github.com/ory/fosite"

	// Internal Imports
	"github.com/p000ic/go-fosite-mongo"
)

func TestClientMongoManager_ImplementsStorageConfigurer(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(storage.Configure); !ok {
		t.Error("ClientManager does not implement interface storage.Configure")
	}
}

func TestClientMongoManager_ImplementsStorageAuthClientMigrator(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(storage.AuthClientMigrator); !ok {
		t.Error("ClientManager does not implement interface storage.AuthClientMigrator")
	}
}

func TestClientMongoManager_ImplementsFositeClientManager(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(fosite.ClientManager); !ok {
		t.Error("ClientManager does not implement interface fosite.ClientManager")
	}
}

func TestClientMongoManager_ImplementsFositeStorage(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(fosite.Storage); !ok {
		t.Error("ClientManager does not implement interface fosite.Storage")
	}
}

func TestClientMongoManager_ImplementsStorageClientStorer(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(storage.ClientStore); !ok {
		t.Error("ClientManager does not implement interface storage.ClientStore")
	}
}

func TestClientMongoManager_ImplementsStorageClientManager(t *testing.T) {
	c := &ClientManager{}

	var i interface{} = c
	if _, ok := i.(storage.ClientManager); !ok {
		t.Error("ClientManager does not implement interface storage.ClientManager")
	}
}
