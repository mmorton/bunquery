package bunquery

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

type QueryFnImpl[QueryDB any, Args any, Res any] func(ctx context.Context, db QueryDB, args Args) (Res, error)
type QueryFn[QueryDB any, Args any, Res any] func(ctx context.Context, args Args) (Res, error)

type QueryDB interface {
	NewSelect() *bun.SelectQuery
}

type MutationDB interface {
	QueryDB

	NewInsert() *bun.InsertQuery
	NewUpdate() *bun.UpdateQuery
	NewDelete() *bun.DeleteQuery
}

type mutation struct {
	ctx     context.Context
	tx      bun.Tx
	binders QueryBindings
}

var _ MutationDB = (*mutation)(nil)

func (mut mutation) NewSelect() *bun.SelectQuery {
	return bindSupportedQuery(mut.ctx, mut.tx, mut.binders, mut.tx.NewSelect())
}

func (mut mutation) NewInsert() *bun.InsertQuery {
	return mut.tx.NewInsert()
}

func (mut mutation) NewUpdate() *bun.UpdateQuery {
	return bindSupportedQuery(mut.ctx, mut.tx, mut.binders, mut.tx.NewUpdate())
}

func (mut mutation) NewDelete() *bun.DeleteQuery {
	return mut.tx.NewDelete()
}

type MutationOptions struct {
	sql.TxOptions
}

type MutationOptionFn func(*MutationOptions) *MutationOptions

func createMutationOptions(opts ...MutationOptionFn) *MutationOptions {
	opt := &MutationOptions{}
	for _, fn := range opts {
		opt = fn(opt)
	}
	return opt
}

func CreateMutation[Args any, Res any](fn QueryFnImpl[MutationDB, Args, Res], opts ...MutationOptionFn) QueryFn[MutationDB, Args, Res] {
	var zed Res
	var res Res
	return func(ctx context.Context, args Args) (Res, error) {
		c, ok := getQueryCtx(ctx)
		if !ok {
			return res, ErrNoQueryContext
		}

		mut := mutation{
			ctx:     ctx,
			binders: c.binders,
		}

		opt := createMutationOptions(opts...)

		if tx, ok := c.db.(bun.Tx); ok {
			mut.tx = tx
		} else if tx, err := c.db.BeginTx(ctx, &opt.TxOptions); err != nil {
			return res, err
		} else {
			mut.tx = tx
		}

		var err error
		if res, err = fn(ctx, mut, args); err != nil {
			if err := mut.tx.Rollback(); err != nil {
				return zed, err
			}
			return zed, err
		} else if err := mut.tx.Commit(); err != nil {
			return zed, err
		}

		return res, nil
	}
}

type query struct {
	ctx     context.Context
	db      bun.IDB
	binders QueryBindings
}

var _ QueryDB = (*query)(nil)

func (q query) NewSelect() *bun.SelectQuery {
	return bindSupportedQuery(q.ctx, q.db, q.binders, q.db.NewSelect())
}

func CreateQuery[Args any, Res any](fn QueryFnImpl[QueryDB, Args, Res]) QueryFn[QueryDB, Args, Res] {
	var res Res
	return func(ctx context.Context, args Args) (Res, error) {
		c, ok := getQueryCtx(ctx)
		if !ok {
			return res, ErrNoQueryContext
		}
		q := query{
			ctx:     ctx,
			db:      c.db,
			binders: c.binders,
		}
		return fn(ctx, q, args)
	}
}
