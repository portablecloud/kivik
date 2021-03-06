package usersdb

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/authdb"
	_ "github.com/flimzy/kivik/driver/couchdb"
	"github.com/flimzy/kivik/errors"
	"github.com/flimzy/kivik/test/kt"
)

type tuser struct {
	ID       string   `json:"_id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Roles    []string `json:"roles"`
	Password string   `json:"password"`
}

var testUser = &tuser{
	ID:       "org.couchdb.user:testUsersdb",
	Name:     "testUsersdb",
	Type:     "user",
	Roles:    []string{"coolguy"},
	Password: "abc123",
}

func TestCouchAuth(t *testing.T) {
	client := kt.GetClient(t)
	db, err := client.DB("_users")
	if err != nil {
		t.Fatalf("Failed to connect to db: %s", err)
	}
	rev, err := db.Put(testUser.ID, testUser)
	if err != nil {
		t.Fatalf("Failed to create user: %s", err)
	}
	defer db.Delete(testUser.ID, rev)
	auth := New(db)
	uCtx, err := auth.Validate(kt.CTX, "testUsersdb", "abc123")
	if err != nil {
		t.Errorf("Validation failure for good password: %s", err)
	}
	if uCtx == nil {
		t.Errorf("User should have been validated")
	}
	uCtx, err = auth.Validate(kt.CTX, "testUsersdb", "foobar")
	if errors.StatusCode(err) != kivik.StatusUnauthorized {
		t.Errorf("Expected Unauthorized password, got %s", err)
	}
	if uCtx != nil {
		t.Errorf("User should not have been validated with wrong password")
	}
	uCtx, err = auth.Validate(kt.CTX, "nobody", "foo")
	if errors.StatusCode(err) != kivik.StatusUnauthorized {
		t.Errorf("Expected Unauthorized for bad username, got %s", err)
	}
	if uCtx != nil {
		t.Errorf("User should not have been validated with wrong username")
	}

	uCtx, err = auth.UserCtx(kt.CTX, "testUsersdb")
	if err != nil {
		t.Errorf("Failed to get roles for valid user: %s", err)
	}
	uCtx.Salt = "" // It's random, so remove it
	if !reflect.DeepEqual(uCtx, &authdb.UserContext{Name: "testUsersdb", Roles: []string{"coolguy"}}) {
		t.Errorf("Got unexpected output: %v", uCtx)
	}
	_, err = auth.UserCtx(kt.CTX, "nobody")
	if errors.StatusCode(err) != kivik.StatusNotFound {
		var msg string
		if err != nil {
			msg = fmt.Sprintf(" Got: %s", err)
		}
		t.Errorf("Expected Not Found fetching roles for bad username.%s", msg)
	}
}
