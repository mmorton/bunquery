package v

import (
	"errors"
	"fmt"
)

type Set[Value any] struct {
	validations []func(Value) error
}

type GenericValidator[Value any, Derived any] interface {
	Msg(msg string) Derived
	Err(func(value Value, check Value, err error) error) Derived
	Check(func(value Value) error) Derived
}

type requirement[Value any] struct {
	check func(value Value) error
	nok   func(value Value, check Value, err error) error
}

type GenericV[Value any, Derived any] struct {
	derived      Derived
	requirements []*requirement[Value]
}

func NewGenericV[Value any, Derived any](derived Derived) *GenericV[Value, Derived] {
	return &GenericV[Value, Derived]{
		derived: derived,
	}
}

// Always manipulates the last requirement's err/message.
func (v *GenericV[Value, Derived]) Msg(msg string) Derived {
	if len(v.requirements) == 0 {
		return v.derived
	}
	v.requirements[len(v.requirements)-1].nok = func(value, check Value, err error) error {
		return fmt.Errorf("%s", msg)
	}
	return v.derived
}

// Always manipulates the last requirement's err/message.
func (v *GenericV[Value, Derived]) Err(err func(value Value, check Value, err error) error) Derived {
	if len(v.requirements) == 0 {
		return v.derived
	}
	v.requirements[len(v.requirements)-1].nok = err
	return v.derived
}

func (v *GenericV[Value, Derived]) Check(check func(value Value) error) Derived {
	v.requirements = append(v.requirements, &requirement[Value]{
		check: check,
		nok: func(value, check Value, err error) error {
			return err
		},
	})
	return v.derived
}

func (v *GenericV[Value, Derived]) Validate(value Value) error {
	errs := make([]error, 0, len(v.requirements))
	for _, r := range v.requirements {
		if err := r.check(value); err != nil {
			if r.nok != nil {
				errs = append(errs, r.nok(value, value, err))
			}
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func Struct[Source any, Value any](grp *Set[Source], get func(Source) Value, next func(*Set[Value])) {
	nextV := &Set[Value]{}
	grp.validations = append(grp.validations, func(source Source) error {
		errs := make([]error, 0, len(nextV.validations))
		for _, check := range nextV.validations {
			if err := check(get(source)); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
		return nil
	})
	next(nextV)
}

func Args[Value any](fn func(validator *Set[Value])) func(Value) (Value, error) {
	var zed Value
	validator := &Set[Value]{}
	fn(validator)
	return func(value Value) (Value, error) {
		errs := make([]error, 0, len(validator.validations))
		for _, check := range validator.validations {
			if err := check(value); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return zed, errors.Join(errs...)
		}
		return value, nil
	}
}

func example() {
	type GetValueArgs struct {
		Idx int
	}

	check := Args(func(validator *Set[*GetValueArgs]) {
		Int(validator, func(args *GetValueArgs) int { return args.Idx }).Gt(5)
	})

	args := &GetValueArgs{Idx: 10}
	check(args)
}
