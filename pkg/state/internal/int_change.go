package internal

import (
	"golang.org/x/exp/constraints"

	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type IntChange[T constraints.Integer] struct {
	present bool
	forced  bool // forced change, must be accounted even if value is zero and present is false
	value   T
}

func NewIntChange[T constraints.Integer](v T) IntChange[T] {
	return IntChange[T]{
		present: v != 0, // zero change can't be considered as present
		forced:  false,  // false by default
		value:   v,
	}
}

func (v IntChange[T]) Value() T { return v.value }

func (v IntChange[T]) Present() bool { return v.present }

func (v IntChange[T]) IsAccountable() bool {
	return v.Present() && v.Value() != 0 || v.forced
}

func (v IntChange[T]) ToForced() IntChange[T] {
	cpy := v
	cpy.forced = true
	return cpy
}

func (v IntChange[T]) Add(other IntChange[T]) (IntChange[T], error) {
	r, err := common.AddInt(v.Value(), other.Value())
	if err != nil {
		return IntChange[T]{}, err
	}
	return IntChange[T]{
		present: v.Present() || other.Present(), // if one of the values has been present, the result is present
		forced:  v.forced || other.forced,       // if one of the values has been forced, the result is forced
		value:   r,
	}, nil
}
