package bunquery

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

type QueryFnImpl[QueryDB any, Args any, Res any] func(ctx context.Context, db QueryDB, args Args) (Res, error)
type QueryFn[QueryDB any, Args any, Res any] func(ctx context.Context, args Args) (Res, error)

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

type MutationFnImpl[MutationDB any, Args any] func(ctx context.Context, db MutationDB, args Args) error
type MutationFn[MutationDB any, Args any] func(ctx context.Context, args Args) error

type MutationDB interface {
	QueryDB

	NewInsert() *bun.InsertQuery
	NewUpdate(bindArgs ...any) *bun.UpdateQuery
	NewDelete(bindArgs ...any) *bun.DeleteQuery
}

type mutation struct {
	ctx     context.Context
	tx      bun.Tx
	binders QueryBindings
}

var _ MutationDB = (*mutation)(nil)

func (mut mutation) NewSelect(bindArgs ...any) *bun.SelectQuery {
	return bindSupportedQuery(mut.ctx, mut.tx, mut.binders, mut.tx.NewSelect(), bindArgs...)
}

func (mut mutation) NewInsert() *bun.InsertQuery {
	return mut.tx.NewInsert()
}

func (mut mutation) NewUpdate(bindArgs ...any) *bun.UpdateQuery {
	return bindSupportedQuery(mut.ctx, mut.tx, mut.binders, mut.tx.NewUpdate(), bindArgs...)
}

func (mut mutation) NewDelete(bindArgs ...any) *bun.DeleteQuery {
	return bindSupportedQuery(mut.ctx, mut.tx, mut.binders, mut.tx.NewDelete(), bindArgs...)
}

type MutationOptionsAware interface {
	TxOptions() *sql.TxOptions
}

type MutationOptions struct {
	sql.TxOptions
}

type MutationOptionFn func(*MutationOptions) *MutationOptions

func WithIsolationLevel(level sql.IsolationLevel) MutationOptionFn {
	return func(opt *MutationOptions) *MutationOptions {
		opt.TxOptions.Isolation = level
		return opt
	}
}

func WithReadOnly(readOnly bool) MutationOptionFn {
	return func(opt *MutationOptions) *MutationOptions {
		opt.TxOptions.ReadOnly = readOnly
		return opt
	}
}

func createMutationOptions(opts ...MutationOptionFn) *MutationOptions {
	opt := &MutationOptions{}
	for _, fn := range opts {
		opt = fn(opt)
	}
	return opt
}

func CreateMutation[Args any](fn MutationFnImpl[MutationDB, Args], opts ...MutationOptionFn) MutationFn[MutationDB, Args] {
	return func(ctx context.Context, args Args) error {
		c, ok := getQueryCtx(ctx)
		if !ok {
			return ErrNoQueryContext
		}

		mut := mutation{
			ctx:     ctx,
			binders: c.binders,
		}

		opt := createMutationOptions(opts...)

		if tx, ok := c.db.(bun.Tx); ok {
			mut.tx = tx
		} else if tx, err := c.db.BeginTx(ctx, &opt.TxOptions); err != nil {
			return err
		} else {
			mut.tx = tx
		}

		var err error
		if err = fn(ctx, mut, args); err != nil {
			if err := mut.tx.Rollback(); err != nil {
				return err
			}
			return err
		} else if err := mut.tx.Commit(); err != nil {
			return err
		}

		return nil
	}
}
