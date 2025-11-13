package bunquery

import (
	"fmt"
	"reflect"

	"github.com/uptrace/bun/schema"
)

type QueryAppenderFunc func(fmter schema.Formatter, b []byte) ([]byte, error)

var _ schema.QueryAppender = (QueryAppenderFunc)(nil)

func (f QueryAppenderFunc) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	return f(fmter, b)
}

func TableAlias(value any) QueryAppenderFunc {
	return func(fmter schema.Formatter, b []byte) ([]byte, error) {
		if name, err := getTableAlias(fmter.Dialect(), value); err != nil {
			return nil, err
		} else {
			return fmter.AppendIdent(b, name), nil
		}
	}
}

func TableName(value any) QueryAppenderFunc {
	return func(fmter schema.Formatter, b []byte) ([]byte, error) {
		if name, err := getTableName(fmter.Dialect(), value); err != nil {
			return nil, err
		} else {
			return fmter.AppendIdent(b, name), nil
		}
	}
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
