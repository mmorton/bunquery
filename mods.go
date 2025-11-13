package bunquery

import (
	"context"
	"maps"
	"slices"

	"github.com/uptrace/bun"
)

type QueryMod interface {
	Kind() string
	Bind(context.Context, bun.IDB, QueryBuilderEx, ...any)
}

type QueryMods []QueryMod

func (m QueryMods) Use(mods ...QueryMod) QueryMods {
	if len(mods) == 0 {
		return m
	}
	kinds := make(map[string]QueryMod, len(m))
	for _, mod := range m {
		kinds[mod.Kind()] = mod
	}
	for _, mod := range mods {
		kinds[mod.Kind()] = mod
	}
	return slices.Collect(maps.Values(kinds))
}

func (m QueryMods) Bind(ctx context.Context, db bun.IDB, qry QueryBuilderEx, args ...any) {
	for _, mod := range m {
		mod.Bind(ctx, db, qry, args...)
	}
}

func applyQueryMods[Query any, Source SupportsQueryBuilderEx[Query]](ctx context.Context, db bun.IDB, mods QueryMods, query Source, args ...any) Source {
	qbx := NewQueryBuilderEx(query)
	mods.Bind(ctx, db, qbx, args...)
	return query
}

type funcQueryMod struct {
	kind string
	fn   func(ctx context.Context, iDB bun.IDB, query QueryBuilderEx, args ...any)
}

var _ QueryMod = (*funcQueryMod)(nil)

func (b *funcQueryMod) Kind() string { return b.kind }
func (b *funcQueryMod) Bind(ctx context.Context, iDB bun.IDB, query QueryBuilderEx, args ...any) {
	b.fn(ctx, iDB, query, args...)
}

func NewQueryMod(kind string, fn func(ctx context.Context, iDB bun.IDB, query QueryBuilderEx, args ...any)) QueryMod {
	return &funcQueryMod{
		kind: kind,
		fn:   fn,
	}
}
