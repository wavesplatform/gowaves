package compiler

import (
	"maps"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
)

const (
	charRange    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lenCharRange = 52
)

func idxToName(n int, seed string) string {
	if n < lenCharRange {
		return string(charRange[n]) + seed
	} else {
		return idxToName(n/lenCharRange-1, string(charRange[n%lenCharRange])+seed)
	}
}

type Compaction struct {
	tree          *ast.Tree
	counter       int
	originalNames map[string]string
	resAbrList    []string

	knownDecs []string
}

func NewCompaction(tree *ast.Tree) Compaction {
	return Compaction{
		tree:          tree,
		counter:       0,
		originalNames: make(map[string]string, 0),
		resAbrList:    []string{},
		knownDecs:     []string{},
	}
}

func (c *Compaction) Compact() {
	var delcs []ast.Node
	for _, d := range c.tree.Declarations {
		newD := c.processDecl(d)
		delcs = append(delcs, newD)
	}
	c.tree.Declarations = delcs
	funcs := []ast.Node{}
	for _, n := range c.tree.Functions {
		f := n.(*ast.FunctionDeclarationNode)
		invParName := c.replaceName(f.InvocationParameter)
		c.knownDecs = append(c.knownDecs, f.InvocationParameter)
		args := []string{}
		for _, a := range f.Arguments {
			name := c.replaceName(a)
			args = append(args, name)
			c.knownDecs = append(c.knownDecs, a)
		}
		body := c.processExpr(f.Body)
		newF := ast.NewFunctionDeclarationNode(f.Name, args, body, nil)
		newF.InvocationParameter = invParName
		funcs = append(funcs, newF)
		c.knownDecs = c.knownDecs[:len(c.knownDecs)-len(args)-1]
	}
	c.tree.Functions = funcs
	if c.tree.Verifier != nil {
		v := c.tree.Verifier.(*ast.FunctionDeclarationNode)
		invParName := c.replaceName(v.InvocationParameter)
		c.knownDecs = append(c.knownDecs, v.InvocationParameter)
		newV := c.processExpr(v)
		newVFunc := newV.(*ast.FunctionDeclarationNode)
		newVFunc.InvocationParameter = invParName
		c.tree.Verifier = newVFunc
		c.knownDecs = c.knownDecs[:len(c.knownDecs)-1]
	}
	c.tree.Meta.Abbreviations = meta.NewAbbreviations(c.resAbrList)
}

func (c *Compaction) replaceName(oldName string) string {
	if compName, ok := c.originalNames[oldName]; ok {
		return compName
	}
	compName := idxToName(c.counter, "")
	if c.hasConflict(compName) {
		c.counter += 1
		return c.replaceName(oldName)
	}
	c.originalNames[oldName] = compName
	c.counter += 1
	c.resAbrList = append(c.resAbrList, oldName)
	return compName
}

func (c *Compaction) contains(oldName string) bool {
	for _, n := range c.knownDecs {
		if n == oldName {
			return true
		}
	}
	return false
}

func (c *Compaction) getReplacedName(oldName string) string {
	if compName, ok := c.originalNames[oldName]; ok {
		return compName
	}
	return oldName
}

func (c *Compaction) hasConflict(compactName string) bool {
	for _, n := range c.tree.Functions {
		f := n.(*ast.FunctionDeclarationNode)
		if f.Name == compactName {
			return true
		}
	}
	return false
}

func (c *Compaction) processDecl(node ast.Node) ast.Node {
	switch expr := node.(type) {
	case *ast.FunctionDeclarationNode:
		fName := c.replaceName(expr.Name)
		args := []string{}
		for _, a := range expr.Arguments {
			name := c.replaceName(a)
			args = append(args, name)
			c.knownDecs = append(c.knownDecs, a)
		}
		body := c.processExpr(expr.Body)
		c.knownDecs = c.knownDecs[:len(c.knownDecs)-len(args)]
		c.knownDecs = append(c.knownDecs, expr.Name)
		return ast.NewFunctionDeclarationNode(fName, args, body, nil)
	case *ast.AssignmentNode:
		name := c.replaceName(expr.Name)
		e := c.processExpr(expr.Expression)
		c.knownDecs = append(c.knownDecs, expr.Name)
		res := ast.NewAssignmentNode(name, e, nil)
		res.NewBlock = expr.NewBlock
		return res
	default:
		return node
	}
}

func (c *Compaction) processExpr(node ast.Node) ast.Node {
	switch expr := node.(type) {
	case *ast.FunctionDeclarationNode:
		f := c.processDecl(expr)
		block := c.processExpr(expr.Block)
		c.knownDecs = c.knownDecs[:len(c.knownDecs)-1]
		f.SetBlock(block)
		return f
	case *ast.AssignmentNode:
		a := c.processDecl(expr)
		block := c.processExpr(expr.Block)
		a.SetBlock(block)
		c.knownDecs = c.knownDecs[:len(c.knownDecs)-1]
		return a
	case *ast.FunctionCallNode:
		var f ast.Function
		switch t := expr.Function.(type) {
		case ast.NativeFunction:
			f = expr.Function
		case ast.UserFunction:
			if c.contains(t.Name()) {
				name := c.getReplacedName(t.Name())
				f = ast.UserFunction(name)
			} else {
				f = expr.Function
			}
		}
		args := []ast.Node{}
		for _, a := range expr.Arguments {
			arg := c.processExpr(a)
			args = append(args, arg)
		}
		return ast.NewFunctionCallNode(f, args)
	case *ast.ReferenceNode:
		if c.contains(expr.Name) {
			name := c.getReplacedName(expr.Name)
			return ast.NewReferenceNode(name)
		}
		return expr
	case *ast.PropertyNode:
		obj := c.processExpr(expr.Object)
		return ast.NewPropertyNode(expr.Name, obj)
	case *ast.ConditionalNode:
		cond := c.processExpr(expr.Condition)
		trueExpr := c.processExpr(expr.TrueExpression)
		falseExpr := c.processExpr(expr.FalseExpression)
		return ast.NewConditionalNode(cond, trueExpr, falseExpr)
	default:
		return node
	}
}

func getUsedNames(node ast.Node, usedNames map[string]struct{}) {
	switch expr := node.(type) {
	case *ast.AssignmentNode:
		getUsedNames(expr.Expression, usedNames)
		getUsedNames(expr.Block, usedNames)
	case *ast.FunctionDeclarationNode:
		getUsedNames(expr.Body, usedNames)
		getUsedNames(expr.Block, usedNames)
	case *ast.FunctionCallNode:
		usedNames[expr.Function.Name()] = struct{}{}
		for _, a := range expr.Arguments {
			getUsedNames(a, usedNames)
		}
	case *ast.PropertyNode:
		getUsedNames(expr.Object, usedNames)
	case *ast.ConditionalNode:
		getUsedNames(expr.Condition, usedNames)
		getUsedNames(expr.TrueExpression, usedNames)
		getUsedNames(expr.FalseExpression, usedNames)
	case *ast.ReferenceNode:
		usedNames[expr.Name] = struct{}{}
	}
}

func diff(map1, map2 map[string]struct{}) map[string]struct{} {
	difference := make(map[string]struct{})

	for key, value := range map1 {
		if _, exists := map2[key]; !exists {
			difference[key] = value
		}
	}

	return difference
}

func getUsedNamesFromList(tree *ast.Tree, exprList []ast.Node, prevNames map[string]struct{}) map[string]struct{} {
	nextNames := make(map[string]struct{})
	for _, expr := range exprList {
		getUsedNames(expr, nextNames)
	}
	nextNames = diff(nextNames, prevNames)
	if len(nextNames) != 0 {
		nextExprList := []ast.Node{}
		for _, d := range tree.Declarations {
			switch e := d.(type) {
			case *ast.FunctionDeclarationNode:
				if _, ok := nextNames[e.Name]; ok {
					nextExprList = append(nextExprList, e.Body)
				}
			case *ast.AssignmentNode:
				if _, ok := nextNames[e.Name]; ok {
					nextExprList = append(nextExprList, e.Expression)
				}
			}
		}
		maps.Copy(prevNames, nextNames)
		return getUsedNamesFromList(tree, nextExprList, prevNames)
	} else {
		return prevNames
	}
}

func removeUnusedCode(tree *ast.Tree) {
	bodies := []ast.Node{}
	for _, n := range tree.Functions {
		f := n.(*ast.FunctionDeclarationNode)
		bodies = append(bodies, f.Body)
	}
	if tree.Verifier != nil {
		v := tree.Verifier.(*ast.FunctionDeclarationNode)
		bodies = append(bodies, v.Body)
	}
	usedNames := getUsedNamesFromList(tree, bodies, make(map[string]struct{}))
	newDecl := []ast.Node{}
	for _, d := range tree.Declarations {
		switch e := d.(type) {
		case *ast.FunctionDeclarationNode:
			if _, ok := usedNames[e.Name]; ok {
				newDecl = append(newDecl, d)
			}
		case *ast.AssignmentNode:
			if _, ok := usedNames[e.Name]; ok {
				newDecl = append(newDecl, d)
			}
		}
	}
	tree.Declarations = newDecl
}
