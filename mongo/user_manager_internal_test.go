package mongo

import (
	"testing"

	"github.com/p000ic/go-fosite-mongo"
)

func TestUserMongoManagerImplementsStorageConfigurer(t *testing.T) {
	u := &UserManager{}

	var i interface{} = u
	if _, ok := i.(storage.Configure); !ok {
		t.Error("UserManager does not implement interface storage.Configure")
	}
}

func TestUserMongoManagerImplementsStorageAuthUserMigrator(t *testing.T) {
	u := &UserManager{}

	var i interface{} = u
	if _, ok := i.(storage.AuthUserMigrator); !ok {
		t.Error("UserManager does not implement interface storage.AuthUserMigrator")
	}
}

func TestUserMongoManagerImplementsStorageUserStorer(t *testing.T) {
	u := &UserManager{}

	var i interface{} = u
	if _, ok := i.(storage.UserStorer); !ok {
		t.Error("UserManager does not implement interface storage.UserStorer")
	}
}

func TestUserMongoManagerImplementsStorageUserManager(t *testing.T) {
	u := &UserManager{}

	var i interface{} = u
	if _, ok := i.(storage.UserManager); !ok {
		t.Error("UserManager does not implement interface storage.UserManager")
	}
}
