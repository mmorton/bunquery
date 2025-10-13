package bunquery

import (
	"context"
	"errors"

	"github.com/uptrace/bun"
)

type queryCtxKey struct{}
type queryCtx struct {
	db      bun.IDB
	binders QueryBindings
}

func NewContext(ctx context.Context, db bun.IDB, binds ...QueryBinder) context.Context {
	return createQueryCtx(ctx, db, NewQueryBindings(binds...))
}

var ErrNoQueryContext = errors.New("No query context.")

func getQueryCtx(ctx context.Context) (*queryCtx, bool) {
	qctx, ok := ctx.Value(queryCtxKey{}).(*queryCtx)
	return qctx, ok
}

func createQueryCtx(ctx context.Context, db bun.IDB, bindings QueryBindings) context.Context {
	return context.WithValue(ctx, queryCtxKey{}, &queryCtx{
		db:      db,
		binders: bindings,
	})
}
