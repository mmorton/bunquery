package bunquery

import (
	"context"
	"errors"

	"github.com/uptrace/bun"
)

type dbCtxKey struct{}
type dbCtx struct {
	db   bun.IDB
	mods QueryMods
}

var ErrNoContext = errors.New("no db context")

func getDbCtx(ctx context.Context) (*dbCtx, bool) {
	qctx, ok := ctx.Value(dbCtxKey{}).(*dbCtx)
	return qctx, ok
}

func createDbCtx(ctx context.Context, db bun.IDB, bindings QueryMods) context.Context {
	return context.WithValue(ctx, dbCtxKey{}, &dbCtx{
		db:   db,
		mods: bindings,
	})
}

func NewContext(ctx context.Context, db bun.IDB, mods ...QueryMod) context.Context {
	return createDbCtx(ctx, db, mods)
}

func UseContextMods(ctx context.Context, mods ...QueryMod) (context.Context, error) {
	if dbCtx, ok := getDbCtx(ctx); ok {
		return createDbCtx(ctx, dbCtx.db, mods), nil
	}
	return ctx, ErrNoContext
}
