package main

import (
	"fmt"
	"strconv"

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
	node       *node32
	Tree       *ast.Tree
	buffer     []rune
	ErrorsList []error
}

func NewASTParser(node *node32, buffer []rune) ASTParser {
	return ASTParser{
		node:       node,
		Tree:       new(ast.Tree),
		buffer:     buffer,
		ErrorsList: []error{},
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
	}
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
	// get Directive key
	dirNameNode := curNode
	dirName := string(p.buffer[curNode.begin:curNode.end])
	curNode = curNode.next
	// skip WS
	curNode = curNode.next
	// get Directive key
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

func (p *ASTParser) ruleScriptRootHandler(node *node32) {
}
