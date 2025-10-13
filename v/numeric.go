package v

import (
	"fmt"

	"golang.org/x/exp/constraints"
)

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
