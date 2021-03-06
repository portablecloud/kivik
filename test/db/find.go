package db

import (
	"sort"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/test/kt"
	"github.com/pkg/errors"
)

func init() {
	kt.Register("Find", find)
}

func find(ctx *kt.Context) {
	ctx.RunAdmin(func(ctx *kt.Context) {
		testFind(ctx, ctx.Admin)
	})
	ctx.RunNoAuth(func(ctx *kt.Context) {
		testFind(ctx, ctx.NoAuth)
	})
	ctx.RunRW(func(ctx *kt.Context) {
		testFindRW(ctx)
	})
}

func testFindRW(ctx *kt.Context) {
	if ctx.Admin == nil {
		// Can't do anything here without admin access
		return
	}
	dbName, expected, err := setUpFindTest(ctx)
	if err != nil {
		ctx.Errorf("Failed to set up temp db: %s", err)
	}
	defer ctx.Admin.DestroyDB(dbName)
	ctx.Run("group", func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			doFindTest(ctx, ctx.Admin, dbName, 0, expected)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			doFindTest(ctx, ctx.NoAuth, dbName, 0, expected)
		})
	})
}

func setUpFindTest(ctx *kt.Context) (dbName string, docIDs []string, err error) {
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

func testFind(ctx *kt.Context, client *kivik.Client) {
	if !ctx.IsSet("databases") {
		ctx.Errorf("databases not set; Did you configure this test?")
		return
	}
	for _, dbName := range ctx.StringSlice("databases") {
		func(dbName string) {
			ctx.Run(dbName, func(ctx *kt.Context) {
				doFindTest(ctx, client, dbName, int64(ctx.Int("offset")), ctx.StringSlice("expected"))
			})
		}(dbName)
	}
}

func doFindTest(ctx *kt.Context, client *kivik.Client, dbName string, expOffset int64, expected []string) {
	ctx.Parallel()
	db, err := client.DB(dbName)
	// Errors may be deferred here, so only return if we actually get
	// an error.
	if err != nil && !ctx.IsExpectedSuccess(err) {
		return
	}

	rows, err := db.Find(`{"selector":{"_id":{"$gt":null}}}`)
	if !ctx.IsExpectedSuccess(err) {
		return
	}
	docIDs := make([]string, 0, len(expected))
	for rows.Next() {
		var doc struct {
			DocID string `json:"_id"`
			Rev   string `json:"_rev"`
			ID    string `json:"id"`
		}
		if err := rows.ScanDoc(&doc); err != nil {
			ctx.Errorf("Failed to scan doc: %s", err)
		}
		docIDs = append(docIDs, doc.DocID)
	}
	if rows.Err() != nil {
		ctx.Fatalf("Failed to fetch row: %s", rows.Err())
	}
	if d := diff.TextSlices(expected, docIDs); d != "" {
		ctx.Errorf("Unexpected document IDs returned:\n%s\n", d)
	}
}
