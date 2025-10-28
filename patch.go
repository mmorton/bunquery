package bunquery

import (
	"fmt"
	"reflect"

	"github.com/uptrace/bun"
)

type Patch[Target any, Derived any] struct {
	target  *Target
	derived *Derived
}

func (patch *Patch[Target, Derived]) Target() *Target {
	return patch.target
}

type ResourcePatcher[Resource any] interface {
	Target() *Resource
}

func CreatePatch[Target any, Derived any](target *Target, derived *Derived) Patch[Target, Derived] {
	return Patch[Target, Derived]{
		target:  target,
		derived: derived,
	}
}

func (patch *Patch[Target, Derived]) Compile() func(*bun.UpdateQuery) *bun.UpdateQuery {
	ourType := reflect.TypeOf(patch)
	return func(query *bun.UpdateQuery) *bun.UpdateQuery {
		drvValue := reflect.ValueOf(patch.derived).Elem()
		drvType := drvValue.Type()

		if drvType.Kind() != reflect.Struct {
			query.Err(fmt.Errorf("derived must be a struct"))
			return query
		}

		query = query.Model(patch.Target()).WherePK()

		for i := 0; i < drvType.NumField(); i++ {
			field := drvType.Field(i)
			if field.Type == ourType.Elem() {
				continue
			}
			if field.Tag.Get("bunpatch") == "-" {
				continue
			}

			value := drvValue.Field(i)
			if value.Kind() == reflect.Pointer {
				if value.IsNil() {
					continue
				}
			}

			col := PascalToDelimited(field.Name, "_")

			query = query.Set("? = ?", bun.Ident(col), value.Interface())
		}

		return query
	}
}
