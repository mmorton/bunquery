package bunquery

import "github.com/uptrace/bun"

type QueryBuilderEx interface {
	bun.QueryBuilder
	With(name string, query bun.Query) QueryBuilderEx
	Err(err error) QueryBuilderEx
}

type expandedQueryBuilder struct {
	bun.QueryBuilder
	with  func(name string, query bun.Query)
	raise func(err error)
}

type SupportsQueryBuilderEx[P any] interface {
	*P
	QueryBuilder() bun.QueryBuilder
	With(name string, query bun.Query) *P
	Err(err error) *P
}

func NewQueryBuilderEx[Query any, Source SupportsQueryBuilderEx[Query]](qry Source) QueryBuilderEx {
	return &expandedQueryBuilder{
		QueryBuilder: qry.QueryBuilder(),
		with:         func(name string, query bun.Query) { qry.With(name, query) },
		raise:        func(err error) { qry.Err(err) },
	}
}

func (qbx expandedQueryBuilder) With(name string, query bun.Query) QueryBuilderEx {
	qbx.with(name, query)
	return qbx
}

func (qbx expandedQueryBuilder) Err(err error) QueryBuilderEx {
	qbx.raise(err)
	return qbx
}
