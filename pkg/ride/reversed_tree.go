package ride

type RNode interface {
	RNode()
}

type RFunc struct {
	Invocation              string
	Name                    string
	Arguments               []string
	ArgumentsWithInvocation []string
	Body                    RNode
}

func (a *RFunc) RNode() {}

type RLet struct {
	Name string
}

func (a *RLet) RNode() {}

type RRet struct {
}

func (a *RRet) RNode() {}

type RCond struct {
}

func (a *RCond) RNode() {}

type RCondEnd struct {
}

func (a *RCondEnd) RNode() {}

type RCondTrue struct {
}

func (a *RCondTrue) RNode() {}

type RCondFalse struct {
}

func (a *RCondFalse) RNode() {}

type RFuncEnd struct {
}

func (a *RFuncEnd) RNode() {}

//type RCond struct {
//	Cond       RNode
//	True       RNode
//	False      RNode
//	Assigments []*RLet
//}
//
//func (a *RCond) RNode() {}

type RCall struct {
	Name string
	//Arguments  []RNode
	Argn uint16
	//Assigments []*RLet
	//Next       RNode
}

func (a *RCall) RNode() {}

//func reverseRnodes(a []RNode) []RNode {
//	out := make([]RNode, len(a))
//	for i := 0; i < len(a); i++ {
//		out[len(a)-1-i] = a[i]
//	}
//	return out
//}

//func (a *RCall) CallTree() []RNode {
//	d := []RNode{
//		a,
//	}
//	for i := len(a.Arguments) - 1; i >= 0; i-- {
//		switch t := a.Arguments[i].(type) {
//		case *RConst:
//			d = append(d, a.Arguments[i])
//		case *RCall:
//			d = append(d, t.CallTree()...)
//		default:
//			panic("")
//		}
//	}
//	return d
//}

//
//type RRef struct {
//	Name       string
//	Assigments []*RLet
//}
//
//func (a *RRef) RNode() {}

//type RLong struct {
//	Value int64
//}
//
//func (a *RLong) RNode() {}

type RConst struct {
	Value rideType
}

func (a *RConst) RNode() {}

type RProperty struct{}

func (a *RProperty) RNode() {}

//type RString struct {
//	Value string
//}
//
//func (a *RString) RNode() {}

//type RBoolean struct {
//	Value bool
//}
//
//func (a *RBoolean) RNode() {}

type RReferenceNode struct {
	Name string
}

func (a *RReferenceNode) RNode() {}

type RStart struct {
	Name string
}

func (a *RStart) RNode() {}

type RDef struct {
	Name      string
	Arguments []string
}

func (a *RDef) RNode() {}

type RBody struct {
	Name string
}

func (a *RBody) RNode() {}
