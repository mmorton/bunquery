package bunquery

import (
	"fmt"
	"reflect"

	"github.com/uptrace/bun/schema"
)

type tableAliasQueryAppender struct {
	value any
}

var _ schema.QueryAppender = (*tableAliasQueryAppender)(nil)

func (r *tableAliasQueryAppender) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	if alias, err := getTableAlias(fmter.Dialect(), r.value); err != nil {
		return nil, err
	} else {
		return fmter.AppendIdent(b, alias), nil
	}
}

func TableAlias(value any) schema.QueryAppender {
	return &tableAliasQueryAppender{value: value}
}

type tableNameQueryAppender struct {
	value any
}

var _ schema.QueryAppender = (*tableNameQueryAppender)(nil)

func (r *tableNameQueryAppender) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	if name, err := getTableName(fmter.Dialect(), r.value); err != nil {
		return nil, err
	} else {
		return fmter.AppendIdent(b, name), nil
	}
}

func TableName(value any) schema.QueryAppender {
	return &tableNameQueryAppender{value: value}
}

func getTableName(dialect schema.Dialect, of any) (string, error) {
	if v, ok := of.(string); ok {
		return v, nil
	} else if tbl := dialect.Tables().Get(reflect.TypeOf(of)); tbl != nil {
		return tbl.Name, nil
	} else {
		return "", fmt.Errorf("could not find table for %v", of)
	}
}

func getTableAlias(dialect schema.Dialect, of any) (string, error) {
	if v, ok := of.(string); ok {
		return v, nil
	} else if tbl := dialect.Tables().Get(reflect.TypeOf(of)); tbl != nil {
		return tbl.Alias, nil
	} else {
		return "", fmt.Errorf("could not find table for %v", of)
	}
}
