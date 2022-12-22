package ride

import "github.com/wavesplatform/gowaves/pkg/util/common"

type complexityCalculator interface {
	overflow() bool
	complexity() int
	limit() int
	testNativeFunctionComplexity(int) bool
	addNativeFunctionComplexity(int)
	testAdditionalUserFunctionComplexity(int) bool
	addAdditionalUserFunctionComplexity(int)
	testConditionalComplexity() bool
	addConditionalComplexity()
	testReferenceComplexity() bool
	addReferenceComplexity()
	testPropertyComplexity() bool
	addPropertyComplexity()
	setLimit(limit int)
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

func (cc *complexityCalculatorV1) testNativeFunctionComplexity(fc int) bool {
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		return false
	}
	return nc <= cc.l
}

func (cc *complexityCalculatorV1) addNativeFunctionComplexity(fc int) {
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		cc.o = true
	}
	cc.c = nc
}

func (cc *complexityCalculatorV1) testAdditionalUserFunctionComplexity(int) bool {
	return true
}

func (cc *complexityCalculatorV1) addAdditionalUserFunctionComplexity(int) {}

func (cc *complexityCalculatorV1) testOne() bool {
	nc, err := common.AddInt(cc.c, 1)
	if err != nil {
		return false
	}
	return nc <= cc.l
}

func (cc *complexityCalculatorV1) addOne() {
	nc, err := common.AddInt(cc.c, 1)
	if err != nil {
		cc.o = true
	}
	cc.c = nc
}

func (cc *complexityCalculatorV1) testConditionalComplexity() bool {
	return cc.testOne()
}

func (cc *complexityCalculatorV1) addConditionalComplexity() {
	cc.addOne()
}

func (cc *complexityCalculatorV1) testReferenceComplexity() bool {
	return cc.testOne()
}

func (cc *complexityCalculatorV1) addReferenceComplexity() {
	cc.addOne()
}

func (cc *complexityCalculatorV1) testPropertyComplexity() bool {
	return cc.testOne()
}

func (cc *complexityCalculatorV1) addPropertyComplexity() {
	cc.addOne()
}

func (cc *complexityCalculatorV1) setLimit(limit int) {
	if limit < 0 {
		cc.l = 0
	} else {
		cc.l = limit
	}
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

func (cc *complexityCalculatorV2) testNativeFunctionComplexity(fc int) bool {
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		return false
	}
	return nc <= cc.l
}

func (cc *complexityCalculatorV2) addNativeFunctionComplexity(fc int) {
	nc, err := common.AddInt(cc.c, fc)
	if err != nil {
		cc.o = true
	}
	cc.c = nc
}

func (cc *complexityCalculatorV2) testAdditionalUserFunctionComplexity(ic int) bool {
	if ic == cc.c {
		return true
	}
	nc, err := common.AddInt(cc.c, 1)
	if err != nil {
		return false
	}
	return nc <= cc.l
}

func (cc *complexityCalculatorV2) addAdditionalUserFunctionComplexity(ic int) {
	if ic != cc.c {
		return
	}
	nc, err := common.AddInt(cc.c, 1)
	if err != nil {
		cc.o = true
	}
	cc.c = nc
}

func (cc *complexityCalculatorV2) testConditionalComplexity() bool {
	return true
}

func (cc *complexityCalculatorV2) addConditionalComplexity() {}

func (cc *complexityCalculatorV2) testReferenceComplexity() bool {
	return true
}

func (cc *complexityCalculatorV2) addReferenceComplexity() {}

func (cc *complexityCalculatorV2) testPropertyComplexity() bool {
	return true
}

func (cc *complexityCalculatorV2) addPropertyComplexity() {}

func (cc *complexityCalculatorV2) setLimit(limit int) {
	if limit < 0 {
		cc.l = 0
	} else {
		cc.l = limit
	}
}
