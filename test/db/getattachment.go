package db

import (
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/test/kt"
)

func init() {
	kt.Register("GetAttachment", getAttachment)
}

func getAttachment(ctx *kt.Context) {
	ctx.RunRW(func(ctx *kt.Context) {
		dbname := ctx.TestDBName()
		defer ctx.Admin.DestroyDB(dbname)
		if err := ctx.Admin.CreateDB(dbname); err != nil {
			ctx.Fatalf("Failed to create db: %s", err)
		}
		adb, err := ctx.Admin.DB(dbname)
		if err != nil {
			ctx.Fatalf("Failed to open db: %s", err)
		}

		doc := map[string]interface{}{
			"_id": "foo",
			"_attachments": map[string]interface{}{
				"foo.txt": map[string]interface{}{
					"content_type": "text/plain",
					"data":         "VGhpcyBpcyBhIGJhc2U2NCBlbmNvZGVkIHRleHQ=",
				},
			},
		}
		if _, err = adb.Put("foo", doc); err != nil {
			ctx.Fatalf("Failed to create doc: %s", err)
		}

		ddoc := map[string]interface{}{
			"_id": "_design/foo",
			"_attachments": map[string]interface{}{
				"foo.txt": map[string]interface{}{
					"content_type": "text/plain",
					"data":         "VGhpcyBpcyBhIGJhc2U2NCBlbmNvZGVkIHRleHQ=",
				},
			},
		}
		if _, err = adb.Put("_design/foo", ddoc); err != nil {
			ctx.Fatalf("Failed to create design doc: %s", err)
		}

		ctx.Run("group", func(ctx *kt.Context) {
			ctx.RunAdmin(func(ctx *kt.Context) {
				ctx.Parallel()
				testGetAttachments(ctx, ctx.Admin, dbname, "foo", "foo.txt")
				testGetAttachments(ctx, ctx.Admin, dbname, "foo", "NotFound")
				testGetAttachments(ctx, ctx.Admin, dbname, "_design/foo", "foo.txt")
			})
			ctx.RunNoAuth(func(ctx *kt.Context) {
				ctx.Parallel()
				testGetAttachments(ctx, ctx.NoAuth, dbname, "foo", "foo.txt")
				testGetAttachments(ctx, ctx.NoAuth, dbname, "foo", "NotFound")
				testGetAttachments(ctx, ctx.NoAuth, dbname, "_design/foo", "foo.txt")
			})
		})
	})
}

func testGetAttachments(ctx *kt.Context, client *kivik.Client, dbname, docID, filename string) {
	ctx.Run(docID+"/"+filename, func(ctx *kt.Context) {
		ctx.Parallel()
		db, err := client.DB(dbname)
		if err != nil {
			ctx.Fatalf("Failed to connect to db")
		}
		att, err := db.GetAttachment(docID, "", filename)
		if !ctx.IsExpectedSuccess(err) {
			return
		}
		if client.Driver() != "pouch" && client.Driver() != "memdown" {
			if att.ContentType != "text/plain" {
				ctx.Errorf("Content-Type: Expected %s, Actual %s", "text/plain", att.ContentType)
			}
		}
	})
}
