package compiler

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	s "github.com/wavesplatform/gowaves/pkg/ride/compiler/signatures"
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
}

func NewASTParser(node *node32, buffer []rune) ASTParser {
	return ASTParser{
		node: node,
		Tree: &ast.Tree{
			Declarations: []ast.Node{},
			Functions:    []ast.Node{},
		},
		buffer:      buffer,
		ErrorsList:  []error{},
		globalStack: NewVarStack(nil),
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
	if curNode.pegRule == rule_ {
		curNode = node.next
	}
	if curNode != nil && curNode.pegRule == ruleDeclaration {
		curNode = p.parseDeclarations(curNode)
	}
	if curNode.pegRule == rule_ {
		curNode = node.next
	}
	if curNode != nil && curNode.pegRule == ruleAnnotatedFunc {
		curNode = p.parseAnnotatedFunc(curNode)
	}
	_ = curNode // TODO: This line added to evade linter error, remove it later
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
		p.checkDirectiveCnt(node, stdlibVersionDirectiveName, directiveCnt)
		version, err := strconv.Atoi(dirValue)
		if err != nil {
			p.addError(fmt.Sprintf("failed to parse version \"%s\" : %s", dirValue, err), dirValueNode.token32)
			break
		}
		if version > 6 {
			p.addError(fmt.Sprintf("invalid %s \"%s\"", stdlibVersionDirectiveName, dirValue), dirValueNode.token32)
		}
		p.Tree.LibVersion = ast.LibraryVersion(version)
	case contentTypeDirectiveName:
		p.checkDirectiveCnt(node, contentTypeDirectiveName, directiveCnt)
		switch dirValue {
		case dappValueName:
			p.Tree.ContentType = ast.ContentTypeApplication
		case expressionValueName:
			p.Tree.ContentType = ast.ContentTypeExpression
		default:
			p.addError(fmt.Sprintf("Undefined %s value: \"%s\"", contentTypeDirectiveName, dirValue), dirNameNode.token32)
		}

	case scriptTypeDirectiveName:
		p.checkDirectiveCnt(node, scriptTypeDirectiveName, directiveCnt)
		switch dirValue {
		case accountValueName:
		case assetValueName:
		case libraryValueName:
			break
			// TODO
		default:
			p.addError(fmt.Sprintf("Undefined %s value: \"%s\"", scriptTypeDirectiveName, dirValue), dirNameNode.token32)
		}
	case importDirectiveName:
		break
		// TODO
	default:
		p.addError(fmt.Sprintf("Undefined directive: \"%s\"", dirName), dirNameNode.token32)
	}

}

func (p *ASTParser) checkDirectiveCnt(node *node32, name string, directiveCnt map[string]int) {
	if val, ok := directiveCnt[name]; ok && val == 1 {
		p.addError(fmt.Sprintf("more than one %s directive", name), node.token32)
	} else {
		directiveCnt[name] = 1
	}
}

func (p *ASTParser) parseDeclarations(node *node32) *node32 {
	curNode := node
	for {
		if curNode != nil && curNode.pegRule == ruleDeclaration {
			expr, _ := p.ruleDeclarationHandler(curNode.up)
			p.Tree.Declarations = append(p.Tree.Declarations, expr)
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

func (p *ASTParser) ruleDeclarationHandler(node *node32) (ast.Node, s.Type) {
	switch node.pegRule {
	case ruleVariable:
		return p.ruleVariableHandler(node)
	case ruleFunc:
		return p.ruleFuncHandler(node)
	default:
		panic(errors.Errorf("wrong type of rule in Declaration: %s", rul3s[node.pegRule]))
	}
}

func (p *ASTParser) ruleVariableHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	// get Variable Name
	varName := string(p.buffer[curNode.begin:curNode.end])
	if _, ok := p.currentStack.GetVariable(varName); ok {
		p.addError(fmt.Sprintf("variable \"%s\" is exist", varName), curNode.token32)
		return nil, s.Undefined
	}
	curNode = curNode.next

	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	expr, varType := p.ruleExprHandler(curNode)
	expr = ast.NewAssignmentNode(varName, expr, nil)
	p.currentStack.PushVariable(Variable{
		Name: varName,
		Type: varType,
	})
	return expr, varType
}

func (p *ASTParser) ruleExprHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up.up
	expr, varType := p.ruleAndOpAtomHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
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
		nextExpr, _ := p.ruleAndOpAtomHandler(curNode)
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
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		// skip andOp
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, _ := p.ruleEqualityGroupOpAtomHandler(curNode)
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
		var funcId string
		if curNode.up.pegRule == ruleEqOp {
			funcId = "0"
		} else {
			funcId = "!="
		}
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, _ := p.ruleCompareGroupOpAtomHandler(curNode)
		expr = ast.NewFunctionCallNode(ast.NativeFunction(funcId), []ast.Node{expr, nextExpr})
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, s.BooleanType
}

func (p *ASTParser) ruleCompareGroupOpAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	expr, varType := p.ruleConsOpAtomHandler(curNode)
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
		nextExpr, _ := p.ruleConsOpAtomHandler(curNode)
		if operator == ruleGtOp {
			expr = ast.NewFunctionCallNode(ast.NativeFunction("102"), []ast.Node{expr, nextExpr})
		} else if operator == ruleGeOp {
			expr = ast.NewFunctionCallNode(ast.NativeFunction("103"), []ast.Node{expr, nextExpr})
		} else if operator == ruleLtOp {
			expr = ast.NewFunctionCallNode(ast.NativeFunction("102"), []ast.Node{nextExpr, expr})
		} else {
			expr = ast.NewFunctionCallNode(ast.NativeFunction("103"), []ast.Node{nextExpr, expr})
		}
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, varType
}

func (p *ASTParser) ruleConsOpAtomHandler(node *node32) (ast.Node, s.Type) {
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
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, nextVarType := p.ruleSumGroupOpAtomHandler(curNode)
		if !s.ListOfAny.Comp(varType) {
			p.addError(fmt.Sprintf("expression must be \"List\" but \"%s\"", varType), curNode.token32)
			return nil, s.Undefined
		}
		resListType.AppendList(nextVarType)
		expr = ast.NewFunctionCallNode(ast.NativeFunction("cons"), []ast.Node{expr, nextExpr})
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, resListType
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
		var funcId string
		if curNode.up.pegRule == ruleSumOp {
			funcId = "100"
		} else {
			funcId = "101"
		}
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, _ := p.ruleMultGroupOpAtomHandler(curNode)
		expr = ast.NewFunctionCallNode(ast.NativeFunction(funcId), []ast.Node{expr, nextExpr})
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, s.IntType
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
		var funcId string
		if curNode.up.pegRule == ruleMulOp {
			funcId = "104"
		} else if curNode.up.pegRule == ruleDivOp {
			funcId = "105"
		} else {
			funcId = "106"
		}
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, _ := p.ruleAtomExprHandler(curNode)
		expr = ast.NewFunctionCallNode(ast.NativeFunction(funcId), []ast.Node{expr, nextExpr})
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, s.IntType
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
	case ruleGettableExpr:
		expr, varType = p.ruleGettableExprHandler(curNode)
	case ruleIfWithError:
		expr, varType = p.ruleIfWithErrorHandler(curNode)
	case ruleMatch:
		break
	case ruleConst:
		expr, varType = p.ruleConstAtomHandler(curNode)
	}
	if unaryOp == ruleNegativeOp {
		expr = ast.NewFunctionCallNode(ast.NativeFunction("-"), []ast.Node{expr})
	} else if unaryOp == ruleNotOp {
		expr = ast.NewFunctionCallNode(ast.NativeFunction("!"), []ast.Node{expr})
	} else if unaryOp == rulePositiveOp {
		_ = 1 // TODO: This line added to evade linter error, remove it later
		// TODO: check type == int
	}
	return expr, varType
}

func (p *ASTParser) ruleConstAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	var expr ast.Node
	var varType s.Type
	switch curNode.pegRule {
	case ruleInteger:
		expr, varType = p.ruleIntegerAtomHandler(curNode)
	case ruleString:
		expr, varType = p.ruleStringAtomHandler(curNode)
	case ruleByteVector:
		expr, varType = p.ruleByteVectorAtomHandler(curNode)
	case ruleBoolean:
		expr, varType = p.ruleBooleanAtomHandler(curNode)
	case ruleList:
		expr, varType = p.ruleListAtomHandler(curNode)
	}
	return expr, varType
}

func (p *ASTParser) ruleIntegerAtomHandler(node *node32) (ast.Node, s.Type) {
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

func (p *ASTParser) ruleStringAtomHandler(node *node32) (ast.Node, s.Type) {
	value := string(p.buffer[node.begin:node.end])
	return ast.NewStringNode(value), s.StringType
}

func (p *ASTParser) ruleByteVectorAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	var err error
	var value []byte
	valueWithBase := string(p.buffer[curNode.begin:curNode.end])
	// get value from baseXX'VALUE'
	valueInBase := valueWithBase[len("baseXX'") : len(valueWithBase)-1]
	switch node.up.pegRule {
	case ruleBase16:
		_, err = hex.Decode(value, []byte(valueInBase))
	case ruleBase58:
		value, err = base58.Decode(valueInBase)
	case ruleBase64:
		_, err = base64.StdEncoding.Decode(value, []byte(valueInBase))
	}
	if err != nil {
		p.addError(fmt.Sprintf("failing to parse ByteVector: %s", err), curNode.token32)
	}
	return ast.NewBytesNode(value), s.ByteVectorType
}

func (p *ASTParser) ruleListAtomHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	return p.ruleListExprSeqHandler(curNode)
}

func (p *ASTParser) ruleListExprSeqHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	listType := s.UnionType{Types: map[string]s.Type{}}
	elem, varType := p.ruleExprHandler(curNode)
	listType.AppendType(varType)
	curNode = curNode.next
	if curNode == nil {
		return ast.NewFunctionCallNode(ast.NativeFunction("1100"), []ast.Node{elem, nil}), s.ListType{Type: listType}
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	secondElem, varType := p.ruleListExprSeqHandler(curNode)
	listType.AppendType(varType)
	return ast.NewFunctionCallNode(ast.NativeFunction("1100"), []ast.Node{elem, secondElem}), s.ListType{Type: listType}
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

func (p *ASTParser) ruleIfWithErrorHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if curNode.pegRule == ruleFailedIfWithoutElse {
		p.addError("If without else", curNode.token32)
		return nil, s.Undefined
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
	if thenType != elseType {
		p.addError(fmt.Sprintf("Expression in the then and else must be similar: \"%s\" \"%s\"", thenType, elseType), curNode.token32)
	}
	return ast.NewConditionalNode(cond, thenExpr, elseExpr), thenType
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
		expr, varType = p.ruleFunctionCallHandler(curNode)
	case ruleIdentifier:
	}
	return expr, varType
}

func (p *ASTParser) ruleParExprHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	return p.ruleExprHandler(curNode)
}

type FuncArgument struct {
	Node    ast.Node
	ASTNode *node32
	Type    s.Type
}

func (p *ASTParser) ruleFunctionCallHandler(node *node32) (ast.Node, s.Type) {
	curNode := node.up
	funcName := string(p.buffer[curNode.begin:curNode.end])
	funcSign, ok := p.currentStack.GetFunc(funcName)
	if !ok {
		funcSign, ok = s.Funcs.Funcs[funcName]
		if !ok {
			p.addError(fmt.Sprintf("udefined function: \"%s\"", funcName), curNode.token32)
			return nil, s.Undefined
		}
	}
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	args := p.ruleArgSeqHandler(curNode)
	if len(args) != len(funcSign.Arguments) {
		p.addError(fmt.Sprintf("Function \"%s\" requires %d arguments, but %d are provided", funcName, len(funcSign.Arguments), len(args)), curNode.token32)
		return nil, funcSign.ReturnType
	}
	var funcArgs []ast.Node
	for i, arg := range args {
		if funcSign.Arguments[i].Comp(arg.Type) {
			funcArgs = append(funcArgs, arg.Node)
			continue
		}
		p.addError(fmt.Sprintf("Cannot use type %s as the type %v", arg.Type, funcSign.Arguments[i]), arg.ASTNode.token32)
	}
	return ast.NewFunctionCallNode(ast.NativeFunction(funcSign.ID), funcArgs), funcSign.ReturnType
}

func (p *ASTParser) ruleArgSeqHandler(node *node32) []FuncArgument {
	if node.pegRule != ruleExprSeq {
		return []FuncArgument{}
	}
	curNode := node.up
	var result []FuncArgument
	for {
		expr, varType := p.ruleExprHandler(curNode)
		result = append(result, FuncArgument{
			Node:    expr,
			ASTNode: curNode,
			Type:    varType,
		})
		curNode = curNode.next
		if curNode == nil {
			break
		}
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		curNode = curNode.up
	}
	return result
}

func (p *ASTParser) ruleBlockHandler(node *node32) (ast.Node, s.Type) {
	p.currentStack = NewVarStack(p.currentStack)
	curNode := node.up
	expr, varType := p.ruleBlockDeclHandler(curNode)
	p.currentStack = p.currentStack.up
	return expr, varType
}

func (p *ASTParser) ruleBlockDeclHandler(node *node32) (ast.Node, s.Type) {
	curNode := node
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	var expr ast.Node
	var varType s.Type
	if curNode.pegRule == ruleDeclaration {
		expr, varType = p.ruleDeclarationHandler(curNode.up)
		if expr == nil {
			return nil, varType
		}
		curNode = curNode.next
	} else {
		return p.ruleExprHandler(curNode)
	}
	blockExpr, varType := p.ruleBlockDeclHandler(curNode)
	expr.SetBlock(blockExpr)
	return expr, varType

}

func (p *ASTParser) ruleFuncHandler(node *node32) (ast.Node, s.Type) {
	p.currentStack = NewVarStack(p.currentStack)
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	funcName := string(p.buffer[curNode.begin:curNode.end])
	if _, ok := p.currentStack.GetFunc(funcName); ok {
		p.addError(fmt.Sprintf("function \"%s\" exist", funcName), curNode.token32)
	}
	if _, ok := s.Funcs.Funcs[funcName]; ok {
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
		ID:         funcName,
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
	}, varType
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
	argType := s.ParseType(string(p.buffer[curNode.begin:curNode.end]))
	p.currentStack.PushVariable(Variable{
		Name: argName,
		Type: argType,
	})
	return argName, argType
}

func (p *ASTParser) parseAnnotatedFunc(node *node32) *node32 {
	curNode := node
	for {
		if curNode != nil && curNode.pegRule == ruleAnnotatedFunc {
			expr, _ := p.ruleAnnotatedFunc(curNode.up)
			p.Tree.Functions = append(p.Tree.Declarations, expr)
			curNode = curNode.next
		}
		if curNode != nil && curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode == nil || (curNode.pegRule != rule_ && curNode.pegRule != ruleAnnotatedFunc) {
			break
		}
	}
	return curNode
}

func (p *ASTParser) ruleAnnotatedFunc(node *node32) (ast.Node, s.Type) {
	curNode := node
	annotation, _ := p.ruleAnnotationSeqHandler(curNode)
	if annotation == "" {
		return nil, s.Undefined
	}
	curNode = curNode.next
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	expr, _ := p.ruleFuncHandler(curNode)
	switch annotation {
	case "Callable":
		p.Tree.Functions = append(p.Tree.Functions, expr)
	case "Verifier":
		if p.Tree.Verifier != nil {
			p.addError("More than one Verifier", node.token32)
		}
		f := expr.(*ast.FunctionDeclarationNode)
		p.Tree.Verifier = f.Body
	}

	// TODO(anton): add callable with specific flag in stack
	return nil, s.Undefined
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
	return name, varName
}

func (p *ASTParser) ruleScriptRootHandler(node *node32) {
}
