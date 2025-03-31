package compiler

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/ride/meta"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	s "github.com/wavesplatform/gowaves/pkg/ride/compiler/stdlib"
)

const (
	stdlibVersionDirectiveName = "STDLIB_VERSION"
	contentTypeDirectiveName   = "CONTENT_TYPE"
	scriptTypeDirectiveName    = "SCRIPT_TYPE"
	importDirectiveName        = "IMPORT"

	dappValueName       = "DAPP"
	expressionValueName = "EXPRESSION"

	accountValueName = "ACCOUNT"
	assetValueName   = "ASSET"
	libraryValueName = "LIBRARY"
)

type scriptType byte

const (
	accountScript scriptType = iota + 1
	assetScript
)

type astError struct {
	msg    string
	begin  textPosition
	end    textPosition
	prefix string
}

func newASTError(msg string, token token32, buffer []rune, prefix string) error {
	begin := int(token.begin)
	end := int(token.end)
	positions := []int{begin, end}
	translations := translatePositions(buffer, positions)
	return &astError{msg: msg, begin: translations[begin], end: translations[end], prefix: prefix}
}

func (e *astError) Error() string {
	return fmt.Sprintf("%s(%d:%d, %d:%d): %s", e.prefix, e.begin.line, e.begin.symbol, e.end.line, e.end.symbol, e.msg)
}

type importPath struct {
	path string
	node *node32
}

type astParser struct {
	node   *node32
	tree   *ast.Tree
	buffer []rune

	errorsList []error
	stack      *stack

	stdFuncs   s.FunctionsSignatures
	stdObjects s.ObjectsSignatures
	stdTypes   map[string]s.Type

	scriptType  scriptType
	importPaths []importPath
	isLibrary   bool
	fileName    string
}

func newASTParser(node *node32, buffer []rune) astParser {
	return astParser{
		node: node,
		tree: &ast.Tree{
			LibVersion:   ast.LibV6,
			ContentType:  ast.ContentTypeApplication,
			Declarations: []ast.Node{},
			Functions:    []ast.Node{},
			Meta: meta.DApp{
				Version:       2,
				Functions:     []meta.Function{},
				Abbreviations: meta.Abbreviations{},
			},
		},
		buffer:     buffer,
		errorsList: []error{},
		stack:      newStack(),
		scriptType: accountScript,
	}
}

func (p *astParser) parse() {
	switch p.node.pegRule {
	case ruleCode:
		p.ruleCodeHandler(p.node.up)
	}
}

func (p *astParser) addError(token token32, format string, args ...any) {
	p.errorsList = append(p.errorsList,
		newASTError(fmt.Sprintf(format, args...), token, p.buffer, p.fileName))
}

func (p *astParser) loadBuildInVarsToStackByVersion() {
	resVars := make(map[string]s.Variable)
	ver := int(p.tree.LibVersion)
	for i := 0; i < ver; i++ {
		for _, v := range s.Vars().Vars[i].Append {
			resVars[v.Name] = v
		}
		for _, v := range s.Vars().Vars[i].Remove {
			delete(resVars, v)
		}
	}
	for _, v := range resVars {
		p.stack.pushVariable(v)
	}
	if !p.tree.IsDApp() {
		txType := p.stdTypes["Transaction"].(s.UnionType)
		txType.AppendType(s.SimpleType{Type: "Order"})
		p.stack.pushVariable(s.Variable{
			Name: "tx",
			Type: txType,
		})
	}
	if p.tree.LibVersion >= ast.LibV4 && p.tree.LibVersion <= ast.CurrentMaxLibraryVersion() {
		if p.scriptType == assetScript {
			p.stack.pushVariable(s.Variable{
				Name: "this",
				Type: s.SimpleType{Type: "Asset"},
			})
		} else {
			p.stack.pushVariable(s.Variable{
				Name: "this",
				Type: s.SimpleType{Type: "Address"},
			})
		}
	}
}

func (p *astParser) ruleCodeHandler(node *node32) {
	switch node.pegRule {
	case ruleDAppRoot:
		p.ruleDAppRootHandler(node.up)
	case ruleScriptRoot:
		p.ruleScriptRootHandler(node.up)
	}
}

func (p *astParser) ruleDAppRootHandler(node *node32) {
	curNode := skipToNextRule(node)
	if isRule(curNode, ruleDirective) {
		curNode = p.parseDirectives(curNode)
	}
	if !p.isLibrary {
		p.stdFuncs = s.FuncsByVersion()[p.tree.LibVersion]
		p.stdObjects = s.ObjectsByVersion()[p.tree.LibVersion]
		p.stdTypes = s.DefaultTypes()[p.tree.LibVersion]
		p.loadBuildInVarsToStackByVersion()
	}
	p.loadImport()
	curNode = skipToNextRule(curNode)
	if isRule(curNode, ruleDeclaration) {
		curNode = p.parseDeclarations(curNode)
	}
	curNode = skipToNextRule(curNode)
	if !p.isLibrary {
		if isRule(curNode, ruleAnnotatedFunc) {
			p.parseAnnotatedFunc(curNode)
		}
	}
}

func (p *astParser) loadLib(lib *astParser) {
	p.tree.Declarations = append(p.tree.Declarations, lib.tree.Declarations...)
	p.errorsList = append(p.errorsList, lib.errorsList...)
}

func (p *astParser) loadImport() {
	for _, path := range p.importPaths {
		if _, err := os.Stat(path.path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				p.addError(path.node.token32, "File '%s' doesn't exist", path.path)
				continue
			}
		}

		buffer, err := os.ReadFile(path.path)
		if err != nil {
			p.addError(path.node.token32, "File '%s' not readable: %v", path.path, err)
			continue
		}
		rawP := Parser{Buffer: string(buffer)}
		err = rawP.Init()
		if err != nil {
			p.addError(path.node.token32, "Failed to parse file '%s': %v", path.path, err)
			continue
		}
		err = rawP.Parse()
		if err != nil {
			p.addError(path.node.token32, "Failed to parse file '%s': %v", path.path, err)
			continue
		}
		parser := astParser{
			node: rawP.AST(),
			tree: &ast.Tree{
				LibVersion:   p.tree.LibVersion,
				ContentType:  ast.ContentTypeApplication,
				Declarations: []ast.Node{},
				Functions:    []ast.Node{},
				Meta: meta.DApp{
					Version:       2,
					Functions:     []meta.Function{},
					Abbreviations: meta.Abbreviations{},
				},
			},
			buffer:     rawP.buffer,
			errorsList: []error{},
			stack:      p.stack,
			stdFuncs:   p.stdFuncs,
			stdObjects: p.stdObjects,
			stdTypes:   p.stdTypes,
			isLibrary:  true,
			fileName:   path.path,
		}
		parser.parse()
		p.loadLib(&parser)
	}
}

func (p *astParser) ruleScriptRootHandler(node *node32) {
	curNode := skipToNextRule(node)
	if isRule(curNode, ruleDirective) {
		curNode = p.parseDirectives(curNode)
	}
	if !p.isLibrary {
		p.stdFuncs = s.FuncsByVersion()[p.tree.LibVersion]
		p.stdObjects = s.ObjectsByVersion()[p.tree.LibVersion]
		p.stdTypes = s.DefaultTypes()[p.tree.LibVersion]
		p.loadBuildInVarsToStackByVersion()
	}
	p.loadImport()
	var decls []ast.Node
	for {
		curNode = skipToNextRule(curNode)
		if isRule(curNode, ruleDeclaration) {
			expr, _ := p.ruleDeclarationHandler(curNode.up, true)
			decls = append(decls, expr...)
			curNode = curNode.next
		}
		if isRule(curNode, ruleExpr) {
			break
		}
	}
	block, varType := p.ruleExprHandler(curNode)
	if block == nil {
		p.addError(curNode.token32, "No expression defined")
		return
	}
	if !s.BooleanType.Equal(varType) {
		p.addError(curNode.token32, "Script should return 'Boolean', but '%s' returned", varType)
		return
	}
	expr := block
	for i := len(decls) - 1; i >= 0; i-- {
		if sep, ok := decls[i].(*ast.ReferenceNode); ok {
			if sep.Name == "$strict" {
				i--
				cond := decls[i].(*ast.ConditionalNode)
				cond.TrueExpression = block
				block = cond
			}
		}
		expr = decls[i]
		expr.SetBlock(block)
		block = expr
	}
	p.tree.Verifier = block
	p.tree.Declarations = nil
	p.tree.Functions = nil
	p.tree.Meta.Functions = nil
	p.tree.Meta.Version = 0
}

func (p *astParser) parseDirectives(node *node32) *node32 {
	directiveCnt := make(map[string]int)
	curNode := node
	for {
		if isRule(curNode, ruleDirective) {
			p.ruleDirectiveHandler(curNode, directiveCnt)
			curNode = curNode.next
		}
		curNode = skipToNextRule(curNode)
		if curNode == nil || (curNode.pegRule != rule_ && curNode.pegRule != ruleDirective) {
			break
		}
	}
	return curNode
}

func (p *astParser) ruleDirectiveHandler(node *node32, directiveCnt map[string]int) {
	curNode := node.up
	// skip WS
	curNode = curNode.next
	// get Directive name
	dirNameNode := curNode
	dirName := p.nodeValue(curNode)
	curNode = curNode.next
	// skip WS
	curNode = curNode.next

	switch dirName {
	case stdlibVersionDirectiveName:
		if p.isLibrary {
			break
		}
		dirValue := p.nodeValue(curNode)
		version, err := strconv.ParseInt(dirValue, 10, 8)
		if err != nil {
			p.addError(curNode.token32, "Failed to parse version '%s': %v", dirValue, err)
			break
		}
		lv, err := ast.NewLibraryVersion(byte(version))
		if err != nil {
			p.addError(curNode.token32, "Invalid directive '%s': %v", stdlibVersionDirectiveName, err)
			lv = ast.LibV1
		}
		p.tree.LibVersion = lv
		p.checkDirectiveCnt(node, stdlibVersionDirectiveName, directiveCnt)

	case contentTypeDirectiveName:
		if p.isLibrary {
			break
		}
		dirValue := p.nodeValue(curNode)
		switch dirValue {
		case dappValueName:
			p.tree.ContentType = ast.ContentTypeApplication
		case expressionValueName:
			p.tree.ContentType = ast.ContentTypeExpression
		case libraryValueName:
			break
		default:
			p.addError(dirNameNode.token32, "Illegal value '%s' of directive '%s'", dirValue, contentTypeDirectiveName)
		}
		p.checkDirectiveCnt(node, contentTypeDirectiveName, directiveCnt)

	case scriptTypeDirectiveName:
		if p.isLibrary {
			break
		}
		dirValue := p.nodeValue(curNode)
		switch dirValue {
		case accountValueName:
			p.scriptType = accountScript
		case assetValueName:
			p.scriptType = assetScript
		default:
			p.addError(dirNameNode.token32, "Illegal value '%s' of directive '%s'", dirValue, scriptTypeDirectiveName)
		}
		p.checkDirectiveCnt(node, scriptTypeDirectiveName, directiveCnt)

	case importDirectiveName:
		if isRule(curNode, rulePaths) {
			curNode = curNode.up
			for curNode != nil {
				switch curNode.pegRule {
				case rulePathString:
					p.importPaths = append(p.importPaths, importPath{path: p.nodeValue(curNode), node: curNode})
					curNode = curNode.next
				case ruleWS:
					curNode = curNode.next
				}
			}
			p.checkDirectiveCnt(node, importDirectiveName, directiveCnt)
		}

	default:
		p.addError(dirNameNode.token32, "Illegal directive '%s'", dirName)
	}

}

func (p *astParser) nodeValue(node *node32) string {
	if node != nil {
		return string(p.buffer[node.begin:node.end])
	}
	return ""
}

func (p *astParser) checkDirectiveCnt(node *node32, name string, directiveCnt map[string]int) {
	if val, ok := directiveCnt[name]; ok && val == 1 {
		p.addError(node.token32, "Directive '%s' is used more than once", name)
	} else {
		directiveCnt[name] = 1
	}
}

func (p *astParser) parseDeclarations(node *node32) *node32 {
	curNode := node
	for {
		if isRule(curNode, ruleDeclaration) {
			expr, _ := p.ruleDeclarationHandler(curNode.up, false)
			p.tree.Declarations = append(p.tree.Declarations, expr...)
			curNode = curNode.next
		}
		curNode = skipToNextRule(curNode)
		if curNode == nil || (curNode.pegRule != rule_ && curNode.pegRule != ruleDeclaration) {
			break
		}
	}
	return curNode
}

func (p *astParser) ruleDeclarationHandler(node *node32, isBlock bool) ([]ast.Node, []s.Type) {
	switch node.pegRule {
	case ruleVariable:
		return p.ruleVariableHandler(node)
	case ruleFunc:
		expr, varType, _ := p.ruleFuncHandler(node)
		if expr == nil {
			return nil, nil
		}
		return []ast.Node{expr}, []s.Type{varType}
	case ruleStrictVariable:
		if !isBlock {
			p.addError(node.token32, "Invalid usage of 'strict' outside block or func")
			return nil, nil
		}
		return p.ruleStrictVariableHandler(node)
	default:
		panic(errors.Errorf("wrong type of rule in Declaration: %s", rul3s[node.pegRule]))
	}
}

func (p *astParser) ruleStrictVariableHandler(node *node32) ([]ast.Node, []s.Type) {
	exprs, varTypes := p.ruleVariableHandler(node)
	if exprs == nil {
		return nil, nil
	}
	decl := exprs[0].(*ast.AssignmentNode)
	cond := ast.NewConditionalNode(ast.NewFunctionCallNode(ast.NativeFunction("0"), []ast.Node{
		ast.NewReferenceNode(decl.Name),
		ast.NewReferenceNode(decl.Name),
	}), nil, ast.NewFunctionCallNode(ast.NativeFunction("2"), []ast.Node{ast.NewStringNode("Strict value is not equal to itself.")}),
	)
	sep := ast.NewReferenceNode("$strict")
	var afterStrict []ast.Node
	for i := len(exprs) - 1; i >= 1; i-- {
		afterStrict = append(afterStrict, exprs[i])
	}
	beforeStrict := []ast.Node{exprs[0]}
	beforeStrict = append(beforeStrict, []ast.Node{cond, sep}...)
	return append(beforeStrict, afterStrict...), varTypes
}

func (p *astParser) ruleVariableHandler(node *node32) ([]ast.Node, []s.Type) {
	curNode := skipToNextRule(node.up)
	switch curNode.pegRule {
	case ruleIdentifier:
		expr, varType := p.simpleVariableDeclaration(node)
		if expr == nil {
			return nil, nil
		}
		return []ast.Node{expr}, []s.Type{varType}
	case ruleTupleRef:
		return p.tupleRefDeclaration(node)
	default:
		return nil, nil
	}
}

func (p *astParser) simpleVariableDeclaration(node *node32) (ast.Node, s.Type) {
	curNode := skipToNextRule(node.up)
	// get Variable Name
	varName := p.nodeValue(curNode)
	curNode = skipToNextRule(curNode.next)
	expr, varType := p.ruleExprHandler(curNode)
	if expr == nil {
		return nil, nil
	}
	if _, ok := p.stack.variable(varName); ok {
		p.addError(curNode.token32, "Variable '%s' already declared", varName)
		return nil, nil
	}
	expr = ast.NewAssignmentNode(varName, expr, nil)
	p.stack.pushVariable(s.Variable{
		Name: varName,
		Type: varType,
	})
	return expr, varType
}

func (p *astParser) tupleRefDeclaration(node *node32) ([]ast.Node, []s.Type) {
	curNode := skipToNextRule(node.up)
	var varNames []string
	tupleRefNode := curNode.up
	for {
		tupleRefNode = skipToNextRule(tupleRefNode)
		if tupleRefNode != nil && tupleRefNode.pegRule == ruleIdentifier {
			name := p.nodeValue(tupleRefNode)
			if _, ok := p.stack.variable(name); ok {
				p.addError(tupleRefNode.token32, "Variable '%s' already declared", name)
				return nil, nil
			}
			varNames = append(varNames, name)
			tupleRefNode = tupleRefNode.next
		}
		if tupleRefNode == nil {
			break
		}
	}
	curNode = skipToNextRule(curNode.next)
	expr, varType := p.ruleExprHandler(curNode)
	if expr == nil {
		return nil, nil
	}
	tup, ok := varType.(s.TupleType)
	tupleLength := s.MaxTupleLength
	if !ok {
		if u, ok := varType.(s.UnionType); ok {
			isTuple := true
			for _, T := range u.Types {
				if tuple, ok := T.(s.TupleType); ok {
					if tupleLength > len(tuple.Types) {
						tupleLength = len(tuple.Types)
					}
				} else {
					isTuple = false
					break
				}
			}
			if !isTuple {
				p.addError(curNode.token32, "Expression should be 'Tuple' but '%s' declared", varType)
				return nil, nil
			}
		} else {
			p.addError(curNode.token32, "Expression should be 'Tuple' but '%s' declared", varType)
			return nil, nil
		}
	} else {
		tupleLength = len(tup.Types)
	}
	if tupleLength < len(varNames) {
		p.addError(node.token32, "Number of Identifiers should be less or equal than Tuple length")
		return nil, nil
	}
	var resExpr []ast.Node
	var resTypes []s.Type
	tupleName := "$t0" + strconv.FormatUint(uint64(node.begin), 10) + strconv.FormatUint(uint64(node.end), 10)
	p.stack.pushVariable(s.Variable{
		Name: tupleName,
		Type: varType,
	})
	resExpr = append(resExpr, &ast.AssignmentNode{
		Name:       tupleName,
		Expression: expr,
	})
	resTypes = append(resTypes, varType)
	for i, name := range varNames {
		resExpr = append(resExpr, &ast.AssignmentNode{
			Name: name,
			Expression: &ast.PropertyNode{
				Name:   "_" + strconv.Itoa(i+1),
				Object: &ast.ReferenceNode{Name: tupleName},
			},
		})
		itemType := getTupleItemTypeByIndex(varType, i)
		resTypes = append(resTypes, itemType)
		p.stack.pushVariable(s.Variable{
			Name: name,
			Type: itemType,
		})
	}
	return resExpr, resTypes
}

func getTupleItemTypeByIndex(t s.Type, i int) s.Type {
	if T, ok := t.(s.TupleType); ok {
		return T.Types[i]
	}
	u, ok := t.(s.UnionType)
	if !ok {
		return nil
	}
	resType := s.UnionType{Types: []s.Type{}}
	for _, T := range u.Types {
		tuple := T.(s.TupleType)
		resType.AppendType(tuple.Types[i])
	}
	return resType.Simplify()
}

func (p *astParser) ruleExprHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up.up
	expr, varType := p.ruleAndOpAtomHandler(curNode)
	if expr == nil {
		return nil, nil
	}
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	for {
		if !varType.Equal(s.BooleanType) {
			p.addError(node.up.up.token32, "Unexpected type, required 'Boolean', but '%s' found", varType.String())
		}
		curNode = skipToNextRule(curNode)
		curNode = skipToNextRule(curNode.next) // skip orOp
		nextExpr, nextExprVarType := p.ruleAndOpAtomHandler(curNode)

		varType = s.JoinTypes(varType, nextExprVarType)

		expr = ast.NewConditionalNode(expr, ast.NewBooleanNode(true), nextExpr)
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, varType
}

func (p *astParser) ruleAndOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleEqualityGroupOpAtomHandler(curNode)
	if expr == nil {
		return nil, nil
	}
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	for {
		if !varType.Equal(s.BooleanType) {
			p.addError(node.up.up.token32, "Unexpected type, required 'Boolean', but '%s' found", varType.String())
		}
		curNode = skipToNextRule(curNode)
		curNode = skipToNextRule(curNode.next) // skip andOp
		nextExpr, nextExprVarType := p.ruleEqualityGroupOpAtomHandler(curNode)

		varType = s.JoinTypes(varType, nextExprVarType)

		expr = ast.NewConditionalNode(expr, nextExpr, ast.NewBooleanNode(false))
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, varType
}

func (p *astParser) ruleEqualityGroupOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleCompareGroupOpAtomHandler(curNode)
	if expr == nil {
		return nil, nil
	}
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	for {
		curNode = skipToNextRule(curNode)
		var funcId ast.Function
		if isRule(curNode.up, ruleEqOp) {
			funcId = ast.NativeFunction("0")
		} else {
			funcId = ast.UserFunction("!=")
		}
		curNode = skipToNextRule(curNode.next)
		nextExpr, nextExprVarType := p.ruleCompareGroupOpAtomHandler(curNode)
		if nextExpr == nil {
			return nil, nil
		}
		if !nextExprVarType.EqualWithEntry(varType) && !varType.EqualWithEntry(nextExprVarType) {
			p.addError(curNode.token32, "Unexpected type, required '%s', but '%s' found", varType.String(), nextExprVarType.String())
		}
		expr = ast.NewFunctionCallNode(funcId, []ast.Node{expr, nextExpr})
		varType = s.BooleanType
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, s.BooleanType
}

func (p *astParser) ruleCompareGroupOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleListGroupOpAtomHandler(curNode)
	if expr == nil {
		return nil, nil
	}
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	if !s.BigIntType.Equal(varType) && !s.IntType.Equal(varType) {
		p.addError(node.up.up.token32, "Unexpected type, required 'BigInt' or 'Int', but '%s' found", varType.String())
	}
	for {
		curNode = skipToNextRule(curNode)
		operator := curNode.up.pegRule
		curNode = skipToNextRule(curNode.next)
		nextExpr, nextExprVarType := p.ruleListGroupOpAtomHandler(curNode)
		if nextExpr == nil {
			return nil, nil
		}
		var gltFun, gleFun string
		if s.BigIntType.Equal(varType) {
			if s.BigIntType.Equal(nextExprVarType) {
				gltFun = "319"
				gleFun = "320"
			} else {
				p.addError(curNode.token32, "Unexpected type, required 'BigInt', but '%s' found", nextExprVarType.String())
			}
		} else if s.IntType.Equal(varType) {
			if s.IntType.Equal(nextExprVarType) {
				gltFun = "102"
				gleFun = "103"
			} else {
				p.addError(curNode.token32, "Unexpected type, required 'Int', but '%s' found", nextExprVarType.String())
			}
		}
		switch operator {
		case ruleGtOp:
			expr = ast.NewFunctionCallNode(ast.NativeFunction(gltFun), []ast.Node{expr, nextExpr})
		case ruleGeOp:
			expr = ast.NewFunctionCallNode(ast.NativeFunction(gleFun), []ast.Node{expr, nextExpr})
		case ruleLtOp:
			expr = ast.NewFunctionCallNode(ast.NativeFunction(gltFun), []ast.Node{nextExpr, expr})
		case ruleLeOp:
			expr = ast.NewFunctionCallNode(ast.NativeFunction(gleFun), []ast.Node{nextExpr, expr})
		}
		varType = s.BooleanType
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, s.BooleanType
}

func (p *astParser) ruleSumGroupOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleMultGroupOpAtomHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	for {
		curNode = skipToNextRule(curNode)
		operator := curNode.up.pegRule
		curNode = skipToNextRule(curNode.next)
		nextExpr, nextExprVarType := p.ruleMultGroupOpAtomHandler(curNode)
		if nextExpr == nil {
			return nil, nil
		}
		var funcId string
		switch operator {
		case ruleSumOp:
			if varType.Equal(s.IntType) && nextExprVarType.Equal(s.IntType) {
				funcId = "100"
			} else if varType.Equal(s.BigIntType) && nextExprVarType.Equal(s.BigIntType) {
				funcId = "311"
			} else if varType.Equal(s.StringType) && nextExprVarType.Equal(s.StringType) {
				funcId = "300"
			} else if varType.Equal(s.ByteVectorType) && nextExprVarType.Equal(s.ByteVectorType) {
				funcId = "203"
			} else {
				p.addError(node.token32, "Unexpected types for '+' operator '%s' and '%s'", varType.String(), nextExprVarType.String())
			}
		case ruleSubOp:
			if varType.Equal(s.IntType) && nextExprVarType.Equal(s.IntType) {
				funcId = "101"
			} else if varType.Equal(s.BigIntType) && nextExprVarType.Equal(s.BigIntType) {
				funcId = "312"
			} else {
				p.addError(node.token32, "Unexpected types for '-' operator '%s' and '%s'", varType.String(), nextExprVarType.String())
			}
		}
		expr = ast.NewFunctionCallNode(ast.NativeFunction(funcId), []ast.Node{expr, nextExpr})
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, varType
}

func (p *astParser) ruleListGroupOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleSumGroupOpAtomHandler(curNode)
	if expr == nil {
		return nil, nil
	}
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	var resExprType s.Type
	var resListType s.ListType
	if t, ok := varType.(s.ListType); ok {
		resListType = t
	} else {
		resListType = s.ListType{Type: varType}
	}
	for {
		curNode = skipToNextRule(curNode)
		operator := curNode.up.pegRule
		curNode = skipToNextRule(curNode.next)
		nextExpr, nextVarType := p.ruleSumGroupOpAtomHandler(curNode)
		if nextExpr == nil {
			return nil, nil
		}
		var funcId ast.Function
		switch operator {
		case ruleConsOp:
			if _, ok := nextVarType.(s.ListType); !ok {
				p.addError(curNode.token32, "Unexpected types for '::' operator '%s' and '%s'", varType, nextVarType)
				return nil, nil
			}
			funcId = ast.NativeFunction("1100")
			resListType.AppendList(nextVarType)
			resExprType = resListType
		case ruleAppendOp:
			if _, ok := varType.(s.ListType); !ok {
				p.addError(curNode.token32, "Unexpected types for ':+' operator '%s' and '%s'", varType, nextVarType)
				return nil, nil
			} else {
				funcId = ast.NativeFunction("1101")
				if l, ok := varType.(s.ListType); ok {
					l.AppendType(nextVarType)
					resExprType = l
				} else if u, ok := varType.(s.UnionType); ok {
					resType := s.UnionType{Types: []s.Type{}}
					for _, uT := range u.Types {
						if ul, okL := uT.(s.ListType); okL {
							ul.AppendType(nextVarType)
							resType.AppendType(ul)
						}
					}
					resExprType = resType.Simplify()
				} else {
					p.addError(curNode.token32, "Unexpected types for ':+' operator '%s' and '%s'", varType, nextVarType)
					return nil, nil
				}
			}
		case ruleConcatOp:
			funcId = ast.NativeFunction("1102")
			if l1, ok := varType.(s.ListType); ok {
				if l2, ok := nextVarType.(s.ListType); ok {
					l1.AppendList(l2)
					resExprType = l1
				} else if u, ok := nextVarType.(s.UnionType); ok {
					resType := s.UnionType{Types: []s.Type{}}
					for _, uT := range u.Types {
						if ul, okL := uT.(s.ListType); okL {
							ul.AppendList(varType)
							resType.AppendType(ul)
						}
					}
					resExprType = resType.Simplify()
				} else {
					p.addError(curNode.token32, "Unexpected types for '++' operator '%s' snd '%s'", varType, nextVarType)
					return nil, nil
				}
			} else if u1, ok := varType.(s.UnionType); ok {
				if l, ok := nextVarType.(s.ListType); ok {
					resType := s.UnionType{Types: []s.Type{}}
					for _, uT := range u1.Types {
						if ul, okL := uT.(s.ListType); okL {
							ul.AppendList(l)
							resType.AppendType(ul)
						}
					}
					resExprType = resType.Simplify()
				} else if u2, ok := nextVarType.(s.UnionType); ok {
					resType := s.UnionType{Types: []s.Type{}}
					for _, uT1 := range u1.Types {
						if ul1, okL1 := uT1.(s.ListType); okL1 {
							for _, uT2 := range u2.Types {
								if ul2, okL2 := uT2.(s.ListType); okL2 {
									ul1.AppendList(ul2)
									resType.AppendType(ul1)
								}
							}
						}
					}
					resExprType = resType.Simplify()
				} else {
					p.addError(curNode.token32, "Unexpected types for '++' operator '%s' and '%s'", varType, nextVarType)
					return nil, nil
				}
			} else {
				p.addError(curNode.token32, "Unexpected types for '++' operator '%s' and '%s'", varType, nextVarType)
				return nil, nil
			}
		default:
			panic("unhandled default case")
		}
		expr = ast.NewFunctionCallNode(funcId, []ast.Node{expr, nextExpr})
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, resExprType
}

func (p *astParser) ruleMultGroupOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleAtomExprHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	for {
		curNode = skipToNextRule(curNode)
		operator := curNode.up.pegRule
		curNode = skipToNextRule(curNode.next)
		nextExpr, nextExprVarType := p.ruleAtomExprHandler(curNode)
		var funcId string
		switch operator {
		case ruleMulOp:
			if varType.Equal(s.IntType) && nextExprVarType.Equal(s.IntType) {
				funcId = "104"
			} else if varType.Equal(s.BigIntType) && nextExprVarType.Equal(s.BigIntType) {
				funcId = "313"
			} else {
				p.addError(node.token32, "Unexpected types for '*' operator '%s' and '%s'", varType.String(), nextExprVarType.String())
			}
		case ruleDivOp:
			if varType.Equal(s.IntType) && nextExprVarType.Equal(s.IntType) {
				funcId = "105"
			} else if varType.Equal(s.BigIntType) && nextExprVarType.Equal(s.BigIntType) {
				funcId = "314"
			} else {
				p.addError(node.token32, "Unexpected types for '/' operator '%s' and '%s'", varType.String(), nextExprVarType.String())
			}
		case ruleModOp:
			if varType.Equal(s.IntType) && nextExprVarType.Equal(s.IntType) {
				funcId = "106"
			} else if varType.Equal(s.BigIntType) && nextExprVarType.Equal(s.BigIntType) {
				funcId = "315"
			} else {
				p.addError(node.token32, "Unexpected types for '%%' operator '%s' and '%s'", varType.String(), nextExprVarType.String())
			}
		}
		expr = ast.NewFunctionCallNode(ast.NativeFunction(funcId), []ast.Node{expr, nextExpr})
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, varType
}

func (p *astParser) ruleAtomExprHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	var unaryOp pegRule
	if isRule(curNode, ruleUnaryOp) {
		unaryOp = curNode.up.pegRule
		curNode = curNode.next
	}
	var expr ast.Node
	var varType s.Type
	switch curNode.pegRule {
	case ruleFoldMacro:
		expr, varType = p.ruleFoldMacroHandler(curNode)
	case ruleGettableExpr:
		expr, varType = p.ruleGettableExprHandler(curNode)
	case ruleIfWithError:
		expr, varType = p.ruleIfWithErrorHandler(curNode)
	case ruleMatch:
		expr, varType = p.ruleMatchHandler(curNode)
	}
	if expr == nil {
		return nil, nil
	}
	switch unaryOp {
	case ruleNegativeOp:
		if varType.Equal(s.IntType) {
			if i, ok := expr.(*ast.LongNode); ok {
				expr = ast.NewLongNode(-i.Value)
			} else {
				expr = ast.NewFunctionCallNode(ast.UserFunction("-"), []ast.Node{expr})
			}
		} else if varType.Equal(s.BigIntType) {
			expr = ast.NewFunctionCallNode(ast.NativeFunction("318"), []ast.Node{expr})
		} else {
			p.addError(curNode.token32, "Unexpected types for unary '-' operator, required 'Int' or 'BigInt', but '%s' found", varType.String())
		}
	case ruleNotOp:
		if varType.Equal(s.BooleanType) {
			expr = ast.NewFunctionCallNode(ast.UserFunction("!"), []ast.Node{expr})
		} else {
			p.addError(curNode.token32, "Unexpected types for unary '!' operator, required 'Boolean', but '%s' found", varType.String())
		}
	case rulePositiveOp:
		if !varType.Equal(s.IntType) && !varType.Equal(s.BigIntType) {
			p.addError(curNode.token32, "Unexpected types for unary '+' operator, required 'Int' or 'BigInt', but %s found", varType.String())
		}
	}
	return expr, varType
}

func (p *astParser) ruleConstHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	var expr ast.Node
	var varType s.Type
	switch curNode.pegRule {
	case ruleInteger:
		expr, varType = p.ruleIntegerHandler(curNode)
	case ruleString:
		expr, varType = p.ruleStringHandler(curNode)
	case ruleByteVector:
		expr, varType = p.ruleByteVectorHandler(curNode)
	case ruleBoolean:
		expr, varType = p.ruleBooleanAtomHandler(curNode)
	case ruleList:
		expr, varType = p.ruleListHandler(curNode)
	case ruleTuple:
		expr, varType = p.ruleTupleHandler(curNode)
	}
	return expr, varType
}

func (p *astParser) ruleIntegerHandler(node *node32) (ast.Node, s.Type) {
	value := p.nodeValue(node)
	if strings.Contains(value, "_") {
		value = strings.ReplaceAll(value, "_", "")
	}
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		p.addError(node.token32, "Failed to parse 'Integer' value: %v", err)
	}
	return ast.NewLongNode(number), s.IntType
}

func (p *astParser) ruleStringHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	var res string
	for curNode != nil {
		switch curNode.pegRule {
		case ruleChar:
			res += p.nodeValue(curNode)
		case ruleEscapedChar:
			escapedChar := p.nodeValue(curNode)
			switch escapedChar {
			case "\\b":
				res += "\b"
			case "\\f":
				res += "\f"
			case "\\n":
				res += "\n"
			case "\\r":
				res += "\r"
			case "\\t":
				res += "\t"
			case "\\\\":
				res += "\\"
			case "\\\"":
				res += "\""
			default:
				p.addError(curNode.token32,
					"Unknown escaped symbol: '%s'. The valid are \\b, \\f, \\n, \\r, \\t, \\\\, \\\"", escapedChar)
			}
		case ruleUnicodeChar:
			unicodeChar := p.nodeValue(curNode)
			char, err := strconv.Unquote(`"` + unicodeChar + `"`)
			if err != nil {
				p.addError(curNode.token32, "Unknown UTF-8 symbol '\\u%s'", unicodeChar)
			} else {
				res += char
			}
		}
		curNode = curNode.next
	}
	return ast.NewStringNode(res), s.StringType
}

func (p *astParser) ruleByteVectorHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	var err error
	var value []byte
	valueWithBase := p.nodeValue(curNode)
	// get value from baseXX'VALUE'
	valueInBase := valueWithBase[len("baseXX'") : len(valueWithBase)-1]
	if len(valueInBase) == 0 {
		return ast.NewBytesNode([]byte{}), s.ByteVectorType
	}
	switch node.up.pegRule {
	case ruleBase16:
		value, err = hex.DecodeString(valueInBase)
	case ruleBase58:
		value, err = base58.Decode(valueInBase)
	case ruleBase64:
		value, err = base64.StdEncoding.DecodeString(valueInBase)
	}
	if err != nil {
		p.addError(node.token32, "Failed to parse 'ByteVector' value: %v", err)
	}
	return ast.NewBytesNode(value), s.ByteVectorType
}

func (p *astParser) ruleListHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if curNode == nil {
		return ast.NewReferenceNode("nil"), s.ListType{}
	}
	curNode = skipToNextRule(curNode)
	if curNode == nil {
		return ast.NewReferenceNode("nil"), s.ListType{}
	}
	return p.ruleListExprSeqHandler(curNode)
}

func (p *astParser) ruleListExprSeqHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	elem, varType := p.ruleExprHandler(curNode)
	listType := s.ListType{Type: varType}
	curNode = curNode.next
	if curNode == nil {
		return ast.NewFunctionCallNode(ast.NativeFunction("1100"), []ast.Node{elem, ast.NewReferenceNode("nil")}), listType
	}
	curNode = skipToNextRule(curNode)
	secondElem, varType := p.ruleListExprSeqHandler(curNode)
	listType.AppendList(varType)
	return ast.NewFunctionCallNode(ast.NativeFunction("1100"), []ast.Node{elem, secondElem}), listType
}

func (p *astParser) ruleBooleanAtomHandler(node *node32) (ast.Node, s.Type) {
	value := p.nodeValue(node)
	var boolValue bool
	switch value {
	case "true":
		boolValue = true
	case "false":
		boolValue = false
	}
	return ast.NewBooleanNode(boolValue), s.BooleanType
}

func (p *astParser) ruleTupleHandler(node *node32) (ast.Node, s.Type) {
	curNode := skipToNextRule(node.up)
	var exprs []ast.Node
	var types []s.Type
	for {
		curNode = skipToNextRule(curNode)
		if isRule(curNode, ruleExpr) {
			expr, varType := p.ruleExprHandler(curNode)
			exprs = append(exprs, expr)
			types = append(types, varType)
			curNode = curNode.next
		}
		if curNode == nil {
			break
		}
	}
	if len(exprs) < 2 || len(exprs) > 22 {
		p.addError(node.token32, "Invalid Tuple length %d (allowed 2 to 22)", len(exprs))
		return nil, nil
	}
	return ast.NewFunctionCallNode(ast.NativeFunction(strconv.Itoa(1300+len(exprs)-2)), exprs), s.TupleType{Types: types}
}

func (p *astParser) ruleIfWithErrorHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if isRule(curNode, ruleFailedIfWithoutElse) {
		p.addError(curNode.token32, "If without else")
		return nil, nil
	}
	curNode = skipToNextRule(curNode.up)
	cond, condType := p.ruleExprHandler(curNode)
	if condType != s.BooleanType {
		p.addError(curNode.token32, "Expression must be 'Boolean' but got '%s'", condType)
	}
	curNode = skipToNextRule(curNode.next)
	var thenExpr ast.Node
	var thenType s.Type
	switch curNode.pegRule {
	case ruleExpr:
		thenExpr, thenType = p.ruleExprHandler(curNode)
	case ruleBlockWithoutPar:
		thenExpr, thenType = p.ruleBlockHandler(curNode)
	}
	curNode = skipToNextRule(curNode.next)
	var elseExpr ast.Node
	var elseType s.Type
	switch curNode.pegRule {
	case ruleExpr:
		elseExpr, elseType = p.ruleExprHandler(curNode)
	case ruleBlockWithoutPar:
		elseExpr, elseType = p.ruleBlockHandler(curNode)
	}
	var resType s.Type
	if thenType == nil || elseType == nil {
		return nil, nil
	}
	resType = s.JoinTypes(thenType, elseType)
	return ast.NewConditionalNode(cond, thenExpr, elseExpr), resType
}

func (p *astParser) ruleGettableExprHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	var expr ast.Node
	var varType s.Type
	switch curNode.pegRule {
	case ruleParExpr:
		expr, varType = p.ruleParExprHandler(curNode)
	case ruleBlock:
		expr, varType = p.ruleBlockHandler(curNode)
	case ruleFunctionCall:
		expr, varType = p.ruleFunctionCallHandler(curNode, nil, nil)
	case ruleIdentifier:
		expr, varType = p.ruleIdentifierHandler(curNode)
	case ruleConst:
		expr, varType = p.ruleConstHandler(curNode)
	}
	curNode = curNode.next
	for curNode != nil {
		curNode = skipToNextRule(curNode)
		switch curNode.pegRule {
		case ruleAsType:
			asNode := skipToNextRule(curNode.up)
			asPegType := asNode.pegRule
			asNode = skipToNextRule(asNode.next)
			t := p.ruleTypesHandler(asNode)
			if t == nil {
				return nil, nil
			}
			var falseExpr ast.Node
			switch asPegType {
			case ruleAsString:
				falseExpr = ast.NewReferenceNode("unit")
				varType = s.UnionType{Types: []s.Type{t, s.SimpleType{Type: "Unit"}}}
			case ruleExactAsString:
				if p.tree.LibVersion >= ast.LibV6 {
					falseExpr = ast.NewFunctionCallNode(
						ast.NativeFunction("2"),
						[]ast.Node{
							ast.NewFunctionCallNode(
								ast.NativeFunction("300"),
								[]ast.Node{
									ast.NewFunctionCallNode(
										ast.NativeFunction("3"),
										[]ast.Node{
											ast.NewReferenceNode("@"),
										},
									),
									ast.NewStringNode(fmt.Sprintf(" couldn't be cast to %s", t.String())),
								},
							),
						},
					)
				} else {
					falseExpr = ast.NewFunctionCallNode(
						ast.NativeFunction("2"),
						[]ast.Node{
							ast.NewStringNode(fmt.Sprintf("Couldn't cast %s to %s", varType.String(), t.String())),
						},
					)
				}
			}
			varType = t
			newExpr := ast.NewAssignmentNode(
				"@",
				expr,
				ast.NewConditionalNode(
					ast.NewFunctionCallNode(
						ast.NativeFunction("1"),
						[]ast.Node{
							ast.NewReferenceNode("@"),
							ast.NewStringNode(t.String()),
						},
					),
					ast.NewReferenceNode("@"),
					falseExpr,
				),
			)
			newExpr.NewBlock = true
			expr = newExpr
		case ruleListAccess:
			listNode := curNode.up
			if l, ok := varType.(s.ListType); !ok && varType != nil {
				p.addError(listNode.token32, "Type must be 'List' but got '%s'", varType.String())
			} else {
				listNode = skipToNextRule(listNode)
				var index ast.Node
				var indexType s.Type
				switch listNode.pegRule {
				case ruleExpr:
					index, indexType = p.ruleExprHandler(listNode)
				case ruleIdentifier:
					index, indexType = p.ruleIdentifierHandler(listNode)
				}
				if !indexType.Equal(s.IntType) {
					p.addError(listNode.token32, "Index type must be 'Int' but got '%s'", indexType.String())
				}
				expr = ast.NewFunctionCallNode(ast.NativeFunction("401"), []ast.Node{expr, index})
				if l.Type == nil {
					varType = l
				} else {
					varType = l.Type
				}
			}
		case ruleFunctionCallAccess:
			newExpr, newExprType := p.ruleFunctionCallHandler(curNode.up, expr, varType)
			if newExpr == nil {
				return expr, varType
			}
			expr, varType = newExpr, newExprType
		case ruleIdentifierAccess:
			newExpr, newExprType := p.ruleIdentifierAccessHandler(curNode.up, expr, varType)
			if newExpr == nil {
				return expr, varType
			}
			expr, varType = newExpr, newExprType
		case ruleTupleAccess:
			minLen := 0
			isTuple := true
			if u, okU := varType.(s.UnionType); okU {
				for _, unionType := range u.Types {
					if tuple, okT := unionType.(s.TupleType); okT {
						if minLen == 0 {
							minLen = len(tuple.Types)
						} else if len(tuple.Types) < minLen {
							minLen = len(tuple.Types)
						}
					} else {
						isTuple = false
						break
					}
				}
			} else if t, okT := varType.(s.TupleType); okT {
				minLen = len(t.Types)
			}

			if !isTuple {
				p.addError(curNode.token32, "Type must be 'Tuple' but got '%s'", varType.String())
				break
			}
			tupleIndexStr := p.nodeValue(curNode)
			indexStr := strings.TrimPrefix(tupleIndexStr, "_")
			index, err := strconv.ParseInt(indexStr, 10, 64)
			if err != nil {
				p.addError(curNode.token32, "Failed to parse tuple index: %v", err)
				return nil, nil
			}
			if index < 1 || index > int64(minLen) {
				p.addError(curNode.token32, "Tuple index must be less then %d", minLen)
				return nil, nil
			}
			expr = ast.NewPropertyNode(tupleIndexStr, expr)
			if u, okU := varType.(s.UnionType); okU {
				resType := s.UnionType{Types: []s.Type{}}
				for _, unionType := range u.Types {
					resType.AppendType(unionType.(s.TupleType).Types[index-1])
				}
				varType = resType.Simplify()
			} else if t, okT := varType.(s.TupleType); okT {
				varType = t.Types[index-1]
			}
		}
		curNode = curNode.next
	}
	return expr, varType
}

func (p *astParser) ruleIdentifierHandler(node *node32) (ast.Node, s.Type) {
	name := p.nodeValue(node)
	v, ok := p.stack.variable(name)
	if !ok {
		p.addError(node.token32, "Variable '%s' doesn't exist", name)
		return nil, nil
	}
	return ast.NewReferenceNode(name), v.Type
}

func (p *astParser) ruleParExprHandler(node *node32) (ast.Node, s.Type) {
	curNode := skipToNextRule(node.up)
	return p.ruleExprHandler(curNode)
}

func listArgsToString(l []s.Type) string {
	res := ""
	for i, t := range l {
		if t == nil {
			res += "Unknown"
		} else {
			res += t.String()
		}
		if i < len(l)-1 {
			res += ", "
		}
	}
	return res
}

func (p *astParser) ruleFunctionCallHandler(node *node32, firstArg ast.Node, firstArgType s.Type) (ast.Node, s.Type) {
	curNode := node.up
	funcName := p.nodeValue(curNode)
	nameNode := curNode
	curNode = skipToNextRule(curNode.next)
	argsNodes, argsTypes, astNodes := p.ruleArgSeqHandler(curNode)

	if firstArg != nil {
		argsNodes = append([]ast.Node{firstArg}, argsNodes...)
		argsTypes = append([]s.Type{firstArgType}, argsTypes...)
	}
	var funcSign s.FunctionParams
	funcSign, ok := p.stack.function(funcName)
	if !ok {
		funcSign, ok = p.stdFuncs.Get(funcName, argsTypes)
		if !ok {
			funcSign, ok = p.stdObjects.GetConstruct(funcName, argsTypes)
			if !ok {
				p.addError(nameNode.token32, "Undefined function '%s(%s)'", funcName, listArgsToString(argsTypes))
				return nil, nil
			}
		}
		if argsNodes == nil {
			argsNodes = []ast.Node{}
		}
		return ast.NewFunctionCallNode(funcSign.ID, argsNodes), funcSign.ReturnType
	}
	if len(argsNodes) != len(funcSign.Arguments) {
		p.addError(curNode.token32, "Function '%s' requires %d arguments, but %d are provided", funcName, len(funcSign.Arguments), len(argsNodes))
		return nil, funcSign.ReturnType
	}
	for i := range argsNodes {
		if funcSign.Arguments[i].EqualWithEntry(argsTypes[i]) {
			continue
		}
		p.addError(astNodes[i].token32, "Cannot use type '%s' as type '%s'", argsTypes[i], funcSign.Arguments[i])
	}
	if argsNodes == nil {
		argsNodes = []ast.Node{}
	}
	return ast.NewFunctionCallNode(funcSign.ID, argsNodes), funcSign.ReturnType
}

func (p *astParser) ruleArgSeqHandler(node *node32) ([]ast.Node, []s.Type, []*node32) {
	if node == nil || node.pegRule != ruleExprSeq {
		return nil, nil, nil
	}
	curNode := node.up
	var resultNodes []ast.Node
	var resultTypes []s.Type
	var resultAstNodes []*node32

	for {
		expr, varType := p.ruleExprHandler(curNode)
		resultNodes = append(resultNodes, expr)
		resultTypes = append(resultTypes, varType)
		resultAstNodes = append(resultAstNodes, curNode)
		curNode = curNode.next
		if curNode == nil {
			break
		}
		curNode = skipToNextRule(curNode)
		curNode = curNode.up
	}
	return resultNodes, resultTypes, resultAstNodes
}

func (p *astParser) ruleIdentifierAccessHandler(node *node32, obj ast.Node, objType s.Type) (ast.Node, s.Type) {
	curNode := node
	fieldName := p.nodeValue(curNode)

	fieldType, ok := p.stdObjects.GetField(objType, fieldName)
	if !ok {
		p.addError(curNode.token32, "Type '%s' has not filed '%s'", objType.String(), fieldName)
		return nil, nil
	}
	return ast.NewPropertyNode(fieldName, obj), fieldType

}

func (p *astParser) ruleBlockHandler(node *node32) (ast.Node, s.Type) {
	p.stack.addFrame()
	curNode := node.up
	var decls []ast.Node
	for {
		curNode = skipToNextRule(curNode)
		if isRule(curNode, ruleDeclaration) {
			expr, _ := p.ruleDeclarationHandler(curNode.up, true)
			decls = append(decls, expr...)
			curNode = curNode.next
		}
		if isRule(curNode, ruleExpr) {
			break
		}
	}
	block, varType := p.ruleExprHandler(curNode)
	expr := block
	for i := len(decls) - 1; i >= 0; i-- {
		if sep, ok := decls[i].(*ast.ReferenceNode); ok {
			if sep.Name == "$strict" {
				i--
				cond := decls[i].(*ast.ConditionalNode)
				cond.TrueExpression = block
				block = cond
			}
		} else {
			expr = decls[i]
			expr.SetBlock(block)
			block = expr
		}
	}
	p.stack.dropFrame()
	return expr, varType
}

func (p *astParser) ruleFuncHandler(node *node32) (ast.Node, s.Type, []s.Type) {
	p.stack.addFrame()
	curNode := skipToNextRule(node.up)
	funcName := p.nodeValue(curNode)
	if _, ok := p.stack.function(funcName); ok {
		p.addError(curNode.token32, "Function '%s' already exists", funcName)
	}
	if ok := p.stdFuncs.Check(funcName); ok {
		p.addError(curNode.token32, "Function '%s' exists in standard library", funcName)
	}
	curNode = curNode.next
	var argsNode *node32
	for {
		curNode = skipToNextRule(curNode)
		if isRule(curNode, ruleFuncArgSeq) {
			argsNode = curNode
			curNode = curNode.next
		}
		if isRule(curNode, ruleExpr) {
			break
		}
	}
	argsNames, argsTypes := p.ruleFuncArgSeqHandler(argsNode)
	expr, varType := p.ruleExprHandler(curNode)
	if argsTypes == nil || expr == nil {
		return nil, nil, nil
	}
	p.stack.dropFrame()
	p.stack.pushFunc(s.FunctionParams{
		ID:         ast.UserFunction(funcName),
		Arguments:  argsTypes,
		ReturnType: varType,
	})

	if len(argsNames) == 0 {
		return &ast.FunctionDeclarationNode{
			Name:                funcName,
			Arguments:           []string{},
			Body:                expr,
			Block:               nil,
			InvocationParameter: "",
		}, varType, argsTypes
	}

	return &ast.FunctionDeclarationNode{
		Name:                funcName,
		Arguments:           argsNames,
		Body:                expr,
		Block:               nil,
		InvocationParameter: "",
	}, varType, argsTypes
}

func (p *astParser) ruleFuncArgSeqHandler(node *node32) ([]string, []s.Type) {
	if node == nil {
		return []string{}, []s.Type{}
	}
	curNode := node.up
	argName, argType := p.ruleFuncArgHandler(curNode)
	if argType == nil {
		return nil, nil
	}
	curNode = curNode.next
	argsNames := []string{argName}
	argsTypes := []s.Type{argType}
	if curNode == nil {
		return argsNames, argsTypes
	}
	curNode = skipToNextRule(curNode)
	nextArgsNames, nextArgsTypes := p.ruleFuncArgSeqHandler(curNode)
	if nextArgsTypes == nil {
		return nil, nil
	}
	return append(argsNames, nextArgsNames...), append(argsTypes, nextArgsTypes...)
}

func (p *astParser) ruleFuncArgHandler(node *node32) (string, s.Type) {
	curNode := node.up
	argName := p.nodeValue(curNode)
	curNode = skipToNextRule(curNode.next)
	argType := p.ruleTypesHandler(curNode)
	if argType == nil {
		return "", nil
	}
	p.stack.pushVariable(s.Variable{
		Name: argName,
		Type: argType,
	})
	return argName, argType
}

func (p *astParser) ruleTypesHandler(node *node32) s.Type {
	curNode := node.up
	var T s.Type
	switch curNode.pegRule {
	case ruleGenericType:
		T = p.ruleGenericTypeHandler(curNode)
	case ruleTupleType:
		T = p.ruleTupleTypeHandler(curNode)
	case ruleType:
		name := p.nodeValue(curNode)
		if foundType, ok := p.stdTypes[name]; !ok {
			p.addError(curNode.token32, "Undefined type '%s'", name)
		} else {
			T = foundType
		}
	}
	if T == nil {
		return nil
	}
	curNode = curNode.next
	if curNode == nil {
		return T
	}

	resType := s.UnionType{Types: []s.Type{}}
	resType.AppendType(T)
	curNode = skipToNextRule(curNode)
	T = p.ruleTypesHandler(curNode)
	if T == nil {
		return nil
	}
	resType.AppendType(T)
	return resType
}

func (p *astParser) ruleTupleTypeHandler(node *node32) s.Type {
	curNode := node.up
	var tupleTypes []s.Type
	for {
		curNode = skipToNextRule(curNode)
		if isRule(curNode, ruleTypes) {
			T := p.ruleTypesHandler(curNode)
			if T == nil {
				return nil
			}
			tupleTypes = append(tupleTypes, T)
			curNode = curNode.next
		}
		if curNode == nil {
			break
		}
	}
	return s.TupleType{Types: tupleTypes}
}

func (p *astParser) ruleGenericTypeHandler(node *node32) s.Type {
	curNode := node.up
	name := p.nodeValue(curNode)
	if name != "List" {
		p.addError(curNode.token32, "Generic type should be 'List', but '%s' found", name)
		return nil
	}
	curNode = skipToNextRule(curNode.next)
	T := p.ruleTypesHandler(curNode)
	if T == nil {
		return T
	}
	return s.ListType{Type: T}
}

func (p *astParser) parseAnnotatedFunc(node *node32) {
	curNode := node
	for {
		if isRule(curNode, ruleAnnotatedFunc) {
			p.ruleAnnotatedFunc(curNode.up)
			curNode = curNode.next
		}
		curNode = skipToNextRule(curNode)
		if curNode == nil || (curNode.pegRule != rule_ && curNode.pegRule != ruleAnnotatedFunc) {
			break
		}
	}
}

func (p *astParser) loadMeta(name string, argsTypes []s.Type) error {
	if int(p.tree.LibVersion) <= 3 {
		p.tree.Meta.Version = 1
	} else {
		p.tree.Meta.Version = 2
	}
	switch p.tree.LibVersion {
	case ast.LibV1, ast.LibV2, ast.LibV3, ast.LibV4, ast.LibV5:
		return p.loadMetaBeforeV6(name, argsTypes)
	case ast.LibV6, ast.LibV7, ast.LibV8:
		return p.loadMetaV6(name, argsTypes)
	}
	return nil
}

func (p *astParser) loadMetaV6(name string, argsTypes []s.Type) error {
	res := meta.Function{
		Name:      name,
		Arguments: []meta.Type{},
	}
	for _, t := range argsTypes {
		switch T := t.(type) {
		case s.SimpleType:
			metaT, err := getMetaType(t)
			if err != nil {
				return err
			}
			res.Arguments = append(res.Arguments, metaT)
		case s.ListType:
			if _, ok := T.Type.(s.SimpleType); !ok {
				return errors.Errorf("Unexpected type in callable args '%s'", t.String())
			}
			metaT, err := getMetaType(T.Type)
			if err != nil {
				return err
			}
			res.Arguments = append(res.Arguments, meta.ListType{Inner: metaT})
		default:
			return errors.Errorf("Unexpected type in callable args '%s'", t.String())
		}
	}
	p.tree.Meta.Functions = append(p.tree.Meta.Functions, res)
	return nil
}

func getMetaType(t s.Type) (meta.SimpleType, error) {
	if simpleType, ok := t.(s.SimpleType); ok {
		switch simpleType.Type {
		case "String":
			return meta.String, nil
		case "Int":
			return meta.Int, nil
		case "Boolean":
			return meta.Boolean, nil
		case "ByteVector":
			return meta.Bytes, nil
		}
	}
	return meta.SimpleType(byte(0)), errors.Errorf("Unexpected type in callable args '%s'", t.String())
}

func (p *astParser) loadMetaBeforeV6(name string, argsTypes []s.Type) error {
	res := meta.Function{
		Name:      name,
		Arguments: []meta.Type{},
	}
	for _, t := range argsTypes {
		switch T := t.(type) {
		case s.SimpleType:
			metaT, err := getMetaType(t)
			if err != nil {
				return err
			}
			res.Arguments = append(res.Arguments, metaT)
		case s.ListType:
			switch u := T.Type.(type) {
			case s.SimpleType:
				metaT, err := getMetaType(T.Type)
				if err != nil {
					return err
				}
				res.Arguments = append(res.Arguments, meta.ListType{Inner: metaT})
			case s.UnionType:
				var resType []meta.SimpleType
				for _, unionT := range u.Types {
					metaT, err := getMetaType(unionT)
					if err != nil {
						return err
					}
					resType = append(resType, metaT)
				}
				res.Arguments = append(res.Arguments, meta.ListType{Inner: meta.UnionType(resType)})
			}
		case s.UnionType:
			var resType []meta.SimpleType
			for _, unionT := range T.Types {
				metaT, err := getMetaType(unionT)
				if err != nil {
					return err
				}
				resType = append(resType, metaT)
			}
			res.Arguments = append(res.Arguments, meta.UnionType(resType))
		default:
			return errors.Errorf("Unexpected type in callable args '%s'", t.String())
		}
	}
	p.tree.Meta.Functions = append(p.tree.Meta.Functions, res)
	return nil
}

func (p *astParser) ruleAnnotatedFunc(node *node32) {
	p.stack.addFrame()
	curNode := node
	annotation, annotationParameter := p.ruleAnnotationSeqHandler(curNode)
	if annotation == "" {
		return
	}
	curNode = skipToNextRule(curNode.next)
	expr, retType, types := p.ruleFuncHandler(curNode)
	if expr == nil || retType == nil {
		return
	}
	f := expr.(*ast.FunctionDeclarationNode)
	f.InvocationParameter = annotationParameter
	switch annotation {
	case "Callable":
		p.tree.Functions = append(p.tree.Functions, expr)
		err := p.loadMeta(f.Name, types)
		if err != nil {
			p.addError(curNode.token32, "%v", err.Error())
		}
		switch p.tree.LibVersion {
		case ast.LibV1, ast.LibV2, ast.LibV3:
			if !s.CallableRetV3.EqualWithEntry(retType) && !s.ThrowType.Equal(retType) {
				p.addError(curNode.token32, "CallableFunc must return %s, but return %s", s.CallableRetV3.String(), retType.String())
			}
		case ast.LibV4:
			if !s.CallableRetV4.EqualWithEntry(retType) && !s.ThrowType.Equal(retType) {
				p.addError(curNode.token32, "CallableFunc must return %s,but return %s", s.CallableRetV4.String(), retType.String())
			}
		case ast.LibV5, ast.LibV6, ast.LibV7, ast.LibV8:
			if !s.CallableRetV5.EqualWithEntry(retType) && !s.ThrowType.Equal(retType) {
				p.addError(curNode.token32, "CallableFunc must return %s, but return %s", s.CallableRetV5.String(), retType.String())
			}
		}
	case "Verifier":
		if p.tree.Verifier != nil {
			p.addError(curNode.token32, "More than one Verifier")
		}
		p.tree.Verifier = f
		if len(types) != 0 {
			p.addError(curNode.token32, "Verifier must not have arguments")
		}
		if !s.BooleanType.Equal(retType) {
			p.addError(curNode.token32, "VerifierFunction must return Boolean or it super type")
		}
	}
	p.stack.dropFrame()
}

func (p *astParser) ruleAnnotationSeqHandler(node *node32) (string, string) {
	curNode := node.up
	annotationNode := curNode.up
	name := p.nodeValue(annotationNode)
	if name != "Callable" && name != "Verifier" {
		p.addError(annotationNode.token32, "Undefined annotation '%s'", name)
		return "", ""
	}
	annotationNode = skipToNextRule(annotationNode)
	annotationNode = annotationNode.next.up
	varName := p.nodeValue(annotationNode)
	annotationNode = annotationNode.next
	if annotationNode != nil {
		p.addError(annotationNode.token32, "More then one variable in annotation '%s'", name)
	}
	curNode = curNode.next
	if curNode != nil {
		p.addError(curNode.token32, "More then one annotation")
	}

	switch name {
	case "Callable":
		p.stack.pushVariable(s.Variable{
			Name: varName,
			Type: s.SimpleType{Type: "Invocation"},
		})
	case "Verifier":
		txType := p.stdTypes["Transaction"].(s.UnionType)
		txType.AppendType(s.SimpleType{Type: "Order"})
		p.stack.pushVariable(s.Variable{
			Name: varName,
			Type: txType,
		})
	}
	return name, varName
}

func (p *astParser) ruleMatchHandler(node *node32) (ast.Node, s.Type) {
	p.stack.addFrame()
	curNode := skipToNextRule(node.up)
	expr, varType := p.ruleExprHandler(curNode)
	curNode = curNode.next
	possibleTypes := s.UnionType{Types: []s.Type{}}

	if t, ok := varType.(s.UnionType); ok {
		possibleTypes = t
	} else {
		possibleTypes.AppendType(varType)
	}
	var matchName string

	if lastMatchName, ok := p.stack.topMatchName(); ok {
		matchNumStr := strings.TrimPrefix(lastMatchName, "$match")
		matchNum, err := strconv.ParseInt(matchNumStr, 10, 64)
		if err != nil {
			p.addError(token32{}, "Failed to parse 'Int' value: %v", err)
		}
		matchName = fmt.Sprintf("$match%d", matchNum+1)
	} else {
		matchName = "$match0"
	}
	p.stack.pushVariable(s.Variable{
		Name: matchName,
		Type: varType,
	})
	var conds, trueStates []ast.Node
	var defaultCase ast.Node
	unionRetType := s.UnionType{Types: []s.Type{}}
	for {
		curNode = skipToNextRule(curNode)
		if isRule(curNode, ruleCase) {
			// new stack for each case
			p.stack.addFrame()
			cond, trueState, caseVarType := p.ruleCaseHandle(curNode, matchName, possibleTypes)
			if trueState == nil {
				if defaultCase != nil {
					p.addError(curNode.token32, "Match should have at most one default case")
				}
				defaultCase = cond
			} else {
				conds = append(conds, cond)
				trueStates = append(trueStates, trueState)
			}
			unionRetType.AppendType(caseVarType)
			curNode = curNode.next
			p.stack.dropFrame()
		}
		if curNode == nil {
			break
		}
	}
	if defaultCase == nil {
		defaultCase = ast.NewFunctionCallNode(ast.NativeFunction("2"), []ast.Node{ast.NewStringNode("Match error")})
	}
	falseState := defaultCase
	for i := len(conds) - 1; i >= 0; i-- {
		falseState = ast.NewConditionalNode(conds[i], trueStates[i], falseState)
	}
	p.stack.dropFrame()
	return ast.NewAssignmentNode(matchName, expr, falseState), unionRetType.Simplify()
}

func (p *astParser) ruleCaseHandle(node *node32, matchName string, possibleTypes s.UnionType) (ast.Node, ast.Node, s.Type) {
	curNode := skipToNextRule(node.up)
	statementNode := curNode
	curNode = skipToNextRule(curNode.next)
	var blockType s.Type
	var cond, trueState, block ast.Node
	switch statementNode.pegRule {
	case ruleValuePattern:
		cond, trueState = p.ruleValuePatternHandler(statementNode, matchName, possibleTypes)
		block, blockType = p.ruleBlockHandler(curNode)
		if trueState == nil {
			trueState = block
		} else {
			trueState.SetBlock(block)
		}
	case ruleTuplePattern:
		ifCond, decls := p.ruleTuplePatternHandler(statementNode, matchName, possibleTypes)
		block, blockType = p.ruleBlockHandler(curNode)
		cond = ifCond
		if decls == nil {
			trueState = block
		} else {
			for i := len(decls) - 1; i >= 0; i-- {
				decls[i].SetBlock(block)
				block = decls[i]
			}
			trueState = block
		}
	case ruleObjectPattern:
		ifCond, decls := p.ruleObjectPatternHandler(statementNode, matchName, possibleTypes)
		block, blockType = p.ruleBlockHandler(curNode)
		if ifCond == nil {
			return block, nil, blockType
		}
		cond = ifCond
		if decls == nil {
			trueState = block
		} else {
			for i := len(decls) - 1; i >= 0; i-- {
				decls[i].SetBlock(block)
				block = decls[i]
			}
			trueState = block
		}
	case rulePlaceholder:
		block, blockType = p.ruleBlockHandler(curNode)
		return block, nil, blockType
	case ruleExpr:
		expr, varType := p.ruleExprHandler(statementNode)
		if !possibleTypes.EqualWithEntry(varType) {
			p.addError(curNode.token32, "Matching not exhaustive: possible Types are '%s', while matched are '%s'", possibleTypes.String(), varType.String())
		}
		cond = ast.NewFunctionCallNode(ast.NativeFunction("0"), []ast.Node{
			expr,
			ast.NewReferenceNode(matchName),
		})
		trueState, blockType = p.ruleBlockHandler(curNode)
	}
	return cond, trueState, blockType
}

func (p *astParser) ruleValuePatternHandler(node *node32, matchName string, possibleTypes s.UnionType) (ast.Node, ast.Node) {
	curNode := node.up
	nameNode := curNode
	curNode = skipToNextRule(curNode.next)
	t := p.ruleTypesHandler(curNode)

	if !possibleTypes.EqualWithEntry(t) {
		p.addError(curNode.token32, "Matching not exhaustive: possible Types are '%s', while matched are '%s'", possibleTypes.String(), t.String())
	}

	var decl ast.Node = nil

	if nameNode.pegRule != rulePlaceholder {
		name := p.nodeValue(nameNode)
		if _, ok := p.stack.variable(name); ok {
			p.addError(nameNode.token32, "Variable '%s' already exists", name)
		}
		p.stack.pushVariable(s.Variable{
			Name: name,
			Type: t,
		})
		decl = ast.NewAssignmentNode(name, ast.NewReferenceNode(matchName), nil)
	}

	if u, ok := t.(s.UnionType); ok {
		var checks []ast.Node
		for _, unionType := range u.Types {
			if tuple, ok := unionType.(s.TupleType); ok {
				combs := createAllCombinationsOfTuples(tuple)
				for _, comb := range combs {
					checks = append(checks, ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewReferenceNode(matchName), ast.NewStringNode(comb.String())}))
				}
			} else {
				checks = append(checks, ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewReferenceNode(matchName), ast.NewStringNode(unionType.String())}))
			}
		}
		var result ast.Node
		for i := 0; i != len(checks); i++ {
			if i == 0 {
				result = checks[i]
				continue
			}
			result = ast.NewConditionalNode(checks[i], ast.NewBooleanNode(true), result)
		}
		return result, decl
	}

	if tuple, ok := t.(s.TupleType); ok {
		var checks []ast.Node
		combs := createAllCombinationsOfTuples(tuple)
		for _, comb := range combs {
			checks = append(checks, ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewReferenceNode(matchName), ast.NewStringNode(comb.String())}))
		}
		var result ast.Node
		for i := 0; i != len(checks); i++ {
			if i == 0 {
				result = checks[i]
				continue
			}
			result = ast.NewConditionalNode(checks[i], ast.NewBooleanNode(true), result)
		}
		return result, decl
	}

	return ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewReferenceNode(matchName), ast.NewStringNode(t.String())}),
		decl
}

func (p *astParser) ruleObjectPatternHandler(node *node32, matchName string, possibleTypes s.UnionType) (ast.Node, []ast.Node) {
	curNode := node.up
	structName := p.nodeValue(curNode)
	if !p.stdObjects.IsExist(structName) {
		p.addError(curNode.token32, "Object with this name '%s' doesn't exist", structName)
		return nil, nil
	}
	if !possibleTypes.EqualWithEntry(s.SimpleType{Type: structName}) {
		p.addError(curNode.token32, "Matching not exhaustive: possible Types are '%s', while matched are '%s'", possibleTypes.String(), structName)
		return nil, nil
	}
	curNode = curNode.next

	var exprs []ast.Node
	var shadowDeclarations []ast.Node
	exprs = append(exprs, ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewReferenceNode(matchName), ast.NewStringNode(structName)}))
	shadowDeclarations = append(shadowDeclarations, ast.NewAssignmentNode(matchName, ast.NewReferenceNode(matchName), nil))
	for curNode != nil {
		expr, decl, newNode := p.ruleObjectFieldsPatternHandler(curNode, matchName, structName)
		curNode = newNode
		for curNode != nil && curNode.pegRule != ruleObjectFieldsPattern {
			curNode = curNode.next
		}
		if expr == nil && decl == nil {
			continue
		}
		exprs = append(exprs, expr)
		if decl != nil {
			shadowDeclarations = append(shadowDeclarations, decl)
		}
	}

	if len(exprs) == 1 {
		return ast.NewConditionalNode(
			exprs[0],
			ast.NewAssignmentNode(matchName, ast.NewReferenceNode(matchName), ast.NewBooleanNode(true)),
			ast.NewBooleanNode(false)), shadowDeclarations
	}

	var resExpr ast.Node

	for i := len(exprs) - 1; i >= 0; i-- {
		if exprs[i] == nil {
			continue
		}
		if i == len(exprs)-1 && i != 0 {
			resExpr = exprs[i]
			continue
		}
		if i == 0 {
			resExpr = ast.NewConditionalNode(exprs[i], ast.NewAssignmentNode(
				matchName,
				ast.NewReferenceNode(matchName),
				resExpr,
			), ast.NewBooleanNode(false))
			continue
		}
		resExpr = ast.NewConditionalNode(exprs[i], resExpr, ast.NewBooleanNode(false))
	}

	return resExpr, shadowDeclarations

}

func (p *astParser) ruleObjectFieldsPatternHandler(node *node32, matchName string, structName string) (ast.Node, ast.Node, *node32) {
	curNode := node.up
	fieldName := p.nodeValue(curNode)
	t, ok := p.stdObjects.GetField(s.SimpleType{Type: structName}, fieldName)
	if !ok {
		p.addError(curNode.token32, "Object '%s' doesn't have field '%s'", structName, fieldName)
		return nil, nil, curNode
	}
	curNode = skipToNextRule(curNode.next)
	switch curNode.pegRule {
	case ruleIdentifier:
		name := p.nodeValue(curNode)
		if _, ok := p.stack.variable(name); ok {
			p.addError(curNode.token32, "Variable '%s' already exists", name)
			return nil, nil, curNode
		}
		p.stack.pushVariable(s.Variable{
			Name: name,
			Type: t,
		})
		return nil,
			ast.NewAssignmentNode(name, ast.NewPropertyNode(fieldName, ast.NewReferenceNode(matchName)), nil), curNode
	case ruleExpr:
		expr, exprType := p.ruleExprHandler(curNode)
		if expr == nil {
			return nil, nil, curNode
		}
		if !t.EqualWithEntry(exprType) {
			p.addError(curNode.token32, "Can't match inferred types: field '%s' has type '%s', but '%s' provided", fieldName, t.String(), exprType.String())
			return nil, nil, curNode
		}
		return ast.NewFunctionCallNode(ast.NativeFunction("0"), []ast.Node{
			expr,
			ast.NewPropertyNode(fieldName, ast.NewReferenceNode(matchName)),
		}), nil, curNode
	}
	return nil, nil, nil
}

func combinations(temp []s.Type, lists [][]s.Type, result *[][]s.Type) {
	if len(lists) == 0 {
		dst := make([]s.Type, len(temp))
		copy(dst, temp)
		*result = append(*result, dst)
		return
	}
	for _, l := range lists[0] {
		combinations(append(temp, l), lists[1:], result)
	}
}

func createAllCombinationsOfTuples(tuple s.TupleType) []s.TupleType {
	var lists [][]s.Type
	for _, t := range tuple.Types {
		if u, ok := t.(s.UnionType); ok {
			lists = append(lists, u.Types)
		} else {
			lists = append(lists, []s.Type{t})
		}
	}
	var result [][]s.Type
	combinations([]s.Type{}, lists, &result)
	var resultListOfTuple []s.TupleType
	for _, l := range result {
		resultListOfTuple = append(resultListOfTuple, s.TupleType{Types: l})
	}
	return resultListOfTuple
}

func (p *astParser) ruleTuplePatternHandler(node *node32, matchName string, possibleTypes s.UnionType) (ast.Node, []ast.Node) {
	curNode := node.up
	var exprs []ast.Node
	var varsTypes []s.Type
	var shadowDeclarations []ast.Node
	cnt := 0
	for {
		curNode = skipToNextRule(curNode)
		if curNode == nil || curNode.pegRule != ruleTupleValuesPattern {
			break
		}
		expr, decl, t := p.ruleTupleValuesPatternHandler(curNode, matchName, cnt, possibleTypes)
		exprs = append(exprs, expr)
		varsTypes = append(varsTypes, t)
		shadowDeclarations = append(shadowDeclarations, decl...)
		cnt++
		curNode = curNode.up
		if curNode.next != nil {
			curNode = curNode.next
		}
		curNode = skipToNextRule(curNode)
	}
	tupleType := s.TupleType{Types: varsTypes}
	if !possibleTypes.EqualWithEntry(tupleType) {
		p.addError(curNode.token32, "Matching not exhaustive: possible Types are '%s', while matched are '%s'", possibleTypes.String(), tupleType)
	}
	var cond ast.Node
	setLast := false
	setPlaceHolder := false
	for i := len(exprs) - 1; i >= 0; i-- {
		if exprs[i] == nil {
			setPlaceHolder = true
			continue
		}
		if !setLast {
			cond = exprs[i]
			setLast = true
			continue
		}
		cond = ast.NewConditionalNode(exprs[i], cond, ast.NewBooleanNode(false))
	}
	if cond == nil {
		cond = ast.NewBooleanNode(true)
	}
	var trueState ast.Node
	if setPlaceHolder {
		trueState = ast.NewFunctionCallNode(ast.NativeFunction("0"), []ast.Node{
			ast.NewFunctionCallNode(ast.NativeFunction("1350"), []ast.Node{ast.NewReferenceNode(matchName)}),
			ast.NewLongNode(int64(len(exprs)))})
	} else {
		trueState = ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{
			ast.NewReferenceNode(matchName),
			ast.NewStringNode(tupleType.String())})
	}
	return ast.NewConditionalNode(cond, trueState, ast.NewBooleanNode(false)), shadowDeclarations
}

func (p *astParser) ruleTupleValuesPatternHandler(node *node32, matchName string, cnt int, possibleTypes s.UnionType) (ast.Node, []ast.Node, s.Type) {
	curNode := node.up
	var expr ast.Node
	var varType s.Type
	var shadowDeclarations []ast.Node
	switch curNode.pegRule {
	case ruleValuePattern:
		curNode = curNode.up
		nameNode := curNode
		curNode = skipToNextRule(curNode.next)
		varType = p.ruleTypesHandler(curNode)

		expr = ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewPropertyNode("_"+strconv.Itoa(cnt+1), ast.NewReferenceNode(matchName)), ast.NewStringNode(varType.String())})
		if nameNode.pegRule != rulePlaceholder {
			name := p.nodeValue(nameNode)
			p.stack.pushVariable(s.Variable{
				Name: name,
				Type: varType,
			})
			shadowDeclarations = append(shadowDeclarations, ast.NewAssignmentNode(name, ast.NewPropertyNode("_"+strconv.Itoa(cnt+1), ast.NewReferenceNode(matchName)), nil))
		}
	case ruleIdentifier:
		name := p.nodeValue(curNode)

		for _, t := range possibleTypes.Types {
			if tuple, ok := t.(s.TupleType); ok {
				if cnt >= len(tuple.Types) {
					continue
				}
				varType = tuple.Types[cnt]
			}
		}

		p.stack.pushVariable(s.Variable{
			Name: name,
			Type: varType,
		})
		shadowDeclarations = append(shadowDeclarations, ast.NewAssignmentNode(name, ast.NewPropertyNode("_"+strconv.Itoa(cnt+1), ast.NewReferenceNode(matchName)), nil))
	case rulePlaceholder:
		// skip and return nil
		break
	case ruleExpr:
		expr, varType = p.ruleExprHandler(curNode)
		expr = ast.NewFunctionCallNode(ast.NativeFunction("0"), []ast.Node{expr, ast.NewPropertyNode("_"+strconv.Itoa(cnt+1), ast.NewReferenceNode(matchName))})
	case ruleGettableExpr:
		expr, varType = p.ruleGettableExprHandler(curNode)
		expr = ast.NewFunctionCallNode(ast.NativeFunction("0"), []ast.Node{expr, ast.NewPropertyNode("_"+strconv.Itoa(cnt+1), ast.NewReferenceNode(matchName))})
	}
	return expr, shadowDeclarations, varType
}

func (p *astParser) ruleFoldMacroHandler(node *node32) (ast.Node, s.Type) {
	curNode := skipToNextRule(node.up)
	// parse num in fold macro
	value := p.nodeValue(curNode)
	if strings.Contains(value, "_") {
		value = strings.ReplaceAll(value, "_", "")
	}
	iterNum, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		p.addError(curNode.token32, "Failed to parse integer value: %v", err)
	}
	curNode = skipToNextRule(curNode.next)
	arr, arrVarType := p.ruleExprHandler(curNode)
	if arr == nil {
		p.addError(curNode.token32, "Undefined first argument of FOLD macros")
		return nil, nil
	}
	l, ok := arrVarType.(s.ListType)
	if !ok {
		p.addError(curNode.token32, "First argument of FOLD macros must be List, but '%s' found",
			arrVarType.String())
		return nil, nil
	}
	elemType := l.Type
	if elemType == nil { // If the type of elements is unknown, set it to Any.
		elemType = s.AnyType
	}
	curNode = skipToNextRule(curNode.next)
	start, startVarType := p.ruleExprHandler(curNode)
	curNode = skipToNextRule(curNode.next)
	funcName := p.nodeValue(curNode)
	funcSign, ok := p.stack.function(funcName)
	if !ok {
		p.addError(curNode.token32, "Undefined function '%s'", funcName)
		return nil, nil
	}
	if len(funcSign.Arguments) != 2 {
		p.addError(curNode.token32, "Function '%s' must have 2 arguments", funcName)
	} else {
		if !funcSign.Arguments[0].EqualWithEntry(startVarType) || !funcSign.Arguments[1].EqualWithEntry(elemType) {
			p.addError(curNode.token32, "Can't find suitable function '%s(%s, %s)'", funcName, elemType.String(), startVarType.String())
		}
	}

	var decls []ast.Node
	assign := ast.NewAssignmentNode("$l", arr, nil)
	assign.NewBlock = true
	decls = append(decls, assign)
	assign = ast.NewAssignmentNode("$s", ast.NewFunctionCallNode(ast.NativeFunction("400"), []ast.Node{ast.NewReferenceNode("$l")}), nil)
	assign.NewBlock = true
	decls = append(decls, assign)
	assign = ast.NewAssignmentNode("$acc0", start, nil)
	assign.NewBlock = true
	decls = append(decls, assign)
	decls = append(decls,
		ast.NewFunctionDeclarationNode("$f0_1", []string{"$a", "$i"},
			ast.NewConditionalNode(ast.NewFunctionCallNode(ast.NativeFunction("103"), []ast.Node{ast.NewReferenceNode("$i"), ast.NewReferenceNode("$s")}),
				ast.NewReferenceNode("$a"),
				ast.NewFunctionCallNode(ast.UserFunction(funcName), []ast.Node{
					ast.NewReferenceNode("$a"),
					ast.NewFunctionCallNode(ast.NativeFunction("401"), []ast.Node{ast.NewReferenceNode("$l"), ast.NewReferenceNode("$i")}),
				}),
			),
			nil,
		),
	)
	decls = append(decls,
		ast.NewFunctionDeclarationNode("$f0_2", []string{"$a", "$i"},
			ast.NewConditionalNode(ast.NewFunctionCallNode(ast.NativeFunction("103"), []ast.Node{ast.NewReferenceNode("$i"), ast.NewReferenceNode("$s")}),
				ast.NewReferenceNode("$a"),
				ast.NewFunctionCallNode(ast.NativeFunction("2"), []ast.Node{ast.NewStringNode("List size exceeds " + strconv.FormatInt(iterNum, 10))}),
			),
			nil,
		),
	)
	fcall := ast.NewFunctionCallNode(ast.UserFunction("$f0_1"), []ast.Node{ast.NewReferenceNode("$acc0"), ast.NewLongNode(0)})
	for i := int64(1); i < iterNum; i++ {
		fcall = ast.NewFunctionCallNode(ast.UserFunction("$f0_1"), []ast.Node{fcall, ast.NewLongNode(i)})
	}
	fcall = ast.NewFunctionCallNode(ast.UserFunction("$f0_2"), []ast.Node{fcall, ast.NewLongNode(iterNum)})
	decls = append(decls, fcall)

	var expr, block ast.Node
	for i := len(decls) - 1; i >= 0; i-- {
		expr = decls[i]
		expr.SetBlock(block)
		block = expr
	}
	return block, funcSign.ReturnType
}

func isRule(node *node32, rule pegRule) bool {
	return node != nil && node.pegRule == rule
}

func skipToNextRule(node *node32) *node32 {
	for {
		if node == nil {
			return nil
		}
		if node.pegRule != rule_ {
			return node
		}
		node = node.next
	}
}
