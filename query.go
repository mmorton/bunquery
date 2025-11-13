package bunquery

import (
	"context"

	"github.com/uptrace/bun"
)

type QueryDB interface {
	Unwrap() bun.IDB
	NewSelect(bindArgs ...any) *bun.SelectQuery
}

type wrapQueryDB struct {
	ctx  context.Context
	db   bun.IDB
	mods QueryMods
}

var _ QueryDB = (*wrapQueryDB)(nil)

func (q wrapQueryDB) Unwrap() bun.IDB {
	return q.db
}

func (q wrapQueryDB) NewSelect(bindArgs ...any) *bun.SelectQuery {
	return applyQueryMods(q.ctx, q.db, q.mods, q.db.NewSelect(), bindArgs...)
}

func UseQuery(ctx context.Context, fn func(ctx context.Context, db QueryDB) error, opts ...AnyOpt) error {
	dbCtx, ok := getDbCtx(ctx)
	if !ok {
		return ErrNoContext
	}
	opt := NewQueryOpts(opts...)
	qDB := wrapQueryDB{
		ctx:  ctx,
		db:   dbCtx.db,
		mods: dbCtx.mods.Use(opt.Mods...),
	}
	return fn(ctx, qDB)
}

func UseQueryDB(ctx context.Context, opts ...AnyOpt) (QueryDB, error) {
	dbCtx, ok := getDbCtx(ctx)
	if !ok {
		return nil, ErrNoContext
	}
	opt := NewQueryOpts(opts...)
	qDB := wrapQueryDB{
		ctx:  ctx,
		db:   dbCtx.db,
		mods: dbCtx.mods.Use(opt.Mods...),
	}
	return qDB, nil
}

type Query[In any, Out any] struct {
	Args    func(args In) (In, error)
	Handler func(ctx context.Context, db QueryDB, args In) (Out, error)
	Use     []QueryMod
}

type QueryEx[In any, Out any, Ext any] struct {
	Args    func(args In) (In, error)
	Handler func(ctx context.Context, db QueryDB, args In, ext Ext) (Out, error)
	Use     []QueryMod
}

func checkArgs[In any](args In, argsFn func(args In) (In, error)) (In, error) {
	if argsFn != nil {
		return argsFn(args)
	}
	return args, nil
}

func CreateQuery[In any, Out any](def Query[In, Out]) func(ctx context.Context, args In) (Out, error) {
	return func(ctx context.Context, args In) (Out, error) {
		var res Out
		var err error
		args, err = checkArgs(args, def.Args)
		if err != nil {
			return res, err
		}
		return res, UseQuery(ctx, func(ctx context.Context, db QueryDB) error {
			res, err = def.Handler(ctx, db, args)
			return err
		}, WithMods(def.Use...))
	}
}

func CreateQueryEx[In any, Out any, Ext any](def QueryEx[In, Out, Ext]) func(ctx context.Context, args In, ext Ext) (Out, error) {
	return func(ctx context.Context, args In, ext Ext) (Out, error) {
		var res Out
		var err error
		args, err = checkArgs(args, def.Args)
		if err != nil {
			return res, err
		}
		return res, UseQuery(ctx, func(ctx context.Context, db QueryDB) error {
			res, err = def.Handler(ctx, db, args, ext)
			return err
		}, WithMods(def.Use...))
	}
}
