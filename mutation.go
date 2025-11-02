package bunquery

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

type MutationCommon interface {
	QueryCommon
	NewInsert() *bun.InsertQuery
	NewUpdate(bindArgs ...any) *bun.UpdateQuery
	NewDelete(bindArgs ...any) *bun.DeleteQuery
}

type MutationDB interface {
	MutationCommon

	Use(binds ...QueryBinder) MutationDB
}

type wrapMutationDB struct {
	ctx     context.Context
	tx      bun.Tx
	binders QueryBindings
}

var _ MutationDB = (*wrapMutationDB)(nil)

func (mut wrapMutationDB) Use(binds ...QueryBinder) MutationDB {
	return wrapMutationDB{
		ctx:     mut.ctx,
		tx:      mut.tx,
		binders: mut.binders.Use(binds...),
	}
}

func (mut wrapMutationDB) NewSelect(bindArgs ...any) *bun.SelectQuery {
	return bindSupportedQuery(mut.ctx, mut.tx, mut.binders, mut.tx.NewSelect(), bindArgs...)
}

func (mut wrapMutationDB) NewInsert() *bun.InsertQuery {
	return mut.tx.NewInsert()
}

func (mut wrapMutationDB) NewUpdate(bindArgs ...any) *bun.UpdateQuery {
	return bindSupportedQuery(mut.ctx, mut.tx, mut.binders, mut.tx.NewUpdate(), bindArgs...)
}

func (mut wrapMutationDB) NewDelete(bindArgs ...any) *bun.DeleteQuery {
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

type Mutation[In any] struct {
	Args      func(args In) (In, error)
	Handler   func(ctx context.Context, db MutationDB, args In) error
	TxOptions *sql.TxOptions
}

type MutationWithOpts[In any, Opts any] struct {
	Args      func(args In) (In, error)
	Handler   func(ctx context.Context, db MutationDB, args In, opts Opts) error
	TxOptions *sql.TxOptions
}

func CreateMutation[In any](def Mutation[In]) func(ctx context.Context, args In) error {
	return func(ctx context.Context, args In) error {
		var err error
		dbCtx, ok := getDbCtx(ctx)
		if !ok {
			return ErrNoContext
		}
		if def.Args != nil {
			args, err = def.Args(args)
			if err != nil {
				return err
			}
		}
		mut := wrapMutationDB{
			ctx:     ctx,
			binders: dbCtx.binders,
		}

		weOwnTx := false

		// If we are already in a Tx, don't create a new one.
		if tx, ok := dbCtx.db.(bun.Tx); ok {
			mut.tx = tx
		} else if tx, err := dbCtx.db.BeginTx(ctx, def.TxOptions); err != nil {
			return err
		} else {
			mut.tx = tx
			// Since we created a new Tx, create a new query context so Tx can be passed through.
			ctx = createDbCtx(ctx, mut.tx, mut.binders)
			weOwnTx = true
		}

		err = def.Handler(ctx, mut, args)

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
			// We don't own the Tx, do not touch.
		}

		return nil
	}
}

func CreateMutationWithOpts[In any, Opts any](def MutationWithOpts[In, Opts]) func(ctx context.Context, args In, opts Opts) error {
	return func(ctx context.Context, args In, opts Opts) error {
		var err error
		dbCtx, ok := getDbCtx(ctx)
		if !ok {
			return ErrNoContext
		}
		if def.Args != nil {
			args, err = def.Args(args)
			if err != nil {
				return err
			}
		}
		mut := wrapMutationDB{
			ctx:     ctx,
			binders: dbCtx.binders,
		}

		weOwnTx := false

		// If we are already in a Tx, don't create a new one.
		if tx, ok := dbCtx.db.(bun.Tx); ok {
			mut.tx = tx
		} else if tx, err := dbCtx.db.BeginTx(ctx, def.TxOptions); err != nil {
			return err
		} else {
			mut.tx = tx
			// Since we created a new Tx, create a new query context so Tx can be passed through.
			ctx = createDbCtx(ctx, mut.tx, mut.binders)
			weOwnTx = true
		}

		err = def.Handler(ctx, mut, args, opts)

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
			// We don't own the Tx, do not touch.
		}

		return nil
	}
}

type QueryMutation[In any, Out any] struct {
	Args      func(args In) (In, error)
	Handler   func(ctx context.Context, db MutationDB, args In) (Out, error)
	TxOptions *sql.TxOptions
}

type QueryMutationWithOpts[In any, Out any, Opts any] struct {
	Args      func(args In) (In, error)
	Handler   func(ctx context.Context, db MutationDB, args In, opts Opts) (Out, error)
	TxOptions *sql.TxOptions
}

func CreateQueryMutation[In any, Out any](def QueryMutation[In, Out]) func(ctx context.Context, args In) (Out, error) {
	var res Out
	return func(ctx context.Context, args In) (Out, error) {
		var err error
		dbCtx, ok := getDbCtx(ctx)
		if !ok {
			return res, ErrNoContext
		}
		if def.Args != nil {
			args, err = def.Args(args)
			if err != nil {
				return res, err
			}
		}
		mut := wrapMutationDB{
			ctx:     ctx,
			binders: dbCtx.binders,
		}

		weOwnTx := false

		// If we are already in a Tx, don't create a new one.
		if tx, ok := dbCtx.db.(bun.Tx); ok {
			mut.tx = tx
		} else if tx, err := dbCtx.db.BeginTx(ctx, def.TxOptions); err != nil {
			return res, err
		} else {
			mut.tx = tx
			// Since we created a new Tx, create a new query context so Tx can be passed through.
			ctx = createDbCtx(ctx, mut.tx, mut.binders)
			weOwnTx = true
		}

		res, err = def.Handler(ctx, mut, args)

		if weOwnTx {
			if err != nil {
				if err := mut.tx.Rollback(); err != nil {
					return res, err
				}
			} else {
				if err := mut.tx.Commit(); err != nil {
					return res, err
				}
			}
		} else {
			// We don't own the Tx, do not touch.
		}

		return res, err
	}
}

func CreateQueryMutationWithOpts[In any, Out any, Opts any](def QueryMutationWithOpts[In, Out, Opts]) func(ctx context.Context, args In, opts Opts) (Out, error) {
	var res Out
	return func(ctx context.Context, args In, opts Opts) (Out, error) {
		var err error
		dbCtx, ok := getDbCtx(ctx)
		if !ok {
			return res, ErrNoContext
		}
		if def.Args != nil {
			args, err = def.Args(args)
			if err != nil {
				return res, err
			}
		}
		mut := wrapMutationDB{
			ctx:     ctx,
			binders: dbCtx.binders,
		}

		weOwnTx := false

		// If we are already in a Tx, don't create a new one.
		if tx, ok := dbCtx.db.(bun.Tx); ok {
			mut.tx = tx
		} else if tx, err := dbCtx.db.BeginTx(ctx, def.TxOptions); err != nil {
			return res, err
		} else {
			mut.tx = tx
			// Since we created a new Tx, create a new query context so Tx can be passed through.
			ctx = createDbCtx(ctx, mut.tx, mut.binders)
			weOwnTx = true
		}

		res, err = def.Handler(ctx, mut, args, opts)

		if weOwnTx {
			if err != nil {
				if err := mut.tx.Rollback(); err != nil {
					return res, err
				}
			} else {
				if err := mut.tx.Commit(); err != nil {
					return res, err
				}
			}
		} else {
			// We don't own the Tx, do not touch.
		}

		return res, err
	}
}
