package estimation

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
)

type function struct {
	expr ast.Expr
	args []string
}

type expression struct {
	expr      ast.Expr
	evaluated bool
}

type context struct {
	name        string
	parent      *context
	expressions map[string]expression
	functions   map[string]function
}

func (c *context) express(name string, e expression) {
	c.expressions[name] = e
}

func (c *context) expression(name string) (expression, bool) {
	e, ok := c.expressions[name]
	if ok {
		return e, true
	}
	if c.parent != nil {
		return c.parent.expression(name)
	}
	return expression{}, false
}

func (c *context) declare(name string, f function) {
	c.functions[name] = f
}

func (c *context) declaration(name string) (function, bool) {
	f, ok := c.functions[name]
	if ok {
		return f, true
	}
	if c.parent != nil {
		return c.parent.declaration(name)
	}
	return function{}, false
}

func (c *context) copy() *context {
	e := make(map[string]expression)
	for k, v := range c.expressions {
		e[k] = v
	}
	f := make(map[string]function)
	for k, v := range c.functions {
		f[k] = v
	}
	var pc *context = nil
	if c.parent != nil {
		pc = c.parent.copy()
	}
	return &context{
		name:        c.name,
		parent:      pc,
		expressions: e,
		functions:   f,
	}
}

type contexts struct {
	items   map[string]*context
	current string
}

func newContexts(variables map[string]ast.Expr) *contexts {
	e := make(map[string]expression, len(variables))
	for k, v := range variables {
		e[k] = expression{expr: v, evaluated: true}
	}
	e["height"] = expression{expr: ast.NewLong(0), evaluated: true}
	e["tx"] = expression{expr: ast.NewObject(map[string]ast.Expr{}), evaluated: true}
	e["this"] = expression{expr: ast.NewObject(map[string]ast.Expr{}), evaluated: true}

	root := &context{
		name:        "root",
		parent:      nil,
		expressions: e,
		functions:   make(map[string]function),
	}

	return &contexts{
		items:   map[string]*context{"root": root},
		current: "root",
	}
}

func (c *contexts) root() *context {
	root, ok := c.items["root"]
	if !ok {
		panic("no root context")
	}
	return root
}

func (c *contexts) deleteRootExpression(name string) {
	root, ok := c.items["root"]
	if !ok {
		panic("no root context")
	}
	delete(root.expressions, name)
}

func (c *contexts) setRootExpression(name string, expr expression) {
	root, ok := c.items["root"]
	if !ok {
		panic("no root context")
	}
	root.expressions[name] = expr
}

func (c *contexts) curr() *context {
	context, ok := c.items[c.current]
	if !ok {
		panic("no current context")
	}
	return context
}

func (c *contexts) branch(name string) *context {
	b := &context{
		name:        name,
		parent:      c.curr(),
		expressions: make(map[string]expression),
		functions:   make(map[string]function),
	}
	c.items[name] = b
	return b
}

func (c *contexts) change(context *context) error {
	_, ok := c.items[context.name]
	if !ok {
		return errors.Errorf("no context '%s'", context.name)
	}
	c.current = context.name
	return nil
}

func (c *contexts) copy() *contexts {
	items := make(map[string]*context)
	for k, v := range c.items {
		items[k] = v.copy()
	}
	return &contexts{
		items:   items,
		current: c.current,
	}
}

type Costs struct {
	Functions map[string]uint64
	DApp      uint64
	Verifier  uint64
}

type Estimator struct {
	Version   int
	catalogue *Catalogue
	contexts  *contexts
}

func NewEstimator(version int, catalogue *Catalogue, variables map[string]ast.Expr) *Estimator {
	return &Estimator{
		Version:   version,
		catalogue: catalogue,
		contexts:  newContexts(variables),
	}
}

func (e *Estimator) Estimate(script *ast.Script) (Costs, error) {
	if script.IsDapp() {
		return e.EstimateDApp(script)
	}
	return e.EstimateVerifier(script)
}

func (e *Estimator) EstimateDApp(script *ast.Script) (Costs, error) {
	if !script.IsDapp() {
		return Costs{}, errors.New("estimation: not a DApp")
	}
	e.contexts.deleteRootExpression("tx")
	e.contexts.setRootExpression("height", expression{expr: ast.NewLong(0), evaluated: true})
	e.contexts.setRootExpression("this", expression{expr: ast.NewUnit(), evaluated: false})
	var declarationsCost uint64 = 0
	for _, d := range script.DApp.Declarations {
		switch decl := d.(type) {
		case *ast.LetExpr:
			e.contexts.setRootExpression(decl.Name, expression{decl.Value, false})
			declarationsCost += 5
		case *ast.FuncDeclaration:
			cc := e.contexts.branch(decl.Name)
			for _, a := range decl.Args {
				cc.express(a, expression{&ast.BooleanExpr{Value: true}, false})
			}
			err := e.contexts.change(cc)
			if err != nil {
				return Costs{}, errors.Wrap(err, "estimation")
			}
			fc, err := e.estimate(decl.Body)
			if err != nil {
				return Costs{}, errors.Wrap(err, "estimation")
			}
			ac := uint64(len(decl.Args) * 5)
			e.catalogue.user[decl.Name] = ac + fc
			err = e.contexts.change(e.contexts.root())
			if err != nil {
				return Costs{}, errors.Wrap(err, "estimation")
			}
			declarationsCost += 5
		}
	}
	r := Costs{
		Functions: make(map[string]uint64, len(script.DApp.CallableFuncs)),
		DApp:      0,
		Verifier:  0,
	}
	var callableCost uint64 = 0
	for _, cf := range script.DApp.CallableFuncs {
		cc := e.contexts.copy()
		c, err := e.estimateCallable(cf)
		if err != nil {
			return Costs{}, errors.Wrap(err, "estimation")
		}
		e.contexts = cc
		r.Functions[cf.FuncDecl.Name] = c + declarationsCost
		if c > callableCost {
			callableCost = c
		}
	}
	v, err := e.estimateCallable(script.DApp.Verifier)
	if err != nil {
		return Costs{}, errors.Wrap(err, "estimation")
	}
	r.Verifier = v + declarationsCost
	if v > callableCost {
		callableCost = v
	}
	r.DApp = declarationsCost + callableCost
	return r, nil
}

func (e *Estimator) EstimateVerifier(script *ast.Script) (Costs, error) {
	if script.IsDapp() {
		return Costs{}, errors.New("estimation: not a simple script")
	}
	verifierCost, err := e.estimate(script.Verifier)
	if err != nil {
		return Costs{}, errors.Wrap(err, "estimation")
	}
	return Costs{Verifier: verifierCost}, nil
}

func (e *Estimator) estimateCallable(callable *ast.DappCallableFunc) (uint64, error) {
	if callable == nil {
		return 0, nil
	}
	e.contexts.setRootExpression(callable.AnnotationInvokeName, expression{&ast.BooleanExpr{Value: true}, false})
	err := e.contexts.change(e.contexts.root())
	if err != nil {
		return 0, err
	}
	cost, err := e.estimate(callable.FuncDecl)
	if err != nil {
		return 0, err
	}
	return cost + 10, nil
}

func (e *Estimator) estimate(expr ast.Expr) (uint64, error) {
	switch ce := expr.(type) {
	case *ast.StringExpr, *ast.LongExpr, *ast.BooleanExpr, *ast.BytesExpr:
		return 1, nil

	case ast.Exprs:
		var total uint64 = 0
		for _, item := range ce {
			c, err := e.estimate(item)
			if err != nil {
				return 0, err
			}
			total += c
		}
		return total, nil

	case *ast.Block:
		switch e.Version {
		case 2:
			tmp := e.contexts.copy()
			cc := e.contexts.curr()
			cc.express(ce.Let.Name, expression{ce.Let.Value, false})
			bc, err := e.estimate(ce.Body)
			if err != nil {
				return 0, err
			}
			e.contexts = tmp
			return bc + 5, nil
		default:
			cc := e.contexts.curr()
			cc.express(ce.Let.Name, expression{ce.Let.Value, false})
			bc, err := e.estimate(ce.Body)
			if err != nil {
				return 0, err
			}
			return bc + 5, nil
		}

	case *ast.BlockV2:
		switch declaration := ce.Decl.(type) {
		case *ast.LetExpr:
			switch e.Version {
			case 2:
				tmp := e.contexts.copy()
				cc := e.contexts.curr()
				cc.express(declaration.Name, expression{declaration.Value, false})
				bc, err := e.estimate(ce.Body)
				if err != nil {
					return 0, err
				}
				e.contexts = tmp
				return bc + 5, nil
			default:
				cc := e.contexts.curr()
				cc.express(declaration.Name, expression{declaration.Value, false})
				bc, err := e.estimate(ce.Body)
				if err != nil {
					return 0, err
				}
				return bc + 5, nil
			}
		case *ast.FuncDeclaration:
			switch e.Version {
			case 2:
				cc := e.contexts.curr()
				cc.declare(declaration.Name, function{declaration.Body, declaration.Args})
				for _, a := range declaration.Args {
					cc.express(a, expression{&ast.BooleanExpr{Value: true}, false})
				}
				// TODO: no branching, estimation is broken
				//e.contexts.branch(declaration.Name)
			default:
				rc := e.contexts.root()
				for _, a := range declaration.Args {
					rc.express(a, expression{&ast.BooleanExpr{Value: true}, false})
				}
				fc, err := e.estimate(declaration.Body)
				if err != nil {
					return 0, err
				}
				ac := uint64(len(declaration.Args) * 5) // arguments cost = 5 * number of arguments
				e.catalogue.user[declaration.Name] = ac + fc
			}
			bc, err := e.estimate(ce.Body)
			if err != nil {
				return 0, err
			}
			return 5 + bc, nil
		default:
			return 0, errors.Errorf("unsupported content of type %T", ce.Decl)
		}

	case *ast.FuncCallExpr:
		cc, err := e.estimate(ce.Func)
		if err != nil {
			return 0, err
		}
		return cc, nil

	case *ast.FunctionCall:
		var fc uint64
		callContext := e.contexts.curr()
		if fd, ok := callContext.declaration(ce.Name); ok {
			// Estimate parameters that was passed to the function
			fc += uint64(ce.Argc * 5)
			ac, err := e.estimate(ce.Argv)
			if err != nil {
				return 0, err
			}
			// TODO no branching for a while
			// Change context to the function's one
			//functionContext, ok := e.contexts.items[ce.Name]
			//if !ok {
			//	return 0, errors.Errorf("no function context '%s'", ce.Name)
			//}
			//err = e.contexts.change(functionContext)
			//if err != nil {
			//	return 0, err
			//}
			if na := len(fd.args); na != ce.Argc {
				return 0, errors.Errorf("unexpected number of arguments %d, function '%s' accepts %d arguments", ce.Argc, ce.Name, na)
			}
			// Create or reset function parameters in order to evaluate them on every call of the function
			for _, a := range fd.args {
				callContext.express(a, expression{&ast.BooleanExpr{Value: true}, false})
			}
			pc, err := e.estimate(fd.expr)
			if err != nil {
				return 0, err
			}
			return fc + ac + pc, nil
		} else {
			fc, ok = e.catalogue.FunctionCost(ce.Name)
			if !ok {
				return 0, errors.Errorf("EstimatorV1: no user function '%s' in scope", ce.Name)
			}
			ac, err := e.estimate(ce.Argv)
			if err != nil {
				return 0, err
			}
			return fc + ac, nil
		}

	case *ast.RefExpr:
		cc := e.contexts.curr()
		inner, ok := cc.expression(ce.Name)
		if !ok {
			return 0, errors.Errorf("no variable '%s' in context", ce.Name)
		}
		if !inner.evaluated {
			ic, err := e.estimate(inner.expr)
			if err != nil {
				return 0, err
			}
			cc := e.contexts.curr()
			cc.express(ce.Name, expression{inner.expr, true})
			return ic + 2, nil
		}
		return 2, nil

	case *ast.IfExpr:
		switch e.Version {
		case 2:
			cc, err := e.estimate(ce.Condition)
			if err != nil {
				return 0, err
			}
			tc, err := e.estimate(ce.True)
			if err != nil {
				return 0, err
			}
			fc, err := e.estimate(ce.False)
			if err != nil {
				return 0, err
			}
			if tc > fc {
				return tc + cc + 1, nil
			}
			return fc + cc + 1, nil
		default:
			cc, err := e.estimate(ce.Condition)
			if err != nil {
				return 0, err
			}
			tmp := e.contexts.copy()
			tc, err := e.estimate(ce.True)
			if err != nil {
				return 0, err
			}
			trueContext := e.contexts.copy()
			e.contexts = tmp
			fc, err := e.estimate(ce.False)
			if err != nil {
				return 0, err
			}
			if tc > fc {
				e.contexts = trueContext
				return tc + cc + 1, nil
			}
			return fc + cc + 1, nil
		}

	case *ast.GetterExpr:
		c, err := e.estimate(ce.Object)
		if err != nil {
			return 0, err
		}
		return c + 2, nil

	case *ast.FuncDeclaration:
		rc := e.contexts.root()
		for _, a := range ce.Args {
			rc.express(a, expression{&ast.BooleanExpr{Value: true}, false})
		}
		fc, err := e.estimate(ce.Body)
		if err != nil {
			return 0, err
		}
		ac := uint64(len(ce.Args) * 6)
		return ac + fc, nil

	default:
		return 0, nil
	}
}

type Catalogue struct {
	native map[int16]uint64
	user   map[string]uint64
}

func (c *Catalogue) FunctionCost(id string) (uint64, bool) {
	v, ok := c.user[id]
	return v, ok
}

func NewCatalogueV2() *Catalogue {
	c := &Catalogue{
		native: make(map[int16]uint64),
		user:   make(map[string]uint64),
	}

	c.user["0"] = 1
	c.user["1"] = 1
	c.user["2"] = 1
	c.user["100"] = 1
	c.user["101"] = 1
	c.user["102"] = 1
	c.user["103"] = 1
	c.user["104"] = 1
	c.user["105"] = 1
	c.user["106"] = 1
	c.user["107"] = 1
	c.user["200"] = 1
	c.user["201"] = 1
	c.user["202"] = 1
	c.user["203"] = 10
	c.user["300"] = 10
	c.user["303"] = 1
	c.user["304"] = 1
	c.user["305"] = 1
	c.user["400"] = 2
	c.user["401"] = 2
	c.user["410"] = 1
	c.user["411"] = 1
	c.user["412"] = 1
	c.user["420"] = 1
	c.user["421"] = 1
	c.user["500"] = 100
	c.user["501"] = 10
	c.user["502"] = 10
	c.user["503"] = 10
	c.user["600"] = 10
	c.user["601"] = 10
	c.user["602"] = 10
	c.user["603"] = 10
	c.user["1000"] = 100
	c.user["1001"] = 100
	c.user["1003"] = 100
	c.user["1040"] = 10
	c.user["1041"] = 10
	c.user["1042"] = 10
	c.user["1043"] = 10
	c.user["1050"] = 100
	c.user["1051"] = 100
	c.user["1052"] = 100
	c.user["1053"] = 100
	c.user["1060"] = 100

	c.user["throw"] = 2
	c.user["addressFromString"] = 124
	c.user["!="] = 26
	c.user["isDefined"] = 35
	c.user["extract"] = 13
	c.user["dropRightBytes"] = 19
	c.user["takeRightBytes"] = 19
	c.user["takeRight"] = 19
	c.user["dropRight"] = 19
	c.user["!"] = 11
	c.user["-"] = 9
	c.user["getInteger"] = 10
	c.user["getBoolean"] = 10
	c.user["getBinary"] = 10
	c.user["getString"] = 10
	c.user["addressFromPublicKey"] = 82
	c.user["wavesBalance"] = 109

	// Type constructors, type constructor cost equals to the number of it's parameters
	c.user["Address"] = 1
	c.user["Alias"] = 1
	c.user["DataEntry"] = 2
	c.user["DataTransaction"] = 9
	c.user["AssetPair"] = 2

	return c
}

func NewCatalogueV3() *Catalogue {
	c := NewCatalogueV2()

	// New native functions
	c.user["108"] = 100
	c.user["109"] = 100
	c.user["504"] = 300
	c.user["604"] = 10
	c.user["605"] = 10
	c.user["1004"] = 100
	c.user["1005"] = 100
	c.user["1006"] = 100
	c.user["700"] = 30
	c.user["1061"] = 10
	c.user["1070"] = 100
	c.user["1100"] = 2
	c.user["1200"] = 20
	c.user["1201"] = 10
	c.user["1202"] = 10
	c.user["1203"] = 20
	c.user["1204"] = 20
	c.user["1205"] = 100
	c.user["1206"] = 20
	c.user["1207"] = 20
	c.user["1208"] = 20

	// Cost updates for existing user functions
	c.user["throw"] = 1
	c.user["isDefined"] = 1
	c.user["!="] = 1
	c.user["!"] = 1
	c.user["-"] = 1

	// Constructors for simple types
	c.user["Ceiling"] = 0
	c.user["Floor"] = 0
	c.user["HalfEven"] = 0
	c.user["Down"] = 0
	c.user["Up"] = 0
	c.user["HalfUp"] = 0
	c.user["HalfDown"] = 0
	c.user["NoAlg"] = 0
	c.user["Md5"] = 0
	c.user["Sha1"] = 0
	c.user["Sha224"] = 0
	c.user["Sha256"] = 0
	c.user["Sha384"] = 0
	c.user["Sha512"] = 0
	c.user["Sha3224"] = 0
	c.user["Sha3256"] = 0
	c.user["Sha3384"] = 0
	c.user["Sha3512"] = 0
	c.user["Unit"] = 0

	// New user functions
	c.user["@extrNative(1040)"] = 10
	c.user["@extrNative(1041)"] = 10
	c.user["@extrNative(1042)"] = 10
	c.user["@extrNative(1043)"] = 10
	c.user["@extrNative(1050)"] = 100
	c.user["@extrNative(1051)"] = 100
	c.user["@extrNative(1052)"] = 100
	c.user["@extrNative(1053)"] = 100
	c.user["@extrUser(getInteger)"] = 10
	c.user["@extrUser(getBoolean)"] = 10
	c.user["@extrUser(getBinary)"] = 10
	c.user["@extrUser(getString)"] = 10
	c.user["@extrUser(addressFromString)"] = 124
	c.user["parseIntValue"] = 20
	c.user["value"] = 13
	c.user["valueOrErrorMessage"] = 13

	c.user["WriteSet"] = 1
	c.user["TransferSet"] = 1
	c.user["ScriptTransfer"] = 3
	c.user["ScriptResult"] = 2
	return c
}

func NewCatalogueV4() *Catalogue {
	c := NewCatalogueV3()
	c.user["IntegerEntry"] = 2
	c.user["BooleanEntry"] = 2
	c.user["BinaryEntry"] = 2
	c.user["StringEntry"] = 2
	c.user["DeleteEntry"] = 1
	c.user["Issue"] = 7
	c.user["Reissue"] = 3
	c.user["Burn"] = 2
	c.user["contains"] = 20
	c.user["valueOrElse"] = 13
	c.user["405"] = 10
	c.user["406"] = 3
	c.user["407"] = 3
	c.user["701"] = 30
	c.user["800"] = 3900
	c.user["900"] = 70
	c.user["1070"] = 5
	c.user["1080"] = 10
	c.user["1091"] = 1
	c.user["1100"] = 2
	c.user["1101"] = 3
	c.user["1102"] = 10
	c.user["1103"] = 5
	c.user["1104"] = 5
	c.user["2400"] = 1900
	c.user["2401"] = 2000
	c.user["2402"] = 2150
	c.user["2403"] = 2300
	c.user["2404"] = 2450
	c.user["2405"] = 2550
	c.user["2406"] = 2700
	c.user["2407"] = 2900
	c.user["2408"] = 3000
	c.user["2409"] = 3150
	c.user["2410"] = 3250
	c.user["2411"] = 3400
	c.user["2412"] = 3500
	c.user["2413"] = 3650
	c.user["2414"] = 3750
	c.user["2500"] = 100
	c.user["2501"] = 110
	c.user["2502"] = 125
	c.user["2503"] = 150
	c.user["2600"] = 100
	c.user["2601"] = 500
	c.user["2602"] = 550
	c.user["2603"] = 625
	c.user["2700"] = 10
	c.user["2701"] = 25
	c.user["2702"] = 50
	c.user["2703"] = 100
	c.user["2800"] = 10
	c.user["2801"] = 25
	c.user["2802"] = 50
	c.user["2803"] = 100
	c.user["2900"] = 10
	c.user["2901"] = 25
	c.user["2902"] = 50
	c.user["2903"] = 100
	return c
}
