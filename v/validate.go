package v

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/exp/constraints"
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

type NumericValidator[Numeric interface {
	constraints.Integer | constraints.Float
}, Derived any] interface {
	Gt(check Numeric) Derived
	Lt(check Numeric) Derived
	Positive() Derived
	Negative() Derived
	NonPositive() Derived
	NonNegative() Derived
	Zero() Derived
	BetweenNeq(min, max Numeric) Derived
}

type NumericV[Value interface {
	constraints.Integer | constraints.Float
}, Derived any] struct {
	*GenericV[Value, Derived]
}

func (v *NumericV[Value, Derived]) Gt(req Value) Derived {
	return v.Check(func(value Value) error {
		if value > req {
			return nil
		}
		return fmt.Errorf("value %v is not greater than %v", value, req)
	})
}

func (v *NumericV[Value, Derived]) Lt(req Value) Derived {
	return v.Check(func(value Value) error {
		if value < req {
			return nil
		}
		return fmt.Errorf("value %v is not less than %v", value, req)
	})
}

func (v *NumericV[Value, Derived]) Positive() Derived {
	return v.Check(func(value Value) error {
		if value > 0 {
			return nil
		}
		return fmt.Errorf("value %v is not positive", value)
	})
}

func (v *NumericV[Value, Derived]) Negative() Derived {
	return v.Check(func(value Value) error {
		if value < 0 {
			return nil
		}
		return fmt.Errorf("value %v is not negative", value)
	})
}

func (v *NumericV[Value, Derived]) NonPositive() Derived {
	var Z Value
	return v.Check(func(value Value) error {
		if value == Z || value < 0 {
			return nil
		}
		return fmt.Errorf("value %v is not non-positive", value)
	})
}

func (v *NumericV[Value, Derived]) NonNegative() Derived {
	var Z Value
	return v.Check(func(value Value) error {
		if value == Z || value > 0 {
			return nil
		}
		return fmt.Errorf("value %v is not non-negative", value)
	})
}

func (v *NumericV[Value, Derived]) Zero() Derived {
	var Z Value
	return v.Check(func(value Value) error {
		if value == Z {
			return nil
		}
		return fmt.Errorf("value %v is not zero", value)
	})
}

func (v *NumericV[Value, Derived]) BetweenNeq(min, max Value) Derived {
	return v.Check(func(value Value) error {
		if value > min && value < max {
			return nil
		}
		return fmt.Errorf("value %v is not between %v and %v", value, min, max)
	})
}

type IntegerValidator[Value interface {
	constraints.Integer
}] interface {
	NumericValidator[Value, IntegerValidator[Value]]

	Eq(Value) IntegerValidator[Value]
	In(values ...Value) IntegerValidator[Value]
}

type IntegerV[Value interface {
	constraints.Integer
}] struct {
	*NumericV[Value, IntegerValidator[Value]]
}

func (v *IntegerV[Value]) Eq(req Value) IntegerValidator[Value] {
	return v.Check(func(value Value) error {
		if value == req {
			return nil
		}
		return fmt.Errorf("value %v is not equal to %v", value, req)
	})
}

func (v *IntegerV[Value]) In(req ...Value) IntegerValidator[Value] {
	return v.Check(func(value Value) error {
		for _, r := range req {
			if value == r {
				return nil
			}
		}
		return fmt.Errorf("value %v is not in %v", value, req)
	})
}

func Int[Source any](grp *Set[Source], get func(Source) int) IntegerValidator[int] {
	intV := &IntegerV[int]{}
	intV.NumericV = &NumericV[int, IntegerValidator[int]]{
		GenericV: &GenericV[int, IntegerValidator[int]]{
			derived: intV,
		},
	}
	grp.validations = append(grp.validations, func(source Source) ErrorSet {
		value := get(source)
		err := intV.Validate(value)
		return err
	})
	return intV
}

type FloatValidator[Value interface {
	constraints.Float
}] interface {
	NumericValidator[Value, FloatValidator[Value]]

	Eq(Value, epsilon Value) FloatValidator[Value]
}

type FloatV[Value interface {
	constraints.Float
}] struct {
	*NumericV[Value, FloatValidator[Value]]
}

func (v *FloatV[Value]) Eq(check, epsi Value) FloatValidator[Value] {
	return v.Check(func(value Value) error {
		diff := check - value
		if diff > Value(-1.0)*epsi && diff < epsi {
			return nil
		}
		return fmt.Errorf("value %v is not equal to %v within epsilon %v", value, value, epsi)
	})
}

func Float[Source any](grp *Set[Source], get func(Source) float64) FloatValidator[float64] {
	floatV := &FloatV[float64]{}
	floatV.NumericV = &NumericV[float64, FloatValidator[float64]]{
		GenericV: &GenericV[float64, FloatValidator[float64]]{
			derived: floatV,
		},
	}
	grp.validations = append(grp.validations, func(source Source) ErrorSet {
		value := get(source)
		err := floatV.Validate(value)
		return err
	})
	return floatV
}

type StringValidator interface {
	Contains(string) StringValidator
}

type stringV struct {
	*GenericV[string, StringValidator]
}

func (v *stringV) Contains(substring string) StringValidator {
	return v.Check(func(value string) error {
		if !strings.Contains(value, substring) {
			return fmt.Errorf("value %v does not contain %v", value, substring)
		}
		return nil
	})
}

func String[Source any](grp *Set[Source], get func(Source) string) StringValidator {
	stringV := &stringV{}
	stringV.GenericV = &GenericV[string, StringValidator]{
		derived: stringV,
	}
	grp.validations = append(grp.validations, func(source Source) ErrorSet {
		value := get(source)
		err := stringV.Validate(value)
		return err
	})
	return stringV
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

func Args[Value any](fn func(validator *Set[Value])) func(Value) error {
	validator := &Set[Value]{}
	fn(validator)
	return func(value Value) error {
		for _, check := range validator.validations {
			if err := check(value); err != nil {
				return err
			}
		}
		return nil
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
