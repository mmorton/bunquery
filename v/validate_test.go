package v

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicValidation(t *testing.T) {
	type GetValueArgs struct {
		Idx int
	}

	check := Args(func(set *Set[*GetValueArgs]) {
		Int(set, func(args *GetValueArgs) int { return args.Idx }).Gt(5)
	})

	var err error
	_, err = check(&GetValueArgs{Idx: 6})
	assert.NoError(t, err)

	_, err = check(&GetValueArgs{Idx: 4})
	assert.Error(t, err)

	_, err = check(&GetValueArgs{Idx: 5})
	assert.Error(t, err)
}

func TestMoreReqsValidation(t *testing.T) {
	type GetValueArgs struct {
		Idx int
	}

	check := Args(func(set *Set[*GetValueArgs]) {
		Int(set, func(args *GetValueArgs) int { return args.Idx }).Gt(5).Lt(7)
	})

	var err error
	_, err = check(&GetValueArgs{Idx: 6})
	assert.NoError(t, err)

	_, err = check(&GetValueArgs{Idx: 4})
	assert.Error(t, err)

	_, err = check(&GetValueArgs{Idx: 8})
	assert.Error(t, err)
}

func TestMorePropValidation(t *testing.T) {
	type GetValueArgs struct {
		Idx int
		Num int
	}

	check := Args(func(set *Set[*GetValueArgs]) {
		Int(set, func(args *GetValueArgs) int { return args.Idx }).Gt(5)
		Int(set, func(args *GetValueArgs) int { return args.Num }).BetweenNeq(2, 4)
	})

	var err error
	_, err = check(&GetValueArgs{Idx: 6, Num: 3})
	assert.NoError(t, err)

	_, err = check(&GetValueArgs{Idx: 4, Num: 3})
	assert.Error(t, err)

	_, err = check(&GetValueArgs{Idx: 6, Num: 2})
	assert.Error(t, err)
}

func TestNestedStructValidation(t *testing.T) {
	type Nested struct {
		Val int
	}
	type GetValueArgs struct {
		Idx    int
		Nested Nested
	}

	check := Args(func(set *Set[*GetValueArgs]) {
		Int(set, func(args *GetValueArgs) int { return args.Idx }).Gt(5)
		Struct(set, func(args *GetValueArgs) Nested { return args.Nested }, func(set *Set[Nested]) {
			Int(set, func(nst Nested) int { return nst.Val }).Gt(10)
		})
	})

	var err error
	_, err = check(&GetValueArgs{Idx: 6, Nested: Nested{Val: 11}})
	assert.NoError(t, err)

	_, err = check(&GetValueArgs{Idx: 4, Nested: Nested{Val: 11}})
	assert.Error(t, err)

	_, err = check(&GetValueArgs{Idx: 5, Nested: Nested{Val: 9}})
	assert.Error(t, err)
}

func TestStringLengthValidation(t *testing.T) {
	type GetValueArgs struct {
		Value string
	}

	check := Args(func(set *Set[*GetValueArgs]) {
		String(set, func(args *GetValueArgs) string { return args.Value }).Length(5)
	})

	var err error
	_, err = check(&GetValueArgs{Value: "hello"})
	assert.NoError(t, err)
}
