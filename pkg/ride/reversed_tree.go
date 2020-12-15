package ride

type RNode interface {
	RNode()
}

type RFunc struct {
	Invocation string
	Name       string
	Arguments  []string
	Body       RNode
}

func (a *RFunc) RNode() {}

type RLet struct {
	Name string
	//N    uniqueid
	Body RNode
}

func (a *RLet) RNode() {}

type RCond struct {
	Cond       RNode
	True       RNode
	False      RNode
	Assigments []*RLet
}

func (a *RCond) RNode() {}

type RCall struct {
	Name       string
	Arguments  []RNode
	Assigments []*RLet
	Next       RNode
}

func (a *RCall) RNode() {}

type RRef struct {
	Name       string
	Assigments []*RLet
}

func (a *RRef) RNode() {}

type RLong struct {
	Value int64
}

func (a *RLong) RNode() {}

type RString struct {
	Value string
}

func (a *RString) RNode() {}
