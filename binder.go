package bunquery

import (
	"context"
	"maps"

	"github.com/uptrace/bun"
)

type QueryBinder interface {
	Kind() string
	Bind(context.Context, bun.IDB, QueryBuilderEx, ...any)
}

type QueryBindings map[string]QueryBinder

func NewQueryBindings(binds ...QueryBinder) QueryBindings {
	return (QueryBindings{}).Use(binds...)
}

func (bindings QueryBindings) Use(binds ...QueryBinder) QueryBindings {
	if len(binds) == 0 {
		return bindings
	}
	var res QueryBindings
	if bindings != nil {
		res = maps.Clone(bindings)
	} else {
		res = QueryBindings{}
	}
	for _, bind := range binds {
		res[bind.Kind()] = bind
	}
	return res
}

func (bindings QueryBindings) Bind(ctx context.Context, db bun.IDB, qry QueryBuilderEx, args ...any) {
	for _, bind := range bindings {
		bind.Bind(ctx, db, qry, args...)
	}
}

func bindSupportedQuery[Query any, Source SupportsQueryBuilderEx[Query]](ctx context.Context, db bun.IDB, bindings QueryBindings, qry Source, args ...any) Source {
	bld := NewQueryBuilderEx(qry)
	bindings.Bind(ctx, db, bld, args...)
	return qry
}
