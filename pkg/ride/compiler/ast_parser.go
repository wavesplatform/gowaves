package compiler

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	"strconv"
	"strings"

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

type ScriptType byte

const (
	Account ScriptType = iota + 1
	Asset
)

type ASTError struct {
	msg   string
	begin textPosition
	end   textPosition
}

func NewASTError(msg string, token token32, buffer []rune) error {
	begin := int(token.begin)
	end := int(token.end)
	positions := []int{begin, end}
	translations := translatePositions(buffer, positions)
	return &ASTError{msg: msg, begin: translations[begin], end: translations[end]}
}

func (e *ASTError) Error() string {
	return fmt.Sprintf("(%d:%d, %d:%d): %s", e.begin.line, e.begin.symbol, e.end.line, e.end.symbol, e.msg)
}

type ASTParser struct {
	node   *node32
	Tree   *ast.Tree
	buffer []rune

	ErrorsList   []error
	globalStack  *VarStack
	currentStack *VarStack

	stdFuncs   s.FunctionsSignatures
	stdObjects s.ObjectsSignatures

	scriptType ScriptType
}

func NewASTParser(node *node32, buffer []rune) ASTParser {
	return ASTParser{
		node: node,
		Tree: &ast.Tree{
			LibVersion:   ast.LibV6,
			ContentType:  ast.ContentTypeApplication,
			Declarations: []ast.Node{},
			Functions:    []ast.Node{},
		},
		buffer:      buffer,
		ErrorsList:  []error{},
		globalStack: NewVarStack(nil),
		scriptType:  Account,
	}
}

func (p *ASTParser) Parse() {
	p.currentStack = p.globalStack
	switch p.node.pegRule {
	case ruleCode:
		p.ruleCodeHandler(p.node.up)
	}
}

func (p *ASTParser) addError(msg string, token token32) {
	p.ErrorsList = append(p.ErrorsList,
		NewASTError(msg, token, p.buffer))
}

func (p *ASTParser) loadBuildInVarsToStackByVersion() {
	resVars := make(map[string]s.Variable, 0)
	ver := int(p.Tree.LibVersion)
	for i := 0; i < ver; i++ {
		for _, v := range s.Vars.Vars[i].Append {
			resVars[v.Name] = v
		}
		for _, v := range s.Vars.Vars[i].Remove {
			delete(resVars, v)
		}
	}
	for _, v := range resVars {
		p.currentStack.PushVariable(v)
	}
	if p.Tree.LibVersion == ast.LibV5 || p.Tree.LibVersion == ast.LibV6 {
		if p.scriptType == Asset {
			p.currentStack.PushVariable(s.Variable{
				Name: "this",
				Type: s.SimpleType{Type: "Asset"},
			})
		} else {
			p.currentStack.PushVariable(s.Variable{
				Name: "this",
				Type: s.SimpleType{Type: "Address"},
			})
		}
	}
}

func (p *ASTParser) ruleCodeHandler(node *node32) {
	switch node.pegRule {
	case ruleDAppRoot:
		p.ruleDAppRootHandler(node.up)
	case ruleScriptRoot:
		p.ruleScriptRootHandler(node.up)
	}
}

func (p *ASTParser) ruleDAppRootHandler(node *node32) {
	curNode := node
	if curNode.pegRule == rule_ {
		curNode = node.next
	}
	if curNode != nil && curNode.pegRule == ruleDirective {
		curNode = p.parseDirectives(curNode)
		_ = curNode
	}
	p.stdFuncs = s.FuncsByVersion[p.Tree.LibVersion]
	p.stdObjects = s.ObjectsByVersion[p.Tree.LibVersion]
	p.loadBuildInVarsToStackByVersion()
	if curNode != nil && curNode.pegRule == rule_ {
		curNode = node.next
	}
	if curNode != nil && curNode.pegRule == ruleDeclaration {
		curNode = p.parseDeclarations(curNode)
	}
	if curNode != nil && curNode.pegRule == rule_ {
		curNode = node.next
	}
	if curNode != nil && curNode.pegRule == ruleAnnotatedFunc {
		p.parseAnnotatedFunc(curNode)
	}
}

func (p *ASTParser) ruleScriptRootHandler(node *node32) {
	curNode := node
	if curNode.pegRule == rule_ {
		curNode = node.next
	}
	if curNode != nil && curNode.pegRule == ruleDirective {
		curNode = p.parseDirectives(curNode)
		_ = curNode
	}
	p.stdFuncs = s.FuncsByVersion[p.Tree.LibVersion]
	p.stdObjects = s.ObjectsByVersion[p.Tree.LibVersion]
	p.loadBuildInVarsToStackByVersion()
	if curNode != nil && curNode.pegRule == rule_ {
		curNode = node.next
	}
	var decls []ast.Node
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode.pegRule == ruleDeclaration {
			expr, _ := p.ruleDeclarationHandler(curNode.up)
			decls = append(decls, expr...)
			curNode = curNode.next
		}
		if curNode.pegRule == ruleExpr {
			break
		}
	}
	block, varType := p.ruleExprHandler(curNode)
	if !varType.Comp(s.BooleanType) {
		p.addError(fmt.Sprintf("script should return Boolean, but %s returned", varType.String()), curNode.token32)
		return
	}
	expr := block
	for i := len(decls) - 1; i >= 0; i-- {
		expr = decls[i]
		expr.SetBlock(block)
		block = expr
	}
	p.Tree.Verifier = block
}

func (p *ASTParser) parseDirectives(node *node32) *node32 {
	directiveCnt := make(map[string]int)
	curNode := node
	for {
		if curNode != nil && curNode.pegRule == ruleDirective {
			p.ruleDirectiveHandler(curNode, directiveCnt)
			curNode = curNode.next
		}
		if curNode != nil && curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode == nil || (curNode.pegRule != rule_ && curNode.pegRule != ruleDirective) {
			break
		}
	}
	return curNode
}

func (p *ASTParser) ruleDirectiveHandler(node *node32, directiveCnt map[string]int) {
	curNode := node.up
	// skip WS
	curNode = curNode.next
	// get Directive name
	dirNameNode := curNode
	dirName := string(p.buffer[curNode.begin:curNode.end])
	curNode = curNode.next
	// skip WS
	curNode = curNode.next
	// get Directive value
	dirValueNode := curNode
	dirValue := string(p.buffer[dirValueNode.begin:dirValueNode.end])

	switch dirName {
	case stdlibVersionDirectiveName:
		version, err := strconv.ParseInt(dirValue, 10, 8)
		if err != nil {
			p.addError(fmt.Sprintf("failed to parse version \"%s\" : %s", dirValue, err), dirValueNode.token32)
			break
		}
		if version > 6 || version < 1 {
			p.addError(fmt.Sprintf("invalid %s \"%s\"", stdlibVersionDirectiveName, dirValue), dirValueNode.token32)
			version = 6
		}
		p.Tree.LibVersion = ast.LibraryVersion(byte(version))
		p.checkDirectiveCnt(node, stdlibVersionDirectiveName, directiveCnt)
	case contentTypeDirectiveName:
		switch dirValue {
		case dappValueName:
			p.Tree.ContentType = ast.ContentTypeApplication
		case expressionValueName:
			p.Tree.ContentType = ast.ContentTypeExpression
		default:
			p.addError(fmt.Sprintf("Illegal directive value \"%s\" for key \"%s\"", dirValue, contentTypeDirectiveName), dirNameNode.token32)
		}
		p.checkDirectiveCnt(node, contentTypeDirectiveName, directiveCnt)

	case scriptTypeDirectiveName:
		switch dirValue {
		case accountValueName:
			p.scriptType = Account
		case assetValueName:
			p.scriptType = Asset
		case libraryValueName:
			break
			// TODO
		default:
			p.addError(fmt.Sprintf("Illegal directive value \"%s\" for key \"%s\"", dirValue, scriptTypeDirectiveName), dirNameNode.token32)
		}
		p.checkDirectiveCnt(node, scriptTypeDirectiveName, directiveCnt)
	case importDirectiveName:
		break
		// TODO
	default:
		p.addError(fmt.Sprintf("Illegal directive key \"%s\"", dirName), dirNameNode.token32)
	}

}

func (p *ASTParser) checkDirectiveCnt(node *node32, name string, directiveCnt map[string]int) {
	if val, ok := directiveCnt[name]; ok && val == 1 {
		p.addError(fmt.Sprintf("Directive key %s is used more than once", name), node.token32)
	} else {
		directiveCnt[name] = 1
	}
}

func (p *ASTParser) parseDeclarations(node *node32) *node32 {
	curNode := node
	for {
		if curNode != nil && curNode.pegRule == ruleDeclaration {
			expr, _ := p.ruleDeclarationHandler(curNode.up)
			p.Tree.Declarations = append(p.Tree.Declarations, expr...)
			curNode = curNode.next
		}
		if curNode != nil && curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode == nil || (curNode.pegRule != rule_ && curNode.pegRule != ruleDeclaration) {
			break
		}
	}
	return curNode
}

func (p *ASTParser) ruleDeclarationHandler(node *node32) ([]ast.Node, []s.Type) {
	switch node.pegRule {
	case ruleVariable:
		return p.ruleVariableHandler(node)
	case ruleFunc:
		expr, varType, _ := p.ruleFuncHandler(node)
		if expr == nil {
			return nil, nil
		}
		return []ast.Node{expr}, []s.Type{varType}
	default:
		panic(errors.Errorf("wrong type of rule in Declaration: %s", rul3s[node.pegRule]))
	}
}

func (p *ASTParser) ruleVariableHandler(node *node32) ([]ast.Node, []s.Type) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
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

func (p *ASTParser) simpleVariableDeclaration(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	// get Variable Name
	varName := string(p.buffer[curNode.begin:curNode.end])
	curNode = curNode.next

	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	expr, varType := p.ruleExprHandler(curNode)
	if expr == nil {
		return nil, nil
	}
	if _, ok := p.currentStack.GetVariable(varName); ok {
		p.addError(fmt.Sprintf("variable \"%s\" is exist", varName), curNode.token32)
		return nil, nil
	}
	expr = ast.NewAssignmentNode(varName, expr, nil)
	p.currentStack.PushVariable(s.Variable{
		Name: varName,
		Type: varType,
	})
	return expr, varType
}

func (p *ASTParser) tupleRefDeclaration(node *node32) ([]ast.Node, []s.Type) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	var varNames []string
	tupleRefNode := curNode.up
	for {
		if tupleRefNode.pegRule == rule_ {
			tupleRefNode = tupleRefNode.next
		}
		if tupleRefNode.pegRule == ruleIdentifier {
			name := string(p.buffer[tupleRefNode.begin:tupleRefNode.end])
			if _, ok := p.currentStack.GetVariable(name); ok {
				p.addError(fmt.Sprintf("variable \"%s\" is exist", name), tupleRefNode.token32)
				return nil, nil
			}
			varNames = append(varNames, name)
			tupleRefNode = tupleRefNode.next
		}
		if tupleRefNode == nil {
			break
		}
	}
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	expr, varType := p.ruleExprHandler(curNode)
	if expr == nil {
		return nil, nil
	}
	tuple, ok := varType.(s.TupleType)
	if !ok {
		p.addError(fmt.Sprintf("Expression mast be \"Tuple\" but \"%s\"", varType), curNode.token32)
		return nil, nil
	}
	if len(tuple.Types) < len(varNames) {
		p.addError("Number of Identifiers must be <= tuple length", node.token32)
		return nil, nil
	}
	var resExpr []ast.Node
	var resTypes []s.Type
	tupleName := "$t0" + strconv.FormatUint(uint64(node.begin), 10) + strconv.FormatUint(uint64(node.end), 10)
	p.currentStack.PushVariable(s.Variable{
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
		resTypes = append(resTypes, tuple.Types[i])
		p.currentStack.PushVariable(s.Variable{
			Name: name,
			Type: tuple.Types[i],
		})
	}
	return resExpr, resTypes
}

func (p *ASTParser) ruleExprHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up.up
	expr, varType := p.ruleAndOpAtomHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	if !varType.Comp(s.BooleanType) {
		p.addError(fmt.Sprintf("Unexpected type, required: Boolean, but %s found", varType.String()), node.up.up.token32)
	}
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		// skip orOp
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, nextExprVarType := p.ruleAndOpAtomHandler(curNode)
		if !nextExprVarType.Comp(s.BooleanType) {
			p.addError(fmt.Sprintf("Unexpected type, required: Boolean, but %s found", nextExprVarType.String()), curNode.token32)
		}
		expr = ast.NewConditionalNode(expr, ast.NewBooleanNode(true), nextExpr)
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, s.BooleanType
}

func (p *ASTParser) ruleAndOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleEqualityGroupOpAtomHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	if !varType.Comp(s.BooleanType) {
		p.addError(fmt.Sprintf("Unexpected type, required: Boolean, but %s found", varType.String()), node.up.up.token32)
	}
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		// skip andOp
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, nextExprVarType := p.ruleEqualityGroupOpAtomHandler(curNode)
		if !nextExprVarType.Comp(s.BooleanType) {
			p.addError(fmt.Sprintf("Unexpected type, required: Boolean, but %s found", nextExprVarType.String()), curNode.token32)
		}
		expr = ast.NewConditionalNode(expr, nextExpr, ast.NewBooleanNode(false))
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, s.BooleanType
}

func (p *ASTParser) ruleEqualityGroupOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleCompareGroupOpAtomHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		var funcId ast.Function
		if curNode.up.pegRule == ruleEqOp {
			funcId = ast.NativeFunction("0")
		} else {
			funcId = ast.UserFunction("!=")
		}
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, nextExprVarType := p.ruleCompareGroupOpAtomHandler(curNode)
		if !nextExprVarType.Comp(varType) {
			p.addError(fmt.Sprintf("Unexpected type, required: Boolean, but %s found", nextExprVarType.String()), curNode.token32)
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

func (p *ASTParser) ruleCompareGroupOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleListGroupOpAtomHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	if !varType.Comp(s.BigIntType) && !varType.Comp(s.IntType) {
		p.addError(fmt.Sprintf("Unexpected type, required: BigInt or Int, but %s found", varType.String()), node.up.up.token32)
	}
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		operator := curNode.up.pegRule
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, nextExprVarType := p.ruleListGroupOpAtomHandler(curNode)
		var gltFun, gleFun string
		if varType.Comp(s.BigIntType) {
			if nextExprVarType.Comp(s.BigIntType) {
				gltFun = "319"
				gleFun = "320"
			} else {
				p.addError(fmt.Sprintf("Unexpected type, required: BigInt, but %s found", varType.String()), curNode.token32)
			}
		} else if varType.Comp(s.IntType) {
			if nextExprVarType.Comp(s.IntType) {
				gltFun = "102"
				gleFun = "103"
			} else {
				p.addError(fmt.Sprintf("Unexpected type, required: Int, but %s found", varType.String()), curNode.token32)
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

func (p *ASTParser) ruleSumGroupOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleMultGroupOpAtomHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		operator := curNode.up.pegRule
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, nextExprVarType := p.ruleMultGroupOpAtomHandler(curNode)
		var funcId string
		switch operator {
		case ruleSumOp:
			if varType.Comp(s.IntType) && nextExprVarType.Comp(s.IntType) {
				funcId = "100"
			} else if varType.Comp(s.BigIntType) && nextExprVarType.Comp(s.BigIntType) {
				funcId = "311"
			} else if varType.Comp(s.StringType) && nextExprVarType.Comp(s.StringType) {
				funcId = "300"
			} else if varType.Comp(s.ByteVectorType) && nextExprVarType.Comp(s.ByteVectorType) {
				funcId = "203"
			} else {
				p.addError(fmt.Sprintf("Unexpected types for + operator: %s, %s", varType.String(), nextExprVarType.String()), node.token32)
			}
		case ruleSubOp:
			if varType.Comp(s.IntType) && nextExprVarType.Comp(s.IntType) {
				funcId = "101"
			} else if varType.Comp(s.BigIntType) && nextExprVarType.Comp(s.BigIntType) {
				funcId = "311"
			} else {
				p.addError(fmt.Sprintf("Unexpected types for - operator: %s, %s", varType.String(), nextExprVarType.String()), node.token32)
			}
		}
		expr = ast.NewFunctionCallNode(ast.NativeFunction(funcId), []ast.Node{expr, nextExpr})
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, s.IntType
}

func (p *ASTParser) ruleListGroupOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleSumGroupOpAtomHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	resListType := s.ListType{Type: varType}
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		operator := curNode.up.pegRule
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, nextVarType := p.ruleSumGroupOpAtomHandler(curNode)
		var funcId ast.Function
		switch operator {
		case ruleConsOp:
			if _, ok := nextVarType.(s.ListType); !ok {
				p.addError(fmt.Sprintf("Unexpected types for :: operator: %s, %s", varType, nextVarType), curNode.token32)
				return nil, nil
			}
			funcId = ast.UserFunction("cons")
			resListType.AppendList(nextVarType)
		case ruleAppendOp:
			if l, ok := varType.(s.ListType); !ok {
				p.addError(fmt.Sprintf("Unexpected types for +: operator: %s, %s", varType, nextVarType), curNode.token32)
				return nil, nil
			} else {
				funcId = ast.NativeFunction("1101")
				l.AppendType(nextVarType)
			}
		case ruleConcatOp:
			_, ok1 := nextVarType.(s.ListType)
			l, ok2 := varType.(s.ListType)
			if !ok1 && !ok2 {
				p.addError(fmt.Sprintf("Unexpected types for ++ operator: %s, %s", varType, nextVarType), curNode.token32)
				return nil, nil
			}
			funcId = ast.NativeFunction("1102")
			l.AppendList(nextVarType)
		}
		expr = ast.NewFunctionCallNode(funcId, []ast.Node{expr, nextExpr})
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, resListType
}

func (p *ASTParser) ruleMultGroupOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleAtomExprHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		operator := curNode.up.pegRule
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, nextExprVarType := p.ruleAtomExprHandler(curNode)
		var funcId string
		switch operator {
		case ruleMulOp:
			if varType.Comp(s.IntType) && nextExprVarType.Comp(s.IntType) {
				funcId = "104"
			} else if varType.Comp(s.BigIntType) && nextExprVarType.Comp(s.BigIntType) {
				funcId = "313"
			} else {
				p.addError(fmt.Sprintf("Unexpected types for * operator: %s, %s", varType.String(), nextExprVarType.String()), node.token32)
			}
		case ruleDivOp:
			if varType.Comp(s.IntType) && nextExprVarType.Comp(s.IntType) {
				funcId = "105"
			} else if varType.Comp(s.BigIntType) && nextExprVarType.Comp(s.BigIntType) {
				funcId = "314"
			} else {
				p.addError(fmt.Sprintf("Unexpected types for / operator: %s, %s", varType.String(), nextExprVarType.String()), node.token32)
			}
		case ruleModOp:
			if varType.Comp(s.IntType) && nextExprVarType.Comp(s.IntType) {
				funcId = "106"
			} else if varType.Comp(s.BigIntType) && nextExprVarType.Comp(s.BigIntType) {
				funcId = "315"
			} else {
				p.addError(fmt.Sprintf("Unexpected types for * operator: %s, %s", varType.String(), nextExprVarType.String()), node.token32)
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

func (p *ASTParser) ruleAtomExprHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	var unaryOp pegRule
	if curNode.pegRule == ruleUnaryOp {
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
	switch unaryOp {
	case ruleNegativeOp:
		if varType.Comp(s.IntType) {
			expr = ast.NewFunctionCallNode(ast.UserFunction("-"), []ast.Node{expr})
		} else if varType.Comp(s.BigIntType) {
			expr = ast.NewFunctionCallNode(ast.NativeFunction("318"), []ast.Node{expr})
		} else {
			p.addError(fmt.Sprintf("Unexpected types for unary - operator, required: Int, BigInt, but %s found", varType.String()), curNode.token32)
		}
	case ruleNotOp:
		if varType.Comp(s.BooleanType) {
			expr = ast.NewFunctionCallNode(ast.UserFunction("!"), []ast.Node{expr})
		} else {
			p.addError(fmt.Sprintf("Unexpected types for unary ! operator, required: Boolean, but %s found", varType.String()), curNode.token32)
		}
	case rulePositiveOp:
		if !varType.Comp(s.IntType) && !varType.Comp(s.BigIntType) {
			p.addError(fmt.Sprintf("Unexpected types for unary + operator, required: Int, BigInt, but %s found", varType.String()), curNode.token32)
		}
	}
	return expr, varType
}

func (p *ASTParser) ruleConstHandler(node *node32) (ast.Node, s.Type) {
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

func (p *ASTParser) ruleIntegerHandler(node *node32) (ast.Node, s.Type) {
	value := string(p.buffer[node.begin:node.end])
	if strings.Contains(value, "_") {
		value = strings.ReplaceAll(value, "_", "")
	}
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		p.addError(fmt.Sprintf("failing to parse Integer: %s", err), node.token32)
	}
	return ast.NewLongNode(number), s.IntType
}

func (p *ASTParser) ruleStringHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	var res string
	for {
		if curNode == nil {
			break
		}
		switch curNode.pegRule {
		case ruleChar:
			res += string(p.buffer[curNode.begin:curNode.end])
		case ruleEscapedChar:
			escapedChar := string(p.buffer[curNode.begin:curNode.end])
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
			default:
				p.addError(fmt.Sprintf("unknown escaped symbol: '%s'. The valid are \\b, \\f, \\n, \\r, \\t", escapedChar), curNode.token32)
			}
		case ruleUnicodeChar:
			unicodeChar := string(p.buffer[curNode.begin:curNode.end])
			char, err := strconv.Unquote(`"` + unicodeChar + `"`)
			if err != nil {
				p.addError(fmt.Sprintf("unknow UTF-8 symbol \"\\u%s\"", unicodeChar), curNode.token32)
			} else {
				res += char
			}
		}
		curNode = curNode.next
	}
	return ast.NewStringNode(res), s.StringType
}

func (p *ASTParser) ruleByteVectorHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	var err error
	var value []byte
	valueWithBase := string(p.buffer[curNode.begin:curNode.end])
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
		p.addError(fmt.Sprintf("failing to parse ByteVector: %s", err), node.token32)
	}
	return ast.NewBytesNode(value), s.ByteVectorType
}

func (p *ASTParser) ruleListHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if curNode == nil {
		return ast.NewReferenceNode("nil"), s.ListType{}
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	return p.ruleListExprSeqHandler(curNode)
}

func (p *ASTParser) ruleListExprSeqHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	elem, varType := p.ruleExprHandler(curNode)
	listType := s.ListType{Type: varType}
	curNode = curNode.next
	if curNode == nil {
		return ast.NewFunctionCallNode(ast.NativeFunction("1100"), []ast.Node{elem, ast.NewReferenceNode("nil")}), listType
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	secondElem, varType := p.ruleListExprSeqHandler(curNode)
	listType.AppendList(varType)
	return ast.NewFunctionCallNode(ast.NativeFunction("1100"), []ast.Node{elem, secondElem}), listType
}

func (p *ASTParser) ruleBooleanAtomHandler(node *node32) (ast.Node, s.Type) {
	value := string(p.buffer[node.begin:node.end])
	var boolValue bool
	switch value {
	case "true":
		boolValue = true
	case "false":
		boolValue = false
	}
	return ast.NewBooleanNode(boolValue), s.BooleanType
}

func (p *ASTParser) ruleTupleHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	var exprs []ast.Node
	var types []s.Type
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode != nil && curNode.pegRule == ruleAtomExpr {
			expr, varType := p.ruleAtomExprHandler(curNode)
			exprs = append(exprs, expr)
			types = append(types, varType)
			curNode = curNode.next
		}
		if curNode == nil {
			break
		}
	}
	if len(exprs) < 2 || len(exprs) > 22 {
		p.addError(fmt.Sprintf("invalid tuple len \"%d\"(allowed 2 to 22)", len(exprs)), node.token32)
		return nil, nil
	}
	return ast.NewFunctionCallNode(ast.NativeFunction(strconv.Itoa(1300+len(exprs)-2)), exprs), s.TupleType{Types: types}
}

func (p *ASTParser) ruleIfWithErrorHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if curNode.pegRule == ruleFailedIfWithoutElse {
		p.addError("If without else", curNode.token32)
		return nil, nil
	}
	curNode = curNode.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	cond, condType := p.ruleExprHandler(curNode)
	if condType != s.BooleanType {
		p.addError(fmt.Sprintf("Expression must be Boolean: \"%s\"", condType), curNode.token32)
	}
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	thenExpr, thenType := p.ruleExprHandler(curNode)
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	elseExpr, elseType := p.ruleExprHandler(curNode)
	var resType s.Type
	if !thenType.Comp(elseType) {
		union := s.UnionType{Types: []s.Type{}}
		union.AppendType(thenType)
		union.AppendType(elseType)
		if len(union.Types) == 1 {
			resType = union.Types[0]
		} else {
			resType = union
		}
	} else {
		resType = thenType
	}
	return ast.NewConditionalNode(cond, thenExpr, elseExpr), resType
}

func (p *ASTParser) ruleGettableExprHandler(node *node32) (ast.Node, s.Type) {
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
	for {
		if curNode == nil {
			break
		}
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		switch curNode.pegRule {
		case ruleListAccess:
			listNode := curNode.up
			if l, ok := varType.(s.ListType); !ok {
				p.addError(fmt.Sprintf("type must be List but is %s", varType.String()), listNode.token32)
			} else {
				if listNode.pegRule == rule_ {
					listNode = listNode.next
				}
				var index ast.Node
				var indexType s.Type
				switch listNode.pegRule {
				case ruleExpr:
					index, indexType = p.ruleExprHandler(listNode)
				case ruleIdentifier:
					index, indexType = p.ruleIdentifierHandler(listNode)
				}
				if !indexType.Comp(s.IntType) {
					p.addError(fmt.Sprintf("index type must be Int but is %s", indexType.String()), listNode.token32)
				}
				expr = ast.NewFunctionCallNode(ast.NativeFunction("401"), []ast.Node{expr, index})
				varType = l.Type
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
			if t, ok := varType.(s.TupleType); !ok {
				p.addError(fmt.Sprintf("type must be Tuple but is %s", varType.String()), curNode.token32)
			} else {
				tupleIndexStr := string(p.buffer[curNode.begin:curNode.end])
				indexStr := strings.TrimPrefix(tupleIndexStr, "_")
				index, err := strconv.ParseInt(indexStr, 10, 64)
				if err != nil {
					p.addError(fmt.Sprintf("error in parsing tuple index: %s", err), curNode.token32)
				}
				if index < 1 || index > int64(len(t.Types)) {
					p.addError(fmt.Sprintf("tuple index must be less then %d", len(t.Types)), curNode.token32)
				}
				expr = ast.NewPropertyNode(tupleIndexStr, expr)
				varType = t.Types[index-1]
			}
		}
		curNode = curNode.next
	}
	return expr, varType
}

func (p *ASTParser) ruleIdentifierHandler(node *node32) (ast.Node, s.Type) {
	name := string(p.buffer[node.begin:node.end])
	v, ok := p.currentStack.GetVariable(name)
	if !ok {
		p.addError(fmt.Sprintf("Variable \"%s\" doesnt't exist", name), node.token32)
		return nil, nil
	}
	return ast.NewReferenceNode(name), v.Type
}

func (p *ASTParser) ruleParExprHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	return p.ruleExprHandler(curNode)
}

func (p *ASTParser) ruleFunctionCallHandler(node *node32, firstArg ast.Node, firstArgType s.Type) (ast.Node, s.Type) {
	curNode := node.up
	funcName := string(p.buffer[curNode.begin:curNode.end])
	nameNode := curNode
	curNode = curNode.next

	if curNode != nil && curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode != nil && curNode.pegRule == rule_ {
		curNode = curNode.next
	}

	argsNodes, argsTypes, astNodes := p.ruleArgSeqHandler(curNode)

	if firstArg != nil {
		argsNodes = append([]ast.Node{firstArg}, argsNodes...)
		argsTypes = append([]s.Type{firstArgType}, argsTypes...)
	}
	var funcSign s.FunctionParams
	funcSign, ok := p.currentStack.GetFunc(funcName)
	if !ok {
		funcSign, ok = p.stdFuncs.Get(funcName, argsTypes)
		if !ok {
			funcSign, ok = p.stdObjects.GetConstruct(funcName, argsTypes)
			if !ok {
				p.addError(fmt.Sprintf("undefined function: \"%s\"", funcName), nameNode.token32)
				return nil, nil
			}
		}
		return ast.NewFunctionCallNode(funcSign.ID, argsNodes), funcSign.ReturnType
	}
	if len(argsNodes) != len(funcSign.Arguments) {
		p.addError(fmt.Sprintf("Function \"%s\" requires %d arguments, but %d are provided", funcName, len(funcSign.Arguments), len(argsNodes)), curNode.token32)
		return nil, funcSign.ReturnType
	}
	for i := range argsNodes {
		if funcSign.Arguments[i].Comp(argsTypes[i]) {
			continue
		}
		p.addError(fmt.Sprintf("Cannot use type %s as the type %v", argsTypes[i], funcSign.Arguments[i]), astNodes[i].token32)
	}
	return ast.NewFunctionCallNode(funcSign.ID, argsNodes), funcSign.ReturnType
}

func (p *ASTParser) ruleArgSeqHandler(node *node32) ([]ast.Node, []s.Type, []*node32) {
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
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		curNode = curNode.up
	}
	return resultNodes, resultTypes, resultAstNodes
}

func (p *ASTParser) ruleIdentifierAccessHandler(node *node32, obj ast.Node, objType s.Type) (ast.Node, s.Type) {
	curNode := node
	fieldName := string(p.buffer[curNode.begin:curNode.end])

	fieldType, ok := p.stdObjects.GetField(objType, fieldName)
	if !ok {
		p.addError(fmt.Sprintf("type %s has not filed %s", objType.String(), fieldName), curNode.token32)
		return nil, nil
	}
	return ast.NewPropertyNode(fieldName, obj), fieldType

}

func (p *ASTParser) ruleBlockHandler(node *node32) (ast.Node, s.Type) {
	p.currentStack = NewVarStack(p.currentStack)
	curNode := node.up
	var decls []ast.Node
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode.pegRule == ruleDeclaration {
			expr, _ := p.ruleDeclarationHandler(curNode.up)
			decls = append(decls, expr...)
			curNode = curNode.next
		}
		if curNode.pegRule == ruleExpr {
			break
		}
	}
	block, varType := p.ruleExprHandler(curNode)
	expr := block
	for i := len(decls) - 1; i >= 0; i-- {
		expr = decls[i]
		expr.SetBlock(block)
		block = expr
	}
	p.currentStack = p.currentStack.up
	return expr, varType
}

func (p *ASTParser) ruleFuncHandler(node *node32) (ast.Node, s.Type, []s.Type) {
	p.currentStack = NewVarStack(p.currentStack)
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	funcName := string(p.buffer[curNode.begin:curNode.end])
	if _, ok := p.currentStack.GetFunc(funcName); ok {
		p.addError(fmt.Sprintf("function \"%s\" exist", funcName), curNode.token32)
	}
	if ok := p.stdFuncs.Check(funcName); ok {
		p.addError(fmt.Sprintf("function \"%s\" exist in standart library", funcName), curNode.token32)
	}
	curNode = curNode.next
	var argsNode *node32
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
			continue
		}
		if curNode.pegRule == ruleFuncArgSeq {
			argsNode = curNode
			curNode = curNode.next
		}
		if curNode.pegRule == ruleExpr {
			break
		}
	}
	argsNames, argsTypes := p.ruleFuncArgSeqHandler(argsNode)
	expr, varType := p.ruleExprHandler(curNode)
	p.currentStack.up.PushFunc(s.FunctionParams{
		ID:         ast.UserFunction(funcName),
		Arguments:  argsTypes,
		ReturnType: varType,
	})
	p.currentStack = p.currentStack.up
	return &ast.FunctionDeclarationNode{
		Name:                funcName,
		Arguments:           argsNames,
		Body:                expr,
		Block:               nil,
		InvocationParameter: "",
	}, varType, argsTypes
}

func (p *ASTParser) ruleFuncArgSeqHandler(node *node32) ([]string, []s.Type) {
	if node == nil {
		return []string{}, []s.Type{}
	}
	// TODO(anton): add Tuple
	curNode := node.up
	argName, argType := p.ruleFuncArgHandler(curNode)
	curNode = curNode.next
	argsNames := []string{argName}
	argsTypes := []s.Type{argType}
	if curNode == nil {
		return argsNames, argsTypes
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	nextArgsNames, nextArgsTypes := p.ruleFuncArgSeqHandler(curNode)
	return append(argsNames, nextArgsNames...), append(argsTypes, nextArgsTypes...)
}

func (p *ASTParser) ruleFuncArgHandler(node *node32) (string, s.Type) {
	curNode := node.up
	argName := string(p.buffer[curNode.begin:curNode.end])
	// TODO(anton): check if name exist args
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	argType := p.ruleTypesHandler(curNode)
	p.currentStack.PushVariable(s.Variable{
		Name: argName,
		Type: argType,
	})
	return argName, argType
}

func (p *ASTParser) ruleTypesHandler(node *node32) s.Type {
	curNode := node.up
	var T s.Type
	switch curNode.pegRule {
	case ruleGenericType:
		T = p.ruleGenericTypeHandler(curNode)
	case ruleTupleType:
		T = p.ruleTupleTypeHandler(curNode)
	case ruleType:
		// TODO: check Types
		T = s.SimpleType{Type: string(p.buffer[curNode.begin:curNode.end])}
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
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	T = p.ruleTypesHandler(curNode)
	if T == nil {
		return nil
	}
	resType.AppendType(T)
	return resType
}

func (p *ASTParser) ruleTupleTypeHandler(node *node32) s.Type {
	curNode := node.up
	var tupleTypes []s.Type
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode.pegRule == ruleTypes {
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

func (p *ASTParser) ruleGenericTypeHandler(node *node32) s.Type {
	curNode := node.up
	name := string(p.buffer[curNode.begin:curNode.end])
	if name != "List" {
		p.addError(fmt.Sprintf("Generig type should be List, but \"%s\"", name), curNode.token32)
		return nil
	}
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	T := p.ruleTypesHandler(curNode)
	if T == nil {
		return T
	}
	return s.ListType{Type: T}
}

func (p *ASTParser) parseAnnotatedFunc(node *node32) {
	curNode := node
	for {
		if curNode != nil && curNode.pegRule == ruleAnnotatedFunc {
			p.ruleAnnotatedFunc(curNode.up)
			curNode = curNode.next
		}
		if curNode != nil && curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode == nil || (curNode.pegRule != rule_ && curNode.pegRule != ruleAnnotatedFunc) {
			break
		}
	}
}

func (p *ASTParser) loadMeta(name string, argsTypes []s.Type) error {
	res := meta.Function{
		Name:      name,
		Arguments: []meta.Type{},
	}
	for _, t := range argsTypes {
		T := t
		isList := false
		if list, ok := t.(s.ListType); ok {
			T = list.Type
			isList = true
		}
		isValid := true
		var argType meta.Type
		if simpleType, ok := T.(s.SimpleType); ok {
			switch simpleType.Type {
			case "String":
				argType = meta.String
			case "Int":
				argType = meta.Int
			case "Boolean":
				argType = meta.Boolean
			case "ByteVector":
				argType = meta.Bytes
			default:
				isValid = false
			}
		}
		if !isValid {
			return errors.Errorf("Unexpected type in callable args : %s", t.String())
		} else {
			if isList {
				argType = meta.ListType{Inner: argType}
			}
			res.Arguments = append(res.Arguments, argType)
		}
	}
	p.Tree.Meta.Functions = append(p.Tree.Meta.Functions, res)
	return nil
}

func (p *ASTParser) ruleAnnotatedFunc(node *node32) {
	p.currentStack = NewVarStack(p.currentStack)
	curNode := node
	annotation, _ := p.ruleAnnotationSeqHandler(curNode)
	if annotation == "" {
		return
	}
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	expr, _, types := p.ruleFuncHandler(curNode)
	switch annotation {
	case "Callable":
		p.Tree.Functions = append(p.Tree.Functions, expr)
		f := expr.(*ast.FunctionDeclarationNode)
		err := p.loadMeta(f.Name, types)
		if err != nil {
			p.addError(fmt.Sprintf("%s", err), node.token32)
		}
	case "Verifier":
		if p.Tree.Verifier != nil {
			p.addError("More than one Verifier", node.token32)
		}
		f := expr.(*ast.FunctionDeclarationNode)
		p.Tree.Verifier = f.Body
		if len(types) != 0 {
			p.addError("Verifyer must not have arguments", node.token32)
		}
	}
	p.currentStack = p.currentStack.up
	// TODO(anton): add callable with specific flag in stack
}

func (p *ASTParser) ruleAnnotationSeqHandler(node *node32) (string, string) {
	curNode := node.up
	annotationNode := curNode.up
	name := string(p.buffer[annotationNode.begin:annotationNode.end])
	if name != "Callable" && name != "Verifier" {
		p.addError(fmt.Sprintf("Undefinded annotation \"%s\"", name), annotationNode.token32)
		return "", ""
	}
	if annotationNode.pegRule == rule_ {
		annotationNode = annotationNode.next
	}
	if annotationNode.pegRule == rule_ {
		annotationNode = annotationNode.next
	}
	annotationNode = annotationNode.next.up
	varName := string(p.buffer[annotationNode.begin:annotationNode.end])
	annotationNode = annotationNode.next
	if annotationNode != nil {
		p.addError(fmt.Sprintf("More then one variable in annotation: \"%s\"", name), annotationNode.token32)
	}
	curNode = curNode.next
	if curNode != nil {
		p.addError("More then one annotation", curNode.token32)
	}

	switch name {
	case "Callable":
		p.currentStack.PushVariable(s.Variable{
			Name: varName,
			Type: s.SimpleType{Type: "Invocation"},
		})
	case "Verifier":
		p.currentStack.PushVariable(s.Variable{
			Name: varName,
			Type: s.SimpleType{Type: "Transaction"},
		})
	}
	return name, varName
}

func (p *ASTParser) ruleMatchHandler(node *node32) (ast.Node, s.Type) {
	p.currentStack = NewVarStack(p.currentStack)
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	expr, varType := p.ruleExprHandler(curNode)
	curNode = curNode.next
	possibleTypes := s.UnionType{Types: []s.Type{}}

	if t, ok := varType.(s.UnionType); ok {
		possibleTypes = t
	} else {
		possibleTypes.AppendType(varType)
	}
	var matchName string

	if lastMatchName, ok := p.currentStack.GetLastMatchName(); ok {
		matchNumStr := strings.TrimPrefix(lastMatchName, "$match")
		matchNum, err := strconv.ParseInt(matchNumStr, 10, 64)
		if err != nil {
			p.addError(fmt.Sprintf("Unexpected error in parse int: %s", err), token32{})
		}
		matchName = fmt.Sprintf("$match%d", matchNum)
	} else {
		matchName = "$match0"
	}
	p.currentStack.PushVariable(s.Variable{
		Name: matchName,
		Type: varType,
	})
	var conds, trueStates []ast.Node
	var defaultCase ast.Node
	unionRetType := s.UnionType{Types: []s.Type{}}
	for {
		if curNode != nil && curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode != nil && curNode.pegRule == ruleCase {
			// new stack for each case
			p.currentStack = NewVarStack(p.currentStack)
			cond, trueState, varType := p.ruleCaseHandle(curNode, matchName, possibleTypes)
			if trueState == nil {
				if defaultCase != nil {
					p.addError("Match should have at most one default case", curNode.token32)
				}
				defaultCase = cond
			} else {
				conds = append(conds, cond)
				trueStates = append(trueStates, trueState)
			}
			unionRetType.AppendType(varType)
			curNode = curNode.next
			p.currentStack = p.currentStack.up
		}
		if curNode == nil {
			break
		}
	}
	if defaultCase == nil {
		p.addError("Match should have default case", node.token32)
		return nil, nil
	}
	falseState := defaultCase
	for i := len(conds) - 1; i >= 0; i-- {
		falseState = ast.NewConditionalNode(conds[i], trueStates[i], falseState)
	}
	p.currentStack = p.currentStack.up
	var retType s.Type
	if len(unionRetType.Types) == 1 {
		retType = unionRetType.Types[0]
	} else {
		retType = unionRetType
	}
	return ast.NewAssignmentNode(matchName, expr, falseState), retType
}

func (p *ASTParser) ruleCaseHandle(node *node32, matchName string, possibleTypes s.UnionType) (ast.Node, ast.Node, s.Type) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	statementNode := curNode
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
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
	}
	return cond, trueState, blockType
}

func (p *ASTParser) ruleValuePatternHandler(node *node32, matchName string, possibleTypes s.UnionType) (ast.Node, ast.Node) {
	curNode := node.up
	nameNode := curNode
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	t := p.ruleTypesHandler(curNode)

	if !possibleTypes.Comp(t) {
		p.addError(fmt.Sprintf("Matching not exhaustive: possibleTypes are \"%s\", while matched are \"%s\"", possibleTypes.String(), t.String()), curNode.token32)
	}

	var decl ast.Node = nil

	if nameNode.pegRule != rulePlaceholder {
		name := string(p.buffer[nameNode.begin:nameNode.end])
		if _, ok := p.currentStack.GetVariable(name); ok {
			p.addError(fmt.Sprintf("Variable %s already exist", name), nameNode.token32)
		}
		p.currentStack.PushVariable(s.Variable{
			Name: name,
			Type: t,
		})
		decl = ast.NewAssignmentNode(name, ast.NewReferenceNode(matchName), nil)
	}

	if u, ok := t.(s.UnionType); ok {
		var checks []ast.Node
		for _, unionType := range u.Types {
			checks = append(checks, ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewReferenceNode(matchName), ast.NewStringNode(unionType.String())}))
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

func (p *ASTParser) ruleObjectPatternHandler(node *node32, matchName string, possibleTypes s.UnionType) (ast.Node, []ast.Node) {
	curNode := node.up
	structName := string(p.buffer[curNode.begin:curNode.end])
	if !p.stdObjects.IsExist(structName) {
		p.addError(fmt.Sprintf("Object with this name %s doesn't exist", structName), curNode.token32)
		return nil, nil
	}
	if !possibleTypes.Comp(s.SimpleType{Type: structName}) {
		p.addError(fmt.Sprintf("Matching not exhaustive: possibleTypes are \"%s\", while matched are \"%s\"", possibleTypes.String(), structName), curNode.token32)
		return nil, nil
	}
	curNode = curNode.next

	var exprs []ast.Node
	var shadowDeclarations []ast.Node
	exprs = append(exprs, ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewReferenceNode(matchName), ast.NewStringNode(structName)}))
	shadowDeclarations = append(shadowDeclarations, ast.NewAssignmentNode(matchName, ast.NewReferenceNode(matchName), nil))
	for {
		if curNode == nil {
			break
		}
		expr, decl, newNode := p.ruleObjectFieldsPatternHandler(curNode, matchName, structName)
		curNode = newNode
		for {
			if curNode == nil || curNode.pegRule == ruleObjectFieldsPattern {
				break
			}
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

func (p *ASTParser) ruleObjectFieldsPatternHandler(node *node32, matchName string, structName string) (ast.Node, ast.Node, *node32) {
	curNode := node.up
	fieldName := string(p.buffer[curNode.begin:curNode.end])
	t, ok := p.stdObjects.GetField(s.SimpleType{Type: structName}, fieldName)
	if !ok {
		p.addError(fmt.Sprintf("Object %s doesn't has field %s", structName, fieldName), curNode.token32)
		return nil, nil, curNode
	}
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	switch curNode.pegRule {
	case ruleIdentifier:
		name := string(p.buffer[curNode.begin:curNode.end])
		if _, ok := p.currentStack.GetVariable(name); ok {
			p.addError(fmt.Sprintf("Variable %s exist", name), curNode.token32)
			return nil, nil, curNode
		}
		p.currentStack.PushVariable(s.Variable{
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
		if !exprType.Comp(t) {
			p.addError(fmt.Sprintf("Can't match inferred types: field %s has type %s, but %s provided", fieldName, t.String(), exprType.String()), curNode.token32)
			return nil, nil, curNode
		}
		return ast.NewFunctionCallNode(ast.NativeFunction("0"), []ast.Node{
			expr,
			ast.NewPropertyNode(fieldName, ast.NewReferenceNode(matchName)),
		}), nil, curNode
	}
	return nil, nil, nil
}

func (p *ASTParser) ruleTuplePatternHandler(node *node32, matchName string, possibleTypes s.UnionType) (ast.Node, []ast.Node) {
	curNode := node.up
	var exprs []ast.Node
	var varsTypes []s.Type
	var shadowDeclarations []ast.Node
	cnt := 0
	for {
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
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
	}
	tupleType := s.TupleType{Types: varsTypes}
	if !possibleTypes.Comp(tupleType) {
		p.addError(fmt.Sprintf("Matching not exhaustive: possibleTypes are \"%s\", while matched are \"%s\"", possibleTypes.String(), tupleType), curNode.token32)
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

func (p *ASTParser) ruleTupleValuesPatternHandler(node *node32, matchName string, cnt int, possibleTypes s.UnionType) (ast.Node, []ast.Node, s.Type) {
	curNode := node.up
	var expr ast.Node
	var varType s.Type
	var shadowDeclarations []ast.Node
	switch curNode.pegRule {
	case ruleValuePattern:
		curNode = curNode.up
		nameNode := curNode
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		varType = p.ruleTypesHandler(curNode)

		expr = ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewPropertyNode("_"+strconv.Itoa(cnt+1), ast.NewReferenceNode(matchName)), ast.NewStringNode(varType.String())})
		if nameNode.pegRule != rulePlaceholder {
			name := string(p.buffer[nameNode.begin:nameNode.end])
			p.currentStack.PushVariable(s.Variable{
				Name: name,
				Type: varType,
			})
			shadowDeclarations = append(shadowDeclarations, ast.NewAssignmentNode(name, ast.NewPropertyNode("_"+strconv.Itoa(cnt+1), ast.NewReferenceNode(matchName)), nil))
		}
	case ruleIdentifier:
		name := string(p.buffer[curNode.begin:curNode.end])

		for _, t := range possibleTypes.Types {
			if tuple, ok := t.(s.TupleType); ok {
				if cnt >= len(tuple.Types) {
					continue
				}
				varType = tuple.Types[cnt]
			}
		}

		p.currentStack.PushVariable(s.Variable{
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

func (p *ASTParser) ruleFoldMacroHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	// parse num in fold macro
	value := string(p.buffer[curNode.begin:curNode.end])
	if strings.Contains(value, "_") {
		value = strings.ReplaceAll(value, "_", "")
	}
	iterNum, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		p.addError(fmt.Sprintf("failing to parse Integer: %s", err), curNode.token32)
	}
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}

	arr, arrVarType := p.ruleExprHandler(curNode)
	var elemType s.Type
	if l, ok := arrVarType.(s.ListType); !ok {
		p.addError(fmt.Sprintf("first argument in fold mast be List, but found %s", arrVarType.String()), curNode.token32)
	} else {
		elemType = l.Type
	}
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}

	start, startVarType := p.ruleExprHandler(curNode)
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}

	funcName := string(p.buffer[curNode.begin:curNode.end])
	funcSign, ok := p.currentStack.GetFunc(funcName)
	if !ok {
		p.addError(fmt.Sprintf("undefined function: \"%s\"", funcName), curNode.token32)
		return nil, nil
	}
	if len(funcSign.Arguments) != 2 {
		if !funcSign.Arguments[0].Comp(startVarType) || !funcSign.Arguments[1].Comp(elemType) {
			p.addError(fmt.Sprintf("Can't find suitable function %s(%s, %s)", funcName, s.ListType{}.String(), startVarType.String()), curNode.token32)
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
	return block, startVarType
}
