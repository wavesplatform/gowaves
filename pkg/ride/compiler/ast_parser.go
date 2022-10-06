package compiler

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
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
	return fmt.Sprintf("((%d, %d), (%d, %d)): %s", e.begin.line, e.begin.symbol, e.end.line, e.end.symbol, e.msg)
}

type ASTParser struct {
	node   *node32
	Tree   *ast.Tree
	buffer []rune

	ErrorsList  []error
	globalStack VarStack
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
		globalStack: VarStack{},
	}
}

func (p *ASTParser) Parse() {
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
			p.ruleDeclarationHandler(curNode.up)
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

func (p *ASTParser) ruleDeclarationHandler(node *node32) {
	switch node.pegRule {
	case ruleVariable:
		p.ruleVariableHandler(node)
	case ruleFunc:
		break
	default:
		panic(errors.Errorf("wrong type of rule in Declaration: %s", rul3s[node.pegRule]))

	}
}

func (p *ASTParser) ruleVariableHandler(node *node32) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	// get Variable Name
	varName := string(p.buffer[curNode.begin:curNode.end])
	if _, ok := p.globalStack.GetVariable(varName); ok {
		p.addError(fmt.Sprintf("variable \"%s\" is exist", varName), curNode.token32)
		return
	}
	curNode = curNode.next

	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	expr, varType := p.ruleExprHandler(curNode)
	p.Tree.Declarations = append(p.Tree.Declarations, ast.NewAssignmentNode(varName, expr, nil))
	p.globalStack.Push(Variable{
		Name: varName,
		Type: varType,
	})
}

func (p *ASTParser) ruleExprHandler(node *node32) (ast.Node, string) {
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
	return expr, "Boolean"
}

func (p *ASTParser) ruleAndOpAtomHandler(node *node32) (ast.Node, string) {
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
	return expr, "Boolean"
}

func (p *ASTParser) ruleEqualityGroupOpAtomHandler(node *node32) (ast.Node, string) {
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
	return expr, "Boolean"
}

func (p *ASTParser) ruleCompareGroupOpAtomHandler(node *node32) (ast.Node, string) {
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

func (p *ASTParser) ruleConsOpAtomHandler(node *node32) (ast.Node, string) {
	curNode := node.up
	expr, varType := p.ruleSumGroupOpAtomHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return expr, varType
	}
	for {
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		curNode = curNode.next
		if curNode.pegRule == rule_ {
			curNode = curNode.next
		}
		nextExpr, _ := p.ruleSumGroupOpAtomHandler(curNode)
		expr = ast.NewFunctionCallNode(ast.NativeFunction("cons"), []ast.Node{expr, nextExpr})
		curNode = curNode.next
		if curNode == nil {
			break
		}
	}
	return expr, "List"
}

func (p *ASTParser) ruleSumGroupOpAtomHandler(node *node32) (ast.Node, string) {
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
	return expr, "Int"
}

func (p *ASTParser) ruleMultGroupOpAtomHandler(node *node32) (ast.Node, string) {
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
	return expr, "Int"
}

func (p *ASTParser) ruleAtomExprHandler(node *node32) (ast.Node, string) {
	curNode := node.up
	var unaryOp pegRule
	if curNode.pegRule == ruleUnaryOp {
		unaryOp = curNode.up.pegRule
		curNode = curNode.next
	}
	var expr ast.Node
	var varType string
	switch curNode.pegRule {
	case ruleFoldMacro:
	case ruleGettableExpr:
	case ruleIfWithError:
	case ruleMatch:
		break
	case ruleConstAtom:
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

func (p *ASTParser) ruleConstAtomHandler(node *node32) (ast.Node, string) {
	curNode := node.up
	var expr ast.Node
	var varType string
	switch curNode.pegRule {
	case ruleIntegerAtom:
		expr, varType = p.ruleIntegerAtomHandler(curNode)
	case ruleStringAtom:
		expr, varType = p.ruleStringAtomHandler(curNode)
	case ruleByteVectorAtom:
		expr, varType = p.ruleByteVectorAtomHandler(curNode)
	case ruleBooleanAtom:
		expr, varType = p.ruleBooleanAtomHandler(curNode)
	case ruleListAtom:
		expr, varType = p.ruleListAtomHandler(curNode)
	}
	return expr, varType
}

func (p *ASTParser) ruleIntegerAtomHandler(node *node32) (ast.Node, string) {
	value := string(p.buffer[node.begin:node.end])
	number, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		p.addError(fmt.Sprintf("failing to parse Integer: %s", err), node.token32)
	}
	return ast.NewLongNode(number), "Integer"
}

func (p *ASTParser) ruleStringAtomHandler(node *node32) (ast.Node, string) {
	value := string(p.buffer[node.begin:node.end])
	return ast.NewStringNode(value), "String"
}

func (p *ASTParser) ruleByteVectorAtomHandler(node *node32) (ast.Node, string) {
	curNode := node.up
	base, err := strconv.Atoi(string(p.buffer[curNode.begin:curNode.end]))
	if err == nil {
		p.addError(fmt.Sprintf("failing to parse base in ByteVector: %s", err), curNode.token32)
	}
	curNode = curNode.next
	var value []byte
	valueInBase := string(p.buffer[curNode.begin:curNode.end])
	switch base {
	case 16:
		_, err = hex.Decode(value, []byte(valueInBase))
	case 58:
		value, err = base58.Decode(valueInBase)
	case 64:
		_, err = base64.StdEncoding.Decode(value, []byte(valueInBase))
	}
	if err == nil {
		p.addError(fmt.Sprintf("failing to parse ByteVector: %s", err), curNode.token32)
	}
	return ast.NewBytesNode(value), "ByteVector"
}

func (p *ASTParser) ruleListAtomHandler(node *node32) (ast.Node, string) {
	curNode := node.up
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	return p.ruleExprSeqHandler(curNode)
}

func (p *ASTParser) ruleExprSeqHandler(node *node32) (ast.Node, string) {
	curNode := node.up
	elem, _ := p.ruleExprHandler(curNode)
	curNode = curNode.next
	if curNode == nil {
		return ast.NewFunctionCallNode(ast.NativeFunction("1100"), []ast.Node{elem, nil}), "List"
	}
	var secondElem ast.Node
	if curNode.pegRule == ruleExpr {
		secondElem, _ = p.ruleExprHandler(curNode)
		secondElem = ast.NewFunctionCallNode(ast.NativeFunction("1100"), []ast.Node{secondElem, nil})
		return ast.NewFunctionCallNode(ast.NativeFunction("1100"), []ast.Node{elem, secondElem}), "List"
	}
	if curNode.pegRule == rule_ {
		curNode = curNode.next
	}
	secondElem, _ = p.ruleExprSeqHandler(curNode)
	return ast.NewFunctionCallNode(ast.NativeFunction("1100"), []ast.Node{elem, secondElem}), "List"
}

func (p *ASTParser) ruleBooleanAtomHandler(node *node32) (ast.Node, string) {
	value := string(p.buffer[node.begin:node.end])
	var boolValue bool
	switch value {
	case "true":
		boolValue = true
	case "false":
		boolValue = false
	}
	return ast.NewBooleanNode(boolValue), "Boolean"
}

func (p *ASTParser) ruleScriptRootHandler(node *node32) {
}
