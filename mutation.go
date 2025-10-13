package bunquery

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

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

type MutationFnImpl[MutationDB any, Args any] func(ctx context.Context, db MutationDB, args Args) error
type MutationFn[MutationDB any, Args any] func(ctx context.Context, args Args) error

func CreateMutation[Args any](fn MutationFnImpl[MutationDB, Args], opts ...MutationOptionFn) MutationFn[MutationDB, Args] {
	return CreateValidatedMutation(Ident, fn)
}

func CreateValidatedMutation[Args any](argsV func(Args) (Args, error), fn MutationFnImpl[MutationDB, Args], opts ...MutationOptionFn) MutationFn[MutationDB, Args] {
	return func(ctx context.Context, args Args) error {
		c, ok := getQueryCtx(ctx)
		if !ok {
			return ErrNoQueryContext
		}

		args, err := argsV(args)
		if err != nil {
			return nil
		}

		mut := mutation{
			ctx:     ctx,
			binders: c.binders,
		}

		opt := createMutationOptions(opts...)
		weOwnTx := false

		// If we are already in a Tx, don't create a new one.
		if tx, ok := c.db.(bun.Tx); ok {
			mut.tx = tx
		} else if tx, err := c.db.BeginTx(ctx, &opt.TxOptions); err != nil {
			return err
		} else {
			mut.tx = tx
			// Since we created a new Tx, create a new query context so Tx can be passed through.
			ctx = createQueryCtx(ctx, mut.tx, mut.binders)
			weOwnTx = true
		}

		err = fn(ctx, mut, args)

		if weOwnTx {
			if err != nil {
				if err := mut.tx.Rollback(); err != nil {
					return err
				}
			} else {
				if err := mut.tx.Commit(); err != nil {
					return err
				}
			}
		} else {
			// We don't own the Tx, don't do anything to it.
		}

		return nil
	}
}

type QueryMutationFnImpl[MutationDB any, Args any, Res any] func(ctx context.Context, db MutationDB, args Args) (Res, error)
type QueryMutationFn[MutationDB any, Args any, Res any] func(ctx context.Context, args Args) (Res, error)

func CreateQueryMutation[Args any, Res any](fn QueryMutationFnImpl[MutationDB, Args, Res], opts ...MutationOptionFn) QueryMutationFn[MutationDB, Args, Res] {
	return CreateValidatedQueryMutation(Ident, fn)
}

func CreateValidatedQueryMutation[Args any, Res any](argsV func(Args) (Args, error), fn QueryMutationFnImpl[MutationDB, Args, Res], opts ...MutationOptionFn) QueryMutationFn[MutationDB, Args, Res] {
	var zed Res
	return func(ctx context.Context, args Args) (Res, error) {
		c, ok := getQueryCtx(ctx)
		if !ok {
			return zed, ErrNoQueryContext
		}

		args, err := argsV(args)
		if err != nil {
			return zed, err
		}

		mut := mutation{
			ctx:     ctx,
			binders: c.binders,
		}

		opt := createMutationOptions(opts...)
		weOwnTx := false

		// If we are already in a Tx, don't create a new one.
		if tx, ok := c.db.(bun.Tx); ok {
			mut.tx = tx
		} else if tx, err := c.db.BeginTx(ctx, &opt.TxOptions); err != nil {
			return zed, err
		} else {
			mut.tx = tx
			// Since we created a new Tx, create a new query context so Tx can be passed through.
			ctx = createQueryCtx(ctx, mut.tx, mut.binders)
			weOwnTx = true
		}

		res, err := fn(ctx, mut, args)

		if weOwnTx {
			if err != nil {
				if err := mut.tx.Rollback(); err != nil {
					return zed, err
				}
			} else {
				if err := mut.tx.Commit(); err != nil {
					return zed, err
				}
			}
		} else {
			// We don't own the Tx, don't do anything to it.
		}

		return res, nil
	}
}

type QueryMutationExFnImpl[MutationDB any, Args any, Ex any, Res any] func(ctx context.Context, db MutationDB, args Args, ex Ex) (Res, error)
type QueryMutationExFn[MutationDB any, Args any, Ex any, Res any] func(ctx context.Context, args Args, ex Ex) (Res, error)

func CreateQueryMutationEx[Args any, Ex any, Res any](fn QueryMutationExFnImpl[MutationDB, Args, Ex, Res], opts ...MutationOptionFn) QueryMutationExFn[MutationDB, Args, Ex, Res] {
	return CreateValidatedQueryMutationEx(Ident, fn)
}

func CreateValidatedQueryMutationEx[Args any, Ex any, Res any](argsV func(Args) (Args, error), fn QueryMutationExFnImpl[MutationDB, Args, Ex, Res], opts ...MutationOptionFn) QueryMutationExFn[MutationDB, Args, Ex, Res] {
	var zed Res
	return func(ctx context.Context, args Args, ex Ex) (Res, error) {
		c, ok := getQueryCtx(ctx)
		if !ok {
			return zed, ErrNoQueryContext
		}

		args, err := argsV(args)
		if err != nil {
			return zed, err
		}

		mut := mutation{
			ctx:     ctx,
			binders: c.binders,
		}

		opt := createMutationOptions(opts...)
		weOwnTx := false

		// If we are already in a Tx, don't create a new one.
		if tx, ok := c.db.(bun.Tx); ok {
			mut.tx = tx
		} else if tx, err := c.db.BeginTx(ctx, &opt.TxOptions); err != nil {
			return zed, err
		} else {
			mut.tx = tx
			// Since we created a new Tx, create a new query context so Tx can be passed through.
			ctx = createQueryCtx(ctx, mut.tx, mut.binders)
			weOwnTx = true
		}

		res, err := fn(ctx, mut, args, ex)

		if weOwnTx {
			if err != nil {
				if err := mut.tx.Rollback(); err != nil {
					return zed, err
				}
			} else {
				if err := mut.tx.Commit(); err != nil {
					return zed, err
				}
			}
		} else {
			// We don't own the Tx, don't do anything to it.
		}

		return res, nil
	}
}
