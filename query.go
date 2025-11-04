package bunquery

import (
	"context"

	"github.com/uptrace/bun"
)

type QueryDB interface {
	NewSelect(bindArgs ...any) *bun.SelectQuery
}

type wrapQueryDB struct {
	ctx     context.Context
	db      bun.IDB
	binders QueryBindings
}

var _ QueryDB = (*wrapQueryDB)(nil)

func (q wrapQueryDB) Use(binds ...QueryBinder) QueryDB {
	return wrapQueryDB{
		ctx:     q.ctx,
		db:      q.db,
		binders: q.binders.Use(binds...),
	}
}

func (q wrapQueryDB) NewSelect(bindArgs ...any) *bun.SelectQuery {
	return bindSupportedQuery(q.ctx, q.db, q.binders, q.db.NewSelect(), bindArgs...)
}

func Ident[Args any](args Args) (Args, error) {
	return args, nil
}

type Query[In any, Out any] struct {
	Args    func(args In) (In, error)
	Handler func(ctx context.Context, db QueryDB, args In) (Out, error)
	Use     []QueryBinder
}

type QueryWithOpts[In any, Out any, Opts any] struct {
	Args    func(args In) (In, error)
	Handler func(ctx context.Context, db QueryDB, args In, opts Opts) (Out, error)
	Use     []QueryBinder
}

func CreateQuery[In any, Out any](def Query[In, Out]) func(ctx context.Context, args In) (Out, error) {
	var zero Out
	return func(ctx context.Context, args In) (Out, error) {
		var err error
		dbCtx, ok := getDbCtx(ctx)
		if !ok {
			return zero, ErrNoContext
		}
		if def.Args != nil {
			args, err = def.Args(args)
			if err != nil {
				return zero, err
			}
		}
		qDB := wrapQueryDB{
			ctx:     ctx,
			db:      dbCtx.db,
			binders: dbCtx.binders.Use(def.Use...),
		}
		return def.Handler(ctx, qDB, args)
	}
}

func CreateQueryWithOpts[In any, Out any, Opts any](def QueryWithOpts[In, Out, Opts]) func(ctx context.Context, args In, opts Opts) (Out, error) {
	var zero Out
	return func(ctx context.Context, args In, opts Opts) (Out, error) {
		var err error
		dbCtx, ok := getDbCtx(ctx)
		if !ok {
			return zero, ErrNoContext
		}
		if def.Args != nil {
			args, err = def.Args(args)
			if err != nil {
				return zero, err
			}
		}
		qDB := wrapQueryDB{
			ctx:     ctx,
			db:      dbCtx.db,
			binders: dbCtx.binders.Use(def.Use...),
		}
		return def.Handler(ctx, qDB, args, opts)
	}
}
