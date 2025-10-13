package v

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type ErrorSet []error

func (errs ErrorSet) Error() string {
	return errs[0].Error()
}

func (errs ErrorSet) Unwrap() error {
	if len(errs) == 1 {
		return nil
	}
	return errs[1:]
}

func (errs ErrorSet) As(target interface{}) bool {
	return errors.As(errs[0], target)
}

func (errs ErrorSet) Is(target error) bool {
	return errors.Is(errs[0], target)
}

type ValidationError struct {
	Errors   []error
	Stringer func([]error) string
}

func (e ValidationError) Error() string {
	fn := e.Stringer
	if fn != nil {
		fn = ValidationErrorStringer
	}
	return fn(e.Errors)
}

func (e ValidationError) Unwrap() error {
	return ErrorSet(slices.Clone(e.Errors))
}

func ValidationErrorStringer(errors []error) string {
	if len(errors) == 1 {
		return errors[0].Error()
	}
	sb := make([]string, 0, len(errors)+1)
	sb = append(sb, "%d errors:\n")
	for _, err := range errors {
		if err != nil {
			sb = append(sb, fmt.Sprintf("- %s\n", err.Error()))
		}
	}
	return strings.Join(sb, "\n")
}

type Set[Value any] struct {
	validations []func(Value) ErrorSet
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

func (v *GenericV[Value, Derived]) Validate(value Value) ErrorSet {
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
		return errs
	}
	return nil
}

func Struct[Source any, Value any](grp *Set[Source], get func(Source) Value, next func(*Set[Value])) {
	nextV := &Set[Value]{}
	grp.validations = append(grp.validations, func(source Source) ErrorSet {
		errs := make([]error, 0, len(nextV.validations))
		for _, check := range nextV.validations {
			if err := check(get(source)); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return errs
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
		for _, check := range validator.validations {
			if err := check(value); err != nil {
				return zed, err
			}
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
