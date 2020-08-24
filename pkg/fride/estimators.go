package fride

// call structure describes how to return from the call
type call struct {
	start    int  // Start position of the function/expression, used as reference value
	ret      int  // Return position, where to go after returning form function/expression
	function bool // Flag to distinguish function and expression calls
	args     int  // number of function arguments
}

// estimationFrame holds values stacks and estimations for both main and alternative branches of execution flow
type estimationFrame struct {
	alternative     bool  // Flag that indicates to count alternative branch estimation
	trunk           int   // Estimation of the trunk
	trunkStack      []int // Values stack of trunk
	branch          int   // Estimation of the alternative branch
	branchStack     []int // Values stack of alternative branch
	nextInstruction int   // Pointer to the instruction next to an alternative branch end
}

func (e *estimationFrame) add(estimation int) {
	if e.alternative {
		e.branch += estimation
	} else {
		e.trunk += estimation
	}
}

func (e *estimationFrame) get() int {
	if e.trunk > e.branch {
		return e.trunk
	}
	return e.branch
}

func (e *estimationFrame) put(pos int) {
	if e.alternative {
		e.branchStack = append(e.branchStack, pos)
	} else {
		e.trunkStack = append(e.trunkStack, pos)
	}
}

func (e *estimationFrame) value(pos int) bool {
	var stack []int
	if e.alternative {
		stack = e.branchStack
	} else {
		stack = e.trunkStack
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == pos {
			return true
		}
	}
	return false
}
