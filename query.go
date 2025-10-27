package bunquery

import (
	"context"

	"github.com/uptrace/bun"
)

type QueryCommon interface {
	NewSelect(bindArgs ...any) *bun.SelectQuery
}

type QueryDB interface {
	QueryCommon
	Use(binds ...QueryBinder) QueryDB
}

type query struct {
	ctx     context.Context
	db      bun.IDB
	binders QueryBindings
}

var _ QueryDB = (*query)(nil)

func (q query) Use(binds ...QueryBinder) QueryDB {
	return query{
		ctx:     q.ctx,
		db:      q.db,
		binders: q.binders.Use(binds...),
	}
}

func (q query) NewSelect(bindArgs ...any) *bun.SelectQuery {
	return bindSupportedQuery(q.ctx, q.db, q.binders, q.db.NewSelect(), bindArgs...)
}

func Ident[Args any](args Args) (Args, error) {
	return args, nil
}

type QueryFnImpl[DB any, Args any, Res any] func(ctx context.Context, db DB, args Args) (Res, error)
type QueryFn[DB any, Args any, Res any] func(ctx context.Context, args Args) (Res, error)

func CreateQuery[Args any, Res any](fn QueryFnImpl[QueryDB, Args, Res]) QueryFn[QueryDB, Args, Res] {
	return CreateQueryV(Ident, fn)
}

func CreateQueryV[Args any, Res any](argsV func(Args) (Args, error), fn QueryFnImpl[QueryDB, Args, Res]) QueryFn[QueryDB, Args, Res] {
	var zed Res
	return func(ctx context.Context, args Args) (Res, error) {
		c, ok := getQueryCtx(ctx)
		if !ok {
			return zed, ErrNoQueryContext
		}
		args, err := argsV(args)
		if err != nil {
			return zed, nil
		}
		q := query{
			ctx:     ctx,
			db:      c.db,
			binders: c.binders,
		}
		return fn(ctx, q, args)
	}
}

type QueryExtendedFnImpl[DB any, Args any, Ex any, Res any] func(ctx context.Context, db DB, args Args, ex Ex) (Res, error)
type QueryExtendedFn[DB any, Args any, Ex any, Res any] func(ctx context.Context, args Args, ex Ex) (Res, error)

func CreateQueryExtended[Args any, Ex any, Res any](fn QueryExtendedFnImpl[QueryDB, Args, Ex, Res]) QueryExtendedFn[QueryDB, Args, Ex, Res] {
	return CreateQueryExtendedV(Ident, fn)
}

func CreateQueryExtendedV[Args any, Ex any, Res any](argsV func(Args) (Args, error), fn QueryExtendedFnImpl[QueryDB, Args, Ex, Res]) QueryExtendedFn[QueryDB, Args, Ex, Res] {
	var zed Res
	return func(ctx context.Context, args Args, ex Ex) (Res, error) {
		c, ok := getQueryCtx(ctx)
		if !ok {
			return zed, ErrNoQueryContext
		}
		args, err := argsV(args)
		if err != nil {
			return zed, nil
		}
		q := query{
			ctx:     ctx,
			db:      c.db,
			binders: c.binders,
		}
		return fn(ctx, q, args, ex)
	}
}
