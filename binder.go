package bunquery

import (
	"context"
	"maps"

	"github.com/uptrace/bun"
)

type QueryBinder interface {
	Kind() string
	Bind(context.Context, bun.IDB, QueryBuilderEx)
}

type QueryBindings map[string]QueryBinder

func NewQueryBindings(binds ...QueryBinder) QueryBindings {
	return (QueryBindings{}).Use(binds...)
}

func (bindings QueryBindings) Use(binds ...QueryBinder) QueryBindings {
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

func (bindings QueryBindings) Bind(ctx context.Context, db bun.IDB, qry QueryBuilderEx) {
	for _, bind := range bindings {
		bind.Bind(ctx, db, qry)
	}
}

func bindSupportedQuery[Query any, Source SupportsQueryBuilderEx[Query]](ctx context.Context, db bun.IDB, bindings QueryBindings, qry Source) Source {
	bld := NewQueryBuilderEx(qry)
	bindings.Bind(ctx, db, bld)
	return qry
}
