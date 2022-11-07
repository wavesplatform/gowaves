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
	if curNode != nil && curNode.pegRule == rule_ {
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
		version, err := strconv.ParseInt(dirValue, 10, 8)
		if err != nil {
			p.addError(fmt.Sprintf("failed to parse version \"%s\" : %s", dirValue, err), dirValueNode.token32)
			break
		}
		if version > 6 || version < 1 {
			p.addError(fmt.Sprintf("invalid %s \"%s\"", stdlibVersionDirectiveName, dirValue), dirValueNode.token32)
		}
		p.Tree.LibVersion = ast.LibraryVersion(byte(version))
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
		expr, varType := p.ruleFuncHandler(node)
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
	p.currentStack.PushVariable(Variable{
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
		p.addError("Number of Identifiers must be <= tuple length", curNode.token32)
		return nil, nil
	}
	var resExpr []ast.Node
	var resTypes []s.Type
	var tupleName string
	switch e := expr.(type) {
	case *ast.FunctionCallNode:
		tupleName = "$t0" + strconv.FormatUint(uint64(node.begin), 10) + strconv.FormatUint(uint64(node.end), 10)
		p.currentStack.PushVariable(Variable{
			Name: tupleName,
			Type: varType,
		})
		resExpr = append(resExpr, &ast.AssignmentNode{
			Name:       tupleName,
			Expression: expr,
		})
		resTypes = append(resTypes, varType)
	case *ast.ReferenceNode:
		tupleName = e.Name
	}
	for i, name := range varNames {
		resExpr = append(resExpr, &ast.AssignmentNode{
			Name: name,
			Expression: &ast.PropertyNode{
				Name:   "_" + strconv.Itoa(i+1),
				Object: &ast.ReferenceNode{Name: tupleName},
			},
		})
		resTypes = append(resTypes, tuple.Types[i])
		p.currentStack.PushVariable(Variable{
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
		if _, ok := nextVarType.(s.ListType); !ok {
			p.addError(fmt.Sprintf("expression must be \"List\" but \"%s\"", varType), curNode.token32)
			return nil, nil
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
		expr, varType = p.ruleMatchHandler(curNode)
	case ruleConst:
		expr, varType = p.ruleConstHandler(curNode)
	case ruleList:
		expr, varType = p.ruleListHandler(curNode)
	case ruleTuple:
		expr, varType = p.ruleTupleHandler(curNode)
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
	value := string(p.buffer[node.begin:node.end])
	return ast.NewStringNode(value), s.StringType
}

func (p *ASTParser) ruleByteVectorHandler(node *node32) (ast.Node, s.Type) {
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

func (p *ASTParser) ruleListHandler(node *node32) (ast.Node, s.Type) {
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
		if curNode.pegRule == ruleAtomExpr {
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
		union := s.UnionType{Types: map[string]s.Type{}}
		union.AppendType(thenType)
		union.AppendType(elseType)
		resType = union
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
		expr, varType = p.ruleFunctionCallHandler(curNode)
	case ruleIdentifier:
		expr, varType = p.ruleIdentifierHandler(curNode)
	case ruleList:
		expr, varType = p.ruleListHandler(curNode)
	case ruleTuple:
		expr, varType = p.ruleTupleHandler(curNode)

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
			return nil, nil
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
	argType := p.ruleTypesHandler(curNode)
	p.currentStack.PushVariable(Variable{
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

	resType := s.UnionType{Types: map[string]s.Type{}}
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
		return nil, nil
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
	return nil, nil
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

func (p *ASTParser) ruleMatchHandler(node *node32) (ast.Node, s.Type) {
	p.currentStack = NewVarStack(p.currentStack)
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	expr, varType := p.ruleExprHandler(curNode)
	curNode = curNode.next
	possibleTypes := map[string]s.Type{}

	if t, ok := varType.(s.UnionType); ok {
		possibleTypes = t.Types
	} else {
		possibleTypes[varType.String()] = varType
	}
	matchName := "$match0"
	var conds, trueStates []ast.Node
	var defaultCase ast.Node
	retType := s.UnionType{Types: map[string]s.Type{}}
	for {
		if curNode != nil && curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode != nil && curNode.pegRule == ruleCase {
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
			retType.AppendType(varType)
			curNode = curNode.next
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
	p.currentStack = p.globalStack
	return ast.NewAssignmentNode(matchName, expr, falseState), retType
}

func (p *ASTParser) ruleCaseHandle(node *node32, matchName string, possibleTypes map[string]s.Type) (ast.Node, ast.Node, s.Type) {
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
	block, blockType := p.ruleBlockHandler(curNode)
	var cond, trueState ast.Node
	switch statementNode.pegRule {
	case ruleValuePattern:
		cond, trueState = p.ruleValuePatternHandler(statementNode, matchName, possibleTypes)
		if trueState == nil {
			trueState = block
		} else {
			trueState.SetBlock(block)
		}
	case ruleTuplePattern:
		ifCond, decls := p.ruleTuplePatternHandler(statementNode, matchName, possibleTypes)
		cond = ifCond
		if decls == nil {
			trueState = block
		} else {
			var expr ast.Node
			for i := len(decls) - 1; i >= 0; i-- {
				decls[i].SetBlock(expr)
				expr = decls[i]
			}
			expr.SetBlock(block)
			trueState = expr
		}
	case ruleObjectPattern:
	case rulePlaceholder:
		return block, nil, blockType
	}
	return cond, trueState, blockType
}

func (p *ASTParser) ruleValuePatternHandler(node *node32, matchName string, possibleTypes map[string]s.Type) (ast.Node, ast.Node) {
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

	if u, ok := t.(s.UnionType); ok {
		for typeName := range u.Types {
			if _, ok := possibleTypes[typeName]; !ok {
				p.addError(fmt.Sprintf("Matching not exhaustive: possibleTypes are \"%s\", while matched are \"%s\"", u.String(), typeName), curNode.token32)
			}
		}
	} else {
		if _, ok := possibleTypes[t.String()]; !ok {
			p.addError(fmt.Sprintf("Matching not exhaustive: possibleTypes are \"%s\", while matched are \"%s\"", u.String(), t.String()), curNode.token32)
		}
	}

	if nameNode.pegRule == rulePlaceholder {
		return ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewReferenceNode(matchName), ast.NewStringNode(t.String())}), nil
	}
	name := string(p.buffer[nameNode.begin:nameNode.end])
	return ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewReferenceNode(matchName), ast.NewStringNode(t.String())}),
		ast.NewAssignmentNode(name, ast.NewReferenceNode(matchName), nil)
}

func (p *ASTParser) ruleTuplePatternHandler(node *node32, matchName string, possibleTypes map[string]s.Type) (ast.Node, []ast.Node) {
	curNode := node.up.up
	var exprs []ast.Node
	var varsTypes []s.Type
	var shadowDeclarations []ast.Node
	cnt := 0
	for {
		if curNode == nil {
			break
		}
		expr, decl, t := p.ruleTupleValuesPatternHandler(node, matchName, possibleTypes, cnt)
		exprs = append(exprs, expr)
		varsTypes = append(varsTypes, t)
		shadowDeclarations = append(shadowDeclarations, decl...)
		cnt++
	}
	tupleType := s.TupleType{Types: varsTypes}
	eq := false
	for _, t := range possibleTypes {
		if t.Comp(tupleType) {
			eq = true
			break
		}
	}
	if !eq {
		p.addError(fmt.Sprintf("Matching not exhaustive: possibleTypes are \"%s\", while matched are \"%s\"", possibleTypes, tupleType), curNode.token32)
	}
	var cond ast.Node
	setLast := false
	setPlaceHolder := false
	for i := len(exprs) - 1; i >= 0; i-- {
		if cond == nil {
			setPlaceHolder = true
			continue
		}
		if !setLast {
			cond = exprs[i]
			setLast = true
		}
		cond = ast.NewConditionalNode(exprs[i], cond, ast.NewBooleanNode(false))
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

func (p *ASTParser) ruleTupleValuesPatternHandler(node *node32, matchName string, possibleTypes map[string]s.Type, cnt int) (ast.Node, []ast.Node, s.Type) {
	curNode := node.up
	var expr ast.Node
	var varType s.Type
	var shadowDeclarations []ast.Node
	switch curNode.pegRule {
	case ruleValuePattern:
		curNode = node.up
		nameNode := curNode
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		t := p.ruleTypesHandler(curNode)

		expr = ast.NewFunctionCallNode(ast.NativeFunction("1"), []ast.Node{ast.NewPropertyNode("_"+strconv.Itoa(cnt+1), ast.NewReferenceNode(matchName)), ast.NewStringNode(t.String())})
		if nameNode.pegRule != rulePlaceholder {
			name := string(p.buffer[nameNode.begin:nameNode.end])
			shadowDeclarations = append(shadowDeclarations, ast.NewAssignmentNode(name, ast.NewReferenceNode(matchName), nil))
		}
	case rulePlaceholder:
		// skip and return nil
		break
	case ruleExpr:
		expr, varType = p.ruleExprHandler(curNode)
		expr = ast.NewFunctionCallNode(ast.NativeFunction("0"), []ast.Node{expr, ast.NewPropertyNode("_"+strconv.Itoa(cnt+1), ast.NewReferenceNode(matchName))})
	case ruleConst:
		expr, varType = p.ruleConstHandler(curNode)
		expr = ast.NewFunctionCallNode(ast.NativeFunction("0"), []ast.Node{expr, ast.NewPropertyNode("_"+strconv.Itoa(cnt+1), ast.NewReferenceNode(matchName))})
	case ruleGettableExpr:
		// TODO(anton)
		//expr, varType = p.ruleGettableExprHandler(curNode)
		//expr = ast.NewFunctionCallNode(ast.NativeFunction("0"), []ast.Node{expr, ast.NewPropertyNode("_"+strconv.Itoa(cnt+1), ast.NewReferenceNode(matchName))})
	}
	return expr, shadowDeclarations, varType
}

func (p *ASTParser) ruleScriptRootHandler(node *node32) {
}
