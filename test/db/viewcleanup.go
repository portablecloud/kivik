package db

import (
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/test/kt"
)

func init() {
	kt.Register("ViewCleanup", viewCleanup)
}

func viewCleanup(ctx *kt.Context) {
	ctx.RunRW(func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			ctx.Parallel()
			testViewCleanup(ctx, ctx.Admin)
		})
		ctx.RunNoAuth(func(ctx *kt.Context) {
			ctx.Parallel()
			testViewCleanup(ctx, ctx.NoAuth)
		})
	})
}

func testViewCleanup(ctx *kt.Context, client *kivik.Client) {
	dbname := ctx.TestDBName()
	defer ctx.Admin.DestroyDB(dbname)
	if err := ctx.Admin.CreateDB(dbname); err != nil {
		ctx.Fatalf("Failed to create test db: %s", err)
	}
	db, err := client.DB(dbname)
	if err != nil {
		ctx.Fatalf("Failed to connect to db: %s", err)
	}
	ctx.CheckError(db.ViewCleanup())
}
