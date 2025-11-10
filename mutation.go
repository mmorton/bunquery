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

type wrapMutationDB struct {
	ctx  context.Context
	tx   bun.Tx
	mods QueryMods
}

var _ MutationDB = (*wrapMutationDB)(nil)

func (mut wrapMutationDB) NewSelect(bindArgs ...any) *bun.SelectQuery {
	return applyQueryMods(mut.ctx, mut.tx, mut.mods, mut.tx.NewSelect(), bindArgs...)
}

func (mut wrapMutationDB) NewInsert() *bun.InsertQuery {
	return mut.tx.NewInsert()
}

func (mut wrapMutationDB) NewUpdate(bindArgs ...any) *bun.UpdateQuery {
	return applyQueryMods(mut.ctx, mut.tx, mut.mods, mut.tx.NewUpdate(), bindArgs...)
}

func (mut wrapMutationDB) NewDelete(bindArgs ...any) *bun.DeleteQuery {
	return applyQueryMods(mut.ctx, mut.tx, mut.mods, mut.tx.NewDelete(), bindArgs...)
}

func UseMutation(ctx context.Context, fn func(ctx context.Context, db MutationDB) error, opts ...AnyOpt) error {
	dbCtx, ok := getDbCtx(ctx)
	if !ok {
		return ErrNoContext
	}

	opt := NewMutationOpts(opts...)
	mDB := wrapMutationDB{
		ctx:  ctx,
		mods: dbCtx.mods.Use(opt.Mods...),
	}

	weOwnTx := false

	// If we are already in a Tx, don't create a new one.
	if tx, ok := dbCtx.db.(bun.Tx); ok {
		mDB.tx = tx
	} else if tx, err := dbCtx.db.BeginTx(ctx, opt.TxOptions); err != nil {
		return err
	} else {
		mDB.tx = tx
		// Since we created a new Tx, create a new query context so Tx can be passed through.
		ctx = createDbCtx(ctx, mDB.tx, mDB.mods)
		weOwnTx = true
	}

	err := fn(ctx, mDB)

	if weOwnTx {
		if err != nil {
			if err := mDB.tx.Rollback(); err != nil {
				return err
			}
		} else {
			if err := mDB.tx.Commit(); err != nil {
				return err
			}
		}
	} else {
		// We don't own the Tx, do not touch.
	}

	return err
}

type Mutation[In any] struct {
	Args      func(args In) (In, error)
	Handler   func(ctx context.Context, db MutationDB, args In) error
	Use       []QueryMod
	TxOptions *sql.TxOptions
}

type MutationEx[In any, Ext any] struct {
	Args      func(args In) (In, error)
	Handler   func(ctx context.Context, db MutationDB, args In, ext Ext) error
	Use       []QueryMod
	TxOptions *sql.TxOptions
}

func CreateMutation[In any](def Mutation[In]) func(ctx context.Context, args In) error {
	return func(ctx context.Context, args In) error {
		var err error
		args, err = checkArgs(args, def.Args)
		if err != nil {
			return err
		}
		return UseMutation(ctx, func(ctx context.Context, db MutationDB) error {
			return def.Handler(ctx, db, args)
		}, WithMods(def.Use...))
	}
}

func CreateMutationEx[In any, Ext any](def MutationEx[In, Ext]) func(ctx context.Context, args In, ext Ext) error {
	return func(ctx context.Context, args In, ext Ext) error {
		var err error
		args, err = checkArgs(args, def.Args)
		if err != nil {
			return err
		}
		return UseMutation(ctx, func(ctx context.Context, db MutationDB) error {
			return def.Handler(ctx, db, args, ext)
		}, WithMods(def.Use...))
	}
}

type QueryMutation[In any, Out any] struct {
	Args      func(args In) (In, error)
	Handler   func(ctx context.Context, db MutationDB, args In) (Out, error)
	Use       []QueryMod
	TxOptions *sql.TxOptions
}

type QueryMutationEx[In any, Out any, Ext any] struct {
	Args      func(args In) (In, error)
	Handler   func(ctx context.Context, db MutationDB, args In, ext Ext) (Out, error)
	Use       []QueryMod
	TxOptions *sql.TxOptions
}

func CreateQueryMutation[In any, Out any](def QueryMutation[In, Out]) func(ctx context.Context, args In) (Out, error) {
	return func(ctx context.Context, args In) (Out, error) {
		var res Out
		var err error
		args, err = checkArgs(args, def.Args)
		if err != nil {
			return res, err
		}
		return res, UseMutation(ctx, func(ctx context.Context, db MutationDB) error {
			res, err = def.Handler(ctx, db, args)
			return err
		}, WithMods(def.Use...))
	}
}

func CreateQueryMutationEx[In any, Out any, Ext any](def QueryMutationEx[In, Out, Ext]) func(ctx context.Context, args In, ext Ext) (Out, error) {
	return func(ctx context.Context, args In, ext Ext) (Out, error) {
		var res Out
		var err error
		args, err = checkArgs(args, def.Args)
		if err != nil {
			return res, err
		}
		return res, UseMutation(ctx, func(ctx context.Context, db MutationDB) error {
			res, err = def.Handler(ctx, db, args, ext)
			return err
		}, WithMods(def.Use...))
	}
}
