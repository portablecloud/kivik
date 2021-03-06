package db

import (
	"sort"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/test/kt"
	"github.com/pkg/errors"
)

func init() {
	kt.Register("AllDocs", allDocs)
}

func allDocs(ctx *kt.Context) {
	ctx.RunAdmin(func(ctx *kt.Context) {
		testAllDocs(ctx, ctx.Admin)
	})
	ctx.RunNoAuth(func(ctx *kt.Context) {
		testAllDocs(ctx, ctx.NoAuth)
	})
	ctx.RunRW(func(ctx *kt.Context) {
		testAllDocsRW(ctx)
	})
}

func testAllDocsRW(ctx *kt.Context) {
	if ctx.Admin == nil {
		// Can't do anything here without admin access
		return
	}
	dbName, expected, err := setUpAllDocsTest(ctx)
	if err != nil {
		ctx.Errorf("Failed to set up temp db: %s", err)
	}
	defer ctx.Admin.DestroyDB(dbName)
	ctx.Run("group", func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			doTest(ctx, ctx.Admin, dbName, 0, expected)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			doTest(ctx, ctx.NoAuth, dbName, 0, expected)
		})
	})
}

func setUpAllDocsTest(ctx *kt.Context) (dbName string, docIDs []string, err error) {
	dbName = ctx.TestDBName()
	if err = ctx.Admin.CreateDB(dbName); err != nil {
		return dbName, nil, errors.Wrap(err, "failed to create db")
	}
	db, err := ctx.Admin.DB(dbName)
	if err != nil {
		return dbName, nil, errors.Wrap(err, "failed to connect to db")
	}
	docIDs = make([]string, 10)
	for i := range docIDs {
		id := ctx.TestDBName()
		doc := struct {
			ID string `json:"id"`
		}{
			ID: id,
		}
		if _, err := db.Put(doc.ID, doc); err != nil {
			return dbName, nil, errors.Wrap(err, "failed to create doc")
		}
		docIDs[i] = id
	}
	sort.Strings(docIDs)
	return dbName, docIDs, nil

}

func testAllDocs(ctx *kt.Context, client *kivik.Client) {
	if !ctx.IsSet("databases") {
		ctx.Errorf("databases not set; Did you configure this test?")
		return
	}
	for _, dbName := range ctx.StringSlice("databases") {
		func(dbName string) {
			ctx.Run(dbName, func(ctx *kt.Context) {
				doTest(ctx, client, dbName, int64(ctx.Int("offset")), ctx.StringSlice("expected"))
			})
		}(dbName)
	}
}

func doTest(ctx *kt.Context, client *kivik.Client, dbName string, expOffset int64, expected []string) {
	ctx.Run("WithDocs", func(ctx *kt.Context) {
		doTestWithDocs(ctx, client, dbName, expOffset, expected)
	})
	ctx.Run("WithoutDocs", func(ctx *kt.Context) {
		doTestWithoutDocs(ctx, client, dbName, expOffset, expected)
	})
}

func doTestWithoutDocs(ctx *kt.Context, client *kivik.Client, dbName string, expOffset int64, expected []string) {
	ctx.Parallel()
	db, err := client.DB(dbName)
	// Errors may be deferred here, so only return if we actually get
	// an error.
	if err != nil && !ctx.IsExpectedSuccess(err) {
		return
	}

	rows, err := db.AllDocs(nil)
	if !ctx.IsExpectedSuccess(err) {
		return
	}
	docIDs := make([]string, 0, len(expected))
	for rows.Next() {
		docIDs = append(docIDs, rows.ID())
	}
	if rows.Err() != nil {
		ctx.Fatalf("Failed to fetch row: %s", rows.Err())
	}
	if d := diff.TextSlices(expected, docIDs); d != "" {
		ctx.Errorf("Unexpected document IDs returned:\n%s\n", d)
	}
	if expOffset != rows.Offset() {
		ctx.Errorf("offset: Expected %d, got %d", expOffset, rows.Offset())
	}
	if int64(len(expected)) != rows.TotalRows() {
		ctx.Errorf("total rows: Expected %d, got %d", len(expected), rows.TotalRows())
	}
}

func doTestWithDocs(ctx *kt.Context, client *kivik.Client, dbName string, expOffset int64, expected []string) {
	ctx.Parallel()
	db, err := client.DB(dbName)
	// Errors may be deferred here, so only return if we actually get
	// an error.
	if err != nil && !ctx.IsExpectedSuccess(err) {
		return
	}
	opts := map[string]interface{}{
		"include_docs": true,
		"update_seq":   true,
	}

	rows, err := db.AllDocs(opts)
	if !ctx.IsExpectedSuccess(err) {
		return
	}
	docIDs := make([]string, 0, len(expected))
	for rows.Next() {
		var doc struct {
			ID  string `json:"_id"`
			Rev string `json:"_rev"`
		}
		if err := rows.ScanDoc(&doc); err != nil {
			ctx.Errorf("Failed to scan doc: %s", err)
		}
		var value struct {
			Rev string `json:"rev"`
		}
		if err := rows.ScanValue(&value); err != nil {
			ctx.Errorf("Failed to scan value: %s", err)
		}
		if value.Rev != doc.Rev {
			ctx.Errorf("doc._rev = %s, but value.rev = %s", doc.Rev, value.Rev)
		}
		if doc.ID != rows.ID() {
			ctx.Errorf("doc._id = %s, but rows.ID = %s", doc.ID, rows.ID())
		}
		docIDs = append(docIDs, rows.ID())
	}
	if rows.Err() != nil {
		ctx.Fatalf("Failed to fetch row: %s", rows.Err())
	}
	if d := diff.TextSlices(expected, docIDs); d != "" {
		ctx.Errorf("Unexpected document IDs returned:\n%s\n", d)
	}
	if expOffset != rows.Offset() {
		ctx.Errorf("offset: Expected %d, got %d", expOffset, rows.Offset())
	}
	ctx.Run("UpdateSeq", func(ctx *kt.Context) {
		if rows.UpdateSeq() == "" {
			ctx.Errorf("Expected updated sequence")
		}
	})
	if int64(len(expected)) != rows.TotalRows() {
		ctx.Errorf("total rows: Expected %d, got %d", len(expected), rows.TotalRows())
	}
}
