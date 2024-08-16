package ride

import (
	"fmt"

	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type complexityCalculator interface {
	error() error
	complexity() int
	limit() int
	testNativeFunctionComplexity(string, int) error
	addNativeFunctionComplexity(string, int)
	testAdditionalUserFunctionComplexity(string, int) error
	addAdditionalUserFunctionComplexity(string, int)
	testConditionalComplexity() error
	addConditionalComplexity()
	testReferenceComplexity() error
	addReferenceComplexity()
	testPropertyComplexity() error
	addPropertyComplexity()
	setLimit(limit uint32)
	clone() complexityCalculator
}

type complexityCalculatorError interface {
	error
	EvaluationErrorWrapType() EvaluationError
}

type integerOverflowError struct {
	name                 string
	nodeComplexity       int
	calculatorComplexity int
	innerError           error
}

func newIntegerOverflowError(
	name string,
	nodeComplexity int,
	calculatorComplexity int,
	innerError error,
) integerOverflowError {
	return integerOverflowError{
		name:                 name,
		nodeComplexity:       nodeComplexity,
		calculatorComplexity: calculatorComplexity,
		innerError:           innerError,
	}
}

func (i integerOverflowError) Unwrap() error {
	return i.innerError
}

func (i integerOverflowError) Error() string {
	return fmt.Sprintf("node '%s' with complexity %d has overflowed integer calculator complexity %d: %v",
		i.name,
		i.nodeComplexity,
		i.calculatorComplexity,
		i.innerError,
	)
}

func (i integerOverflowError) EvaluationErrorWrapType() EvaluationError {
	return ComplexityLimitExceed // compatibility with the old implementation
}

type complexityLimitExceededError struct {
	name                 string
	nodeComplexity       int
	calculatorComplexity int
	complexityLimit      int
}

func newComplexityLimitOverflowError(
	name string,
	nodeComplexity int,
	calculatorComplexity int,
	limit int,
) complexityLimitExceededError {
	return complexityLimitExceededError{
		name:                 name,
		nodeComplexity:       nodeComplexity,
		calculatorComplexity: calculatorComplexity,
		complexityLimit:      limit,
	}
}

func (o complexityLimitExceededError) EvaluationErrorWrapType() EvaluationError {
	return ComplexityLimitExceed
}

func (o complexityLimitExceededError) Error() string {
	return fmt.Sprintf("node '%s' with complexity %d has exceeded the complexity limit %d with result complexity %d",
		o.name, o.nodeComplexity, o.complexityLimit, o.calculatorComplexity,
	)
}

type zeroComplexityError struct {
	name string
}

func newZeroComplexityError(name string) zeroComplexityError {
	return zeroComplexityError{name: name}
}

func (z zeroComplexityError) EvaluationErrorWrapType() EvaluationError {
	return EvaluationFailure
}

func (z zeroComplexityError) Error() string {
	return fmt.Sprintf("node '%s' has zero complexity", z.name)
}

func newComplexityCalculatorByRideV6Activation(rideV6 bool) complexityCalculator {
	if rideV6 {
		return &complexityCalculatorV2{}
	}
	return &complexityCalculatorV1{}
}

type complexityCalculatorV1 struct {
	err complexityCalculatorError
	c   int
	l   int
}

func (cc *complexityCalculatorV1) error() error {
	return cc.err
}

func (cc *complexityCalculatorV1) complexity() int {
	return cc.c
}

func (cc *complexityCalculatorV1) limit() int {
	return cc.l
}

func (cc *complexityCalculatorV1) clone() complexityCalculator {
	return &complexityCalculatorV1{
		err: cc.err,
		c:   cc.c,
		l:   cc.l,
	}
}

func (cc *complexityCalculatorV1) testNativeFunctionComplexity(name string, fc int) error {
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		return newIntegerOverflowError(name, fc, cc.complexity(), err)
	}
	if nc > cc.l {
		return newComplexityLimitOverflowError(name, fc, nc, cc.limit())
	}
	return nil
}

func (cc *complexityCalculatorV1) addNativeFunctionComplexity(name string, fc int) {
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		cc.err = newIntegerOverflowError(name, fc, cc.complexity(), err)
	}
	cc.c = nc
}

func (cc *complexityCalculatorV1) testAdditionalUserFunctionComplexity(string, int) error {
	return nil
}

func (cc *complexityCalculatorV1) addAdditionalUserFunctionComplexity(string, int) {}

func (cc *complexityCalculatorV1) testOne(name string) error {
	const complexity = 1
	nc, err := common.AddInt(cc.c, complexity)
	if err != nil {
		return newIntegerOverflowError(name, complexity, cc.complexity(), err)
	}
	if nc > cc.l {
		return newComplexityLimitOverflowError(name, complexity, nc, cc.limit())
	}
	return nil
}

func (cc *complexityCalculatorV1) addOne(name string) {
	const complexity = 1
	nc, err := common.AddInt(cc.c, complexity)
	if err != nil {
		cc.err = newIntegerOverflowError(name, complexity, cc.complexity(), err)
	}
	cc.c = nc
}

const (
	conditionalNodeCodeName  = "<conditional_node>"
	referenceNodeCodeName    = "<reference_node>"
	propertyNodeCallCodeName = "<property_node>"
)

func (cc *complexityCalculatorV1) testConditionalComplexity() error {
	return cc.testOne(conditionalNodeCodeName)
}

func (cc *complexityCalculatorV1) addConditionalComplexity() {
	cc.addOne(conditionalNodeCodeName)
}

func (cc *complexityCalculatorV1) testReferenceComplexity() error {
	return cc.testOne(referenceNodeCodeName)
}

func (cc *complexityCalculatorV1) addReferenceComplexity() {
	cc.addOne(referenceNodeCodeName)
}

func (cc *complexityCalculatorV1) testPropertyComplexity() error {
	return cc.testOne(propertyNodeCallCodeName)
}

func (cc *complexityCalculatorV1) addPropertyComplexity() {
	cc.addOne(propertyNodeCallCodeName)
}

func (cc *complexityCalculatorV1) setLimit(limit uint32) {
	cc.l = int(limit)
}

type complexityCalculatorV2 struct {
	err complexityCalculatorError
	c   int
	l   int
}

func (cc *complexityCalculatorV2) error() error {
	return cc.err
}

func (cc *complexityCalculatorV2) complexity() int {
	return cc.c
}

func (cc *complexityCalculatorV2) limit() int {
	return cc.l
}

func (cc *complexityCalculatorV2) clone() complexityCalculator {
	return &complexityCalculatorV2{
		err: cc.err,
		c:   cc.c,
		l:   cc.l,
	}
}

func (cc *complexityCalculatorV2) testNativeFunctionComplexity(name string, fc int) error {
	if fc == 0 { // sanity check: zero complexity for functions is not allowed since Ride V6
		return newZeroComplexityError(name)
	}
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		return newIntegerOverflowError(name, fc, cc.complexity(), err)
	}
	if nc > cc.l {
		return newComplexityLimitOverflowError(name, fc, nc, cc.limit())
	}
	return nil
}

func (cc *complexityCalculatorV2) addNativeFunctionComplexity(name string, fc int) {
	if fc == 0 { // sanity check: zero complexity for functions is not allowed since Ride V6
		cc.err = newZeroComplexityError(name)
		return // don't add zero complexity
	}
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		cc.err = newIntegerOverflowError(name, fc, cc.complexity(), err)
	}
	cc.c = nc
}

func (cc *complexityCalculatorV2) testAdditionalUserFunctionComplexity(name string, ic int) error {
	// In case of no complexity spent in the user function we don't have to test the new complexity value.
	// Just return `true` and current complexity.
	// That means we can safely call companion function `addAdditionalUserFunctionComplexity`.
	if ic == cc.c {
		return nil
	}
	const additionalComplexity = 1
	nc, err := common.AddInt(cc.c, additionalComplexity)
	if err != nil {
		return newIntegerOverflowError(name, additionalComplexity, cc.complexity(), err)
	}
	if nc > cc.l {
		return newComplexityLimitOverflowError(name, additionalComplexity, nc, cc.limit())
	}
	return nil
}

func (cc *complexityCalculatorV2) addAdditionalUserFunctionComplexity(name string, ic int) {
	// The condition is opposite to the previous function because if complexity was spent in the user function
	// we don't have to add additional 1.
	if ic != cc.c {
		return
	}
	const additionalComplexity = 1
	nc, err := common.AddInt(cc.c, additionalComplexity)
	if err != nil {
		cc.err = newIntegerOverflowError(name, additionalComplexity, cc.complexity(), err)
	}
	cc.c = nc
}

func (cc *complexityCalculatorV2) testConditionalComplexity() error {
	return nil
}

func (cc *complexityCalculatorV2) addConditionalComplexity() {}

func (cc *complexityCalculatorV2) testReferenceComplexity() error {
	return nil
}

func (cc *complexityCalculatorV2) addReferenceComplexity() {}

func (cc *complexityCalculatorV2) testPropertyComplexity() error {
	return nil
}

func (cc *complexityCalculatorV2) addPropertyComplexity() {}

func (cc *complexityCalculatorV2) setLimit(limit uint32) {
	cc.l = int(limit)
}
