// +build js

package test

import (
	"testing"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver/pouchdb"
	"github.com/flimzy/kivik/test/kt"
	"github.com/gopherjs/gopherjs/js"
)

func init() {
	kivik.Register("memdown", &pouchdb.Driver{
		Defaults: map[string]interface{}{
			"db": js.Global.Call("require", "memdown"),
		},
	})
}

func TestPouchLocal(t *testing.T) {
	client, err := kivik.New("memdown", "")
	if err != nil {
		t.Errorf("Failed to connect to PouchDB/memdown driver: %s", err)
		return
	}
	clients := &kt.Context{
		RW:    true,
		Admin: client,
	}
	runTests(clients, SuitePouchLocal, t)
}

func TestPouchRemote(t *testing.T) {
	doTest(SuitePouchRemote, "KIVIK_TEST_DSN_COUCH16", t)
}
