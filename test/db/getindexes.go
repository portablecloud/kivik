package db

import (
	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/test/kt"
)

func init() {
	kt.Register("GetIndexes", getIndexes)
}

func getIndexes(ctx *kt.Context) {
	ctx.RunAdmin(func(ctx *kt.Context) {
		ctx.Parallel()
		roGetIndexesTests(ctx, ctx.Admin)
	})
	ctx.RunNoAuth(func(ctx *kt.Context) {
		ctx.Parallel()
		roGetIndexesTests(ctx, ctx.NoAuth)
	})
	ctx.RunRW(func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			ctx.Parallel()
			rwGetIndexesTests(ctx, ctx.Admin)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			ctx.Parallel()
			rwGetIndexesTests(ctx, ctx.NoAuth)
		})
	})
}

func roGetIndexesTests(ctx *kt.Context, client *kivik.Client) {
	databases := ctx.MustStringSlice("databases")
	for _, dbname := range databases {
		func(dbname string) {
			ctx.Run(dbname, func(ctx *kt.Context) {
				ctx.Parallel()
				testGetIndexes(ctx, ctx.Admin, dbname, ctx.Interface("indexes"))
			})
		}(dbname)
	}
}

func rwGetIndexesTests(ctx *kt.Context, client *kivik.Client) {
	dbname := ctx.TestDBName()
	defer ctx.Admin.DestroyDB(dbname)
	if err := ctx.Admin.CreateDB(dbname); err != nil {
		ctx.Fatalf("Failed to create db: %s", err)
	}
	dba, err := ctx.Admin.DB(dbname)
	if err != nil {
		ctx.Fatalf("Failed to open db as admin: %s", err)
	}
	if err = dba.CreateIndex("foo", "bar", `{"fields":["foo"]}`); err != nil {
		ctx.Fatalf("Failed to create index: %s", err)
	}
	testGetIndexes(ctx, client, dbname, []kivik.Index{
		kt.AllDocsIndex,
		kivik.Index{
			DesignDoc: "_design/foo",
			Name:      "bar",
			Type:      "json",
			Definition: map[string]interface{}{
				"fields": []map[string]string{
					{"foo": "asc"},
				},
			},
		},
	})
}

func testGetIndexes(ctx *kt.Context, client *kivik.Client, dbname string, expected interface{}) {
	db, err := client.DB(dbname)
	if err != nil {
		ctx.Fatalf("Failed to open db: %s", err)
	}
	indexes, err := db.GetIndexes()
	if !ctx.IsExpectedSuccess(err) {
		return
	}
	if d := diff.AsJSON(expected, indexes); d != "" {
		ctx.Errorf("Indexes differ from expectation:\n%s\n", d)
	}
}
