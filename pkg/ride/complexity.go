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

// addComplexity is an inlinable non-generic equivalent of common.AddInt for complexity values.
// It returns the same sum as common.AddInt, including zero in case of overflow, and false instead of an error.
func addComplexity(a, b int) (int, bool) {
	c := a + b
	if (c > a) == (b > 0) {
		return c, true
	}
	return 0, false
}

// testComplexity checks that complexity fc can be added to the current complexity c without overflow and
// without exceeding the limit l. Error values are built only on the failure paths.
func testComplexity(name string, fc, c, l int) error {
	nc, ok := addComplexity(c, fc)
	if !ok {
		return overflowError(name, fc, c)
	}
	if nc > l {
		return newComplexityLimitOverflowError(name, fc, nc, l)
	}
	return nil
}

// overflowError builds the error of overflowed addition of fc to c. The inner error is taken from common.AddInt,
// so the message is exactly the same as before.
//
//go:noinline
func overflowError(name string, fc, c int) complexityCalculatorError {
	_, err := common.AddInt(c, fc)
	return newIntegerOverflowError(name, fc, c, err)
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
	return testComplexity(name, fc, cc.c, cc.l)
}

func (cc *complexityCalculatorV1) addNativeFunctionComplexity(name string, fc int) {
	nc, ok := addComplexity(cc.c, fc)
	if !ok {
		cc.err = overflowError(name, fc, cc.c)
	}
	cc.c = nc
}

func (cc *complexityCalculatorV1) testAdditionalUserFunctionComplexity(string, int) error {
	return nil
}

func (cc *complexityCalculatorV1) addAdditionalUserFunctionComplexity(string, int) {}

func (cc *complexityCalculatorV1) testOne(name string) error {
	const complexity = 1
	return testComplexity(name, complexity, cc.c, cc.l)
}

func (cc *complexityCalculatorV1) addOne(name string) {
	const complexity = 1
	nc, ok := addComplexity(cc.c, complexity)
	if !ok {
		cc.err = overflowError(name, complexity, cc.c)
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
	return testComplexity(name, fc, cc.c, cc.l)
}

func (cc *complexityCalculatorV2) addNativeFunctionComplexity(name string, fc int) {
	if fc == 0 { // sanity check: zero complexity for functions is not allowed since Ride V6
		cc.err = newZeroComplexityError(name)
		return // don't add zero complexity
	}
	nc, ok := addComplexity(cc.c, fc)
	if !ok {
		cc.err = overflowError(name, fc, cc.c)
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
	return testComplexity(name, additionalComplexity, cc.c, cc.l)
}

func (cc *complexityCalculatorV2) addAdditionalUserFunctionComplexity(name string, ic int) {
	// The condition is opposite to the previous function because if complexity was spent in the user function
	// we don't have to add additional 1.
	if ic != cc.c {
		return
	}
	const additionalComplexity = 1
	nc, ok := addComplexity(cc.c, additionalComplexity)
	if !ok {
		cc.err = overflowError(name, additionalComplexity, cc.c)
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
