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

func root(variables map[string]ast.Expr) *context {
	e := make(map[string]expression, len(variables))
	for k, v := range variables {
		e[k] = expression{expr: v, evaluated: true}
	}
	e["height"] = expression{expr: ast.NewLong(0), evaluated: true}
	e["tx"] = expression{expr: ast.NewObject(map[string]ast.Expr{}), evaluated: true}
	e["this"] = expression{expr: ast.NewObject(map[string]ast.Expr{}), evaluated: true}
	return &context{
		name:        "root",
		expressions: e,
		functions:   make(map[string]function),
	}
}

func (c *context) branch(name string) *context {
	return &context{
		name:        name,
		parent:      c,
		expressions: make(map[string]expression),
		functions:   make(map[string]function),
	}
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

type Estimator struct {
	version   int
	catalogue *Catalogue
	contexts  map[string]*context
	current   string
}

func NewEstimator(version int, catalogue *Catalogue, variables map[string]ast.Expr) *Estimator {
	rc := root(variables)
	cs := map[string]*context{rc.name: rc}
	return &Estimator{
		version:   version,
		catalogue: catalogue,
		contexts:  cs,
		current:   rc.name,
	}
}

func (e *Estimator) Estimate(script *ast.Script) (int64, error) {
	if script.IsDapp() {
		rc, ok := e.contexts["root"]
		if !ok {
			return 0, errors.New("estimation: no root context")
		}
		delete(rc.expressions, "tx")
		rc.expressions["height"] = expression{expr: ast.NewLong(0), evaluated: true}
		rc.expressions["this"] = expression{expr: ast.NewUnit(), evaluated: false}
		var declarationsCost int64 = 0
		for _, d := range script.DApp.Declarations {
			switch decl := d.(type) {
			case *ast.LetExpr:
				rc.express(decl.Name, expression{decl.Value, false})
				declarationsCost += 5
			case *ast.FuncDeclaration:
				for _, a := range decl.Args {
					rc.express(a, expression{&ast.BooleanExpr{Value: true}, false})
				}
				c := rc.branch(decl.Name)
				e.contexts[c.name] = c

				_, err := e.change(decl.Name)
				if err != nil {
					return 0, errors.Wrap(err, "estimation")
				}
				fc, err := e.estimate(decl.Body)
				if err != nil {
					return 0, errors.Wrap(err, "estimation")
				}
				ac := int64(len(decl.Args) * 5)
				e.catalogue.user[decl.Name] = ac + fc
				_, err = e.change("root")
				if err != nil {
					return 0, errors.Wrap(err, "estimation")
				}
				declarationsCost += 5
			}
		}
		var callableCost int64 = 0
		for _, cf := range script.DApp.CallableFuncs {
			cc, cn := e.copyContexts()
			c, err := e.estimateCallable(cf)
			if err != nil {
				return 0, errors.Wrap(err, "estimation")
			}
			e.restoreContexts(cc, cn)
			if c > callableCost {
				callableCost = c
			}
		}
		c, err := e.estimateCallable(script.DApp.Verifier)
		if err != nil {
			return 0, errors.Wrap(err, "estimation")
		}
		if c > callableCost {
			callableCost = c
		}
		return declarationsCost + callableCost, nil
	}
	verifierCost, err := e.estimate(script.Verifier)
	if err != nil {
		return 0, errors.Wrap(err, "estimation")
	}
	return verifierCost, nil
}

func (e *Estimator) context() (*context, error) {
	c, ok := e.contexts[e.current]
	if !ok {
		return nil, errors.Errorf("failed to get current context by name '%s'", e.current)
	}
	return c, nil
}

func (e *Estimator) change(name string) (*context, error) {
	c, ok := e.contexts[name]
	if !ok {
		return nil, errors.Errorf("failed to change context to context named '%s'", name)
	}
	e.current = name
	return c, nil
}

func (e *Estimator) copyContexts() (map[string]*context, string) {
	cp := make(map[string]*context)
	for k, v := range e.contexts {
		e := make(map[string]expression)
		for ke, ve := range v.expressions {
			e[ke] = ve
		}
		f := make(map[string]function)
		for kf, vf := range v.functions {
			f[kf] = vf
		}
		c := context{
			name:        k,
			parent:      nil,
			expressions: e,
			functions:   f,
		}
		cp[k] = &c
	}
	for k, v := range e.contexts {
		if v.parent != nil {
			p, ok := cp[v.parent.name]
			if ok {
				cp[k].parent = p
			}
		}
	}
	return cp, e.current
}

func (e *Estimator) restoreContexts(cp map[string]*context, current string) {
	e.contexts = cp
	e.current = current
}

func (e *Estimator) estimateCallable(callable *ast.DappCallableFunc) (int64, error) {
	if callable == nil {
		return 0, nil
	}
	rc, ok := e.contexts["root"]
	if !ok {
		return 0, errors.New("no root context")
	}
	rc.express(callable.AnnotationInvokeName, expression{&ast.BooleanExpr{Value: true}, false})
	cost, err := e.estimate(callable.FuncDecl)
	if err != nil {
		return 0, err
	}
	return cost + 10, nil
}

func (e *Estimator) estimate(expr ast.Expr) (int64, error) {
	switch ce := expr.(type) {
	case *ast.StringExpr, *ast.LongExpr, *ast.BooleanExpr, *ast.BytesExpr:
		return 1, nil

	case ast.Exprs:
		var total int64 = 0
		for _, item := range ce {
			c, err := e.estimate(item)
			if err != nil {
				return 0, err
			}
			total += c
		}
		return total, nil

	case *ast.Block:
		cc, err := e.context()
		if err != nil {
			return 0, err
		}
		cc.express(ce.Let.Name, expression{ce.Let.Value, false})
		bc, err := e.estimate(ce.Body)
		if err != nil {
			return 0, err
		}
		return bc + 5, nil

	case *ast.BlockV2:
		switch declaration := ce.Decl.(type) {
		case *ast.LetExpr:
			cc, err := e.context()
			if err != nil {
				return 0, err
			}
			cc.express(declaration.Name, expression{declaration.Value, false})
			bc, err := e.estimate(ce.Body)
			if err != nil {
				return 0, err
			}
			return bc + 5, nil
		case *ast.FuncDeclaration:
			switch e.version {
			case 2:
				cc, err := e.context()
				if err != nil {
					return 0, err
				}
				cc.declare(declaration.Name, function{declaration.Body, declaration.Args})
				nc := cc.branch(declaration.Name)
				e.contexts[nc.name] = nc
			default:
				rc, ok := e.contexts["root"]
				if !ok {
					return 0, errors.New("no root context")
				}
				for _, a := range declaration.Args {
					rc.express(a, expression{&ast.BooleanExpr{Value: true}, false})
				}
				fc, err := e.estimate(declaration.Body)
				if err != nil {
					return 0, err
				}
				ac := int64(len(declaration.Args) * 5) // arguments cost = 5 * number of arguments
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
		var fc int64
		callContext, err := e.context()
		if err != nil {
			return 0, err
		}
		if fd, ok := callContext.declaration(ce.Name); ok {
			// Estimate parameters that was passed to the function
			fc += int64(ce.Argc * 5)
			ac, err := e.estimate(ce.Argv)
			if err != nil {
				return 0, err
			}
			// Change context to the function's one
			functionContext, err := e.change(ce.Name)
			if err != nil {
				return 0, err
			}
			if na := len(fd.args); na != ce.Argc {
				return 0, errors.Errorf("unexpected number of arguments %d, function '%s' accepts %d arguments", ce.Argc, ce.Name, na)
			}
			// Create or reset function parameters in order to evaluate them on every call of the function
			for _, a := range fd.args {
				functionContext.express(a, expression{&ast.BooleanExpr{Value: true}, false})
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
		cc, err := e.context()
		if err != nil {
			return 0, err
		}
		inner, ok := cc.expression(ce.Name)
		if !ok {
			return 0, errors.Errorf("no variable '%s' in context", ce.Name)
		}
		if !inner.evaluated {
			ic, err := e.estimate(inner.expr)
			if err != nil {
				return 0, err
			}
			cc, err := e.context()
			if err != nil {
				return 0, err
			}
			cc.express(ce.Name, expression{inner.expr, true})
			return ic + 2, nil
		}
		return 2, nil

	case *ast.IfExpr:
		cc, err := e.estimate(ce.Condition)
		if err != nil {
			return 0, err
		}
		tmp, tmpCurr := e.copyContexts()
		tc, err := e.estimate(ce.True)
		if err != nil {
			return 0, err
		}
		trueContext, trueCurr := e.copyContexts()
		e.restoreContexts(tmp, tmpCurr)
		fc, err := e.estimate(ce.False)
		if err != nil {
			return 0, err
		}
		if tc > fc {
			e.restoreContexts(trueContext, trueCurr)
			return tc + cc + 1, nil
		}
		return fc + cc + 1, nil

	case *ast.GetterExpr:
		c, err := e.estimate(ce.Object)
		if err != nil {
			return 0, err
		}
		return c + 2, nil

	case *ast.FuncDeclaration:
		cc, err := e.context()
		if err != nil {
			return 0, err
		}
		for _, a := range ce.Args {
			cc.express(a, expression{&ast.BooleanExpr{Value: true}, false})
		}
		c, err := e.estimate(ce.Body)
		if err != nil {
			return 0, err
		}
		return c + int64(len(ce.Args)*6), nil

	default:
		return 0, nil
	}
}

type Catalogue struct {
	native map[int16]int64
	user   map[string]int64
}

func (c *Catalogue) FunctionCost(id string) (int64, bool) {
	v, ok := c.user[id]
	return v, ok
}

func NewCatalogueV2() *Catalogue {
	c := &Catalogue{
		native: make(map[int16]int64),
		user:   make(map[string]int64),
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
