package storage_test

import (
	// Standard Library Imports
	"testing"

	// External Imports
	"github.com/ory/fosite"

	// Internal Imports
	"github.com/p000ic/go-fosite-mongo"
)

func TestClient_ImplementsFositeClientInterface(t *testing.T) {
	c := &storage.Client{}

	var i interface{} = c
	if _, ok := i.(fosite.Client); !ok {
		t.Error("storage.Client does not implement interface fosite.Client")
	}
}
