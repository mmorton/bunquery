package bunquery

import (
	"context"

	"github.com/uptrace/bun"
)

type QueryDB interface {
	NewSelect(bindArgs ...any) *bun.SelectQuery
}

type query struct {
	ctx     context.Context
	db      bun.IDB
	binders QueryBindings
}

var _ QueryDB = (*query)(nil)

func (q query) NewSelect(bindArgs ...any) *bun.SelectQuery {
	return bindSupportedQuery(q.ctx, q.db, q.binders, q.db.NewSelect(), bindArgs...)
}

func Ident[Args any](args Args) (Args, error) {
	return args, nil
}

type QueryFnImpl[QueryDB any, Args any, Res any] func(ctx context.Context, db QueryDB, args Args) (Res, error)
type QueryFn[QueryDB any, Args any, Res any] func(ctx context.Context, args Args) (Res, error)

func CreateQuery[Args any, Res any](fn QueryFnImpl[QueryDB, Args, Res]) QueryFn[QueryDB, Args, Res] {
	return CreateValidatedQuery(Ident, fn)
}

func CreateValidatedQuery[Args any, Res any](argsV func(Args) (Args, error), fn QueryFnImpl[QueryDB, Args, Res]) QueryFn[QueryDB, Args, Res] {
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

type QueryExFnImpl[QueryDB any, Args any, Ex any, Res any] func(ctx context.Context, db QueryDB, args Args, ex Ex) (Res, error)
type QueryExFn[QueryDB any, Args any, Ex any, Res any] func(ctx context.Context, args Args, ex Ex) (Res, error)

func CreateQueryEx[Args any, Ex any, Res any](fn QueryExFnImpl[QueryDB, Args, Ex, Res]) QueryExFn[QueryDB, Args, Ex, Res] {
	return CreateValidatedQueryEx(Ident, fn)
}

func CreateValidatedQueryEx[Args any, Ex any, Res any](argsV func(Args) (Args, error), fn QueryExFnImpl[QueryDB, Args, Ex, Res]) QueryExFn[QueryDB, Args, Ex, Res] {
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
