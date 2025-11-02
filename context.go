package bunquery

import (
	"context"
	"errors"

	"github.com/uptrace/bun"
)

type dbCtxKey struct{}
type dbCtx struct {
	db      bun.IDB
	binders QueryBindings
}

var ErrNoContext = errors.New("No db context.")

func getDbCtx(ctx context.Context) (*dbCtx, bool) {
	qctx, ok := ctx.Value(dbCtxKey{}).(*dbCtx)
	return qctx, ok
}

func createDbCtx(ctx context.Context, db bun.IDB, bindings QueryBindings) context.Context {
	return context.WithValue(ctx, dbCtxKey{}, &dbCtx{
		db:      db,
		binders: bindings,
	})
}

func NewContext(ctx context.Context, db bun.IDB, binds ...QueryBinder) context.Context {
	return createDbCtx(ctx, db, NewQueryBindings(binds...))
}

func BindContext(ctx context.Context, binds ...QueryBinder) (context.Context, error) {
	if dbCtx, ok := getDbCtx(ctx); ok {
		return createDbCtx(ctx, dbCtx.db, NewQueryBindings(binds...)), nil
	}
	return ctx, ErrNoContext
}
