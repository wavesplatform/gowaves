package ride

import (
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type complexityCalculator interface {
	overflow() bool
	complexity() int
	limit() int
	testNativeFunctionComplexity(int) (bool, int)
	addNativeFunctionComplexity(string, int)
	testAdditionalUserFunctionComplexity(int) (bool, int)
	addAdditionalUserFunctionComplexity(string, int)
	testConditionalComplexity() (bool, int)
	addConditionalComplexity()
	testReferenceComplexity() (bool, int)
	addReferenceComplexity()
	testPropertyComplexity() (bool, int)
	addPropertyComplexity()
	setLimit(limit uint32)
}

func newComplexityCalculator(lib ast.LibraryVersion, limit uint32) complexityCalculator {
	if lib >= ast.LibV6 {
		return &complexityCalculatorV2{l: int(limit)}
	}
	return &complexityCalculatorV1{l: int(limit)}
}

func newComplexityCalculatorByRideV6Activation(rideV6 bool) complexityCalculator {
	if rideV6 {
		return &complexityCalculatorV2{}
	}
	return &complexityCalculatorV1{}
}

type complexityCalculatorV1 struct {
	o bool
	c int
	l int
}

func (cc *complexityCalculatorV1) overflow() bool {
	return cc.o
}

func (cc *complexityCalculatorV1) complexity() int {
	return cc.c
}

func (cc *complexityCalculatorV1) limit() int {
	return cc.l
}

func (cc *complexityCalculatorV1) testNativeFunctionComplexity(fc int) (bool, int) {
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		return false, nc
	}
	return nc <= cc.l, nc
}

func (cc *complexityCalculatorV1) addNativeFunctionComplexity(name string, fc int) {
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		cc.o = true
	}
	zap.L().Debug("addNativeFunctionComplexityV1",
		zap.String("name", name),
		zap.Int("fc", fc),
		zap.Int("sum", nc),
	)
	cc.c = nc
}

func (cc *complexityCalculatorV1) testAdditionalUserFunctionComplexity(int) (bool, int) {
	return true, cc.c
}

func (cc *complexityCalculatorV1) addAdditionalUserFunctionComplexity(string, int) {}

func (cc *complexityCalculatorV1) testOne() (bool, int) {
	nc, err := common.AddInt(cc.c, 1)
	if err != nil {
		return false, nc
	}
	return nc <= cc.l, nc
}

func (cc *complexityCalculatorV1) addOne() {
	nc, err := common.AddInt(cc.c, 1)
	if err != nil {
		cc.o = true
	}
	cc.c = nc
}

func (cc *complexityCalculatorV1) testConditionalComplexity() (bool, int) {
	return cc.testOne()
}

func (cc *complexityCalculatorV1) addConditionalComplexity() {
	cc.addOne()
	zap.L().Debug("addConditionalComplexityV1", zap.Int("sum", cc.c))
}

func (cc *complexityCalculatorV1) testReferenceComplexity() (bool, int) {
	return cc.testOne()
}

func (cc *complexityCalculatorV1) addReferenceComplexity() {
	cc.addOne()
	zap.L().Debug("addReferenceComplexityV1", zap.Int("sum", cc.c))
}

func (cc *complexityCalculatorV1) testPropertyComplexity() (bool, int) {
	return cc.testOne()
}

func (cc *complexityCalculatorV1) addPropertyComplexity() {
	cc.addOne()
	zap.L().Debug("addPropertyComplexityV1", zap.Int("sum", cc.c))
}

func (cc *complexityCalculatorV1) setLimit(limit uint32) {
	cc.l = int(limit)
}

type complexityCalculatorV2 struct {
	o bool
	c int
	l int
}

func (cc *complexityCalculatorV2) overflow() bool {
	return cc.o
}

func (cc *complexityCalculatorV2) complexity() int {
	return cc.c
}

func (cc *complexityCalculatorV2) limit() int {
	return cc.l
}

func (cc *complexityCalculatorV2) testNativeFunctionComplexity(fc int) (bool, int) {
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		return false, nc
	}
	return nc <= cc.l, nc
}

func (cc *complexityCalculatorV2) addNativeFunctionComplexity(name string, fc int) {
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		cc.o = true
	}
	zap.L().Debug("addNativeFunctionComplexityV2",
		zap.String("name", name),
		zap.Int("fc", fc),
		zap.Int("sum", nc),
	)
	cc.c = nc
}

func (cc *complexityCalculatorV2) testAdditionalUserFunctionComplexity(ic int) (bool, int) {
	// In case of no complexity spent in the user function we don't have to test the new complexity value.
	// Just return `true` and current complexity.
	// That means we can safely call companion function `addAdditionalUserFunctionComplexity`.
	if ic == cc.c {
		return true, cc.c
	}
	nc, err := common.AddInt(cc.c, 1)
	if err != nil {
		return false, nc
	}
	return nc <= cc.l, nc
}

func (cc *complexityCalculatorV2) addAdditionalUserFunctionComplexity(name string, ic int) {
	// The condition is opposite to the previous function because if complexity was spent in the user function
	// we don't have to add additional 1.
	if ic != cc.c {
		return
	}
	nc, err := common.AddInt(cc.c, 1)
	if err != nil {
		cc.o = true
	}
	zap.L().Debug("addAdditionalUserFunctionComplexityV2", zap.String("name", name), zap.Int("sum", nc))
	cc.c = nc
}

func (cc *complexityCalculatorV2) testConditionalComplexity() (bool, int) {
	return true, cc.c
}

func (cc *complexityCalculatorV2) addConditionalComplexity() {}

func (cc *complexityCalculatorV2) testReferenceComplexity() (bool, int) {
	return true, cc.c
}

func (cc *complexityCalculatorV2) addReferenceComplexity() {}

func (cc *complexityCalculatorV2) testPropertyComplexity() (bool, int) {
	return true, cc.c
}

func (cc *complexityCalculatorV2) addPropertyComplexity() {}

func (cc *complexityCalculatorV2) setLimit(limit uint32) {
	cc.l = int(limit)
}
