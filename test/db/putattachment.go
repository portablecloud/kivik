package db

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/test/kt"
)

func init() {
	kt.Register("PutAttachment", putAttachment)
}

func putAttachment(ctx *kt.Context) {
	ctx.RunRW(func(ctx *kt.Context) {
		dbname := ctx.TestDBName()
		defer ctx.Admin.DestroyDB(dbname)
		if err := ctx.Admin.CreateDB(dbname); err != nil {
			ctx.Fatalf("Failed to create db: %s", err)
		}
		ctx.Run("group", func(ctx *kt.Context) {
			ctx.RunAdmin(func(ctx *kt.Context) {
				ctx.Parallel()
				testPutAttachment(ctx, ctx.Admin, dbname)
			})
			ctx.RunNoAuth(func(ctx *kt.Context) {
				ctx.Parallel()
				testPutAttachment(ctx, ctx.NoAuth, dbname)
			})
		})
	})
}

func testPutAttachment(ctx *kt.Context, client *kivik.Client, dbname string) {
	db, err := client.DB(dbname)
	if err != nil {
		ctx.Fatalf("Failed to open db: %s", err)
	}
	adb, err2 := ctx.Admin.DB(dbname)
	if err2 != nil {
		ctx.Fatalf("Failed to open admin db: %s", err2)
	}
	ctx.Run("Update", func(ctx *kt.Context) {
		ctx.Parallel()
		docID, rev, err2 := adb.CreateDoc(map[string]string{"name": "Robert"})
		if err2 != nil {
			ctx.Fatalf("Failed to create doc: %s", err2)
		}
		att := kivik.NewAttachment("test.txt", "text/plain", stringReadCloser("test content"))
		_, err = db.PutAttachment(docID, rev, att)
		ctx.CheckError(err)
	})
	ctx.Run("Create", func(ctx *kt.Context) {
		ctx.Parallel()
		docID := ctx.TestDBName()
		att := kivik.NewAttachment("test.txt", "text/plain", stringReadCloser("test content"))
		_, err = db.PutAttachment(docID, "", att)
		ctx.CheckError(err)
	})
	ctx.Run("Conflict", func(ctx *kt.Context) {
		ctx.Parallel()
		docID, _, err2 := adb.CreateDoc(map[string]string{"name": "Robert"})
		if err2 != nil {
			ctx.Fatalf("Failed to create doc: %s", err2)
		}
		att := kivik.NewAttachment("test.txt", "text/plain", stringReadCloser("test content"))
		_, err = db.PutAttachment(docID, "5-20bd3c7d7d6b81390c6679d8bae8795b", att)
		ctx.CheckError(err)
	})
	ctx.Run("UpdateDesignDoc", func(ctx *kt.Context) {
		ctx.Parallel()
		docID := "_design/" + ctx.TestDBName()
		doc := map[string]string{
			"_id": docID,
		}
		var rev string
		rev, err = adb.Put(docID, doc)
		if err != nil {
			ctx.Fatalf("Failed to create design doc: %s", err)
		}
		att := kivik.NewAttachment("test.txt", "text/plain", stringReadCloser("test content"))
		_, err = db.PutAttachment(docID, rev, att)
		ctx.CheckError(err)
	})
	ctx.Run("CreateDesignDoc", func(ctx *kt.Context) {
		ctx.Parallel()
		docID := "_design/" + ctx.TestDBName()
		att := kivik.NewAttachment("test.txt", "text/plain", stringReadCloser("test content"))
		_, err = db.PutAttachment(docID, "", att)
		ctx.CheckError(err)
	})
}

func stringReadCloser(str string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(str))
}
