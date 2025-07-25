package stdlib

// Code generated by peg -output=type.peg.go type.peg DO NOT EDIT.

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleMainRule
	ruleTypes
	rule_
	ruleEOF
	ruleType
	ruleGenericType
	ruleTupleType
)

var rul3s = [...]string{
	"Unknown",
	"MainRule",
	"Types",
	"_",
	"EOF",
	"Type",
	"GenericType",
	"TupleType",
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(w io.Writer, pretty bool, buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for range depth {
				fmt.Fprintf(w, " ")
			}
			rule := rul3s[node.pegRule]
			quote := strconv.Quote(string(([]rune(buffer)[node.begin:node.end])))
			if !pretty {
				fmt.Fprintf(w, "%v %v\n", rule, quote)
			} else {
				fmt.Fprintf(w, "\x1B[36m%v\x1B[m %v\n", rule, quote)
			}
			if node.up != nil {
				print(node.up, depth+1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

func (node *node32) Print(w io.Writer, buffer string) {
	node.print(w, false, buffer)
}

func (node *node32) PrettyPrint(w io.Writer, buffer string) {
	node.print(w, true, buffer)
}

type tokens32 struct {
	tree []token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(os.Stdout, buffer)
}

func (t *tokens32) WriteSyntaxTree(w io.Writer, buffer string) {
	t.AST().Print(w, buffer)
}

func (t *tokens32) PrettyPrintSyntaxTree(buffer string) {
	t.AST().PrettyPrint(os.Stdout, buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	tree, i := t.tree, int(index)
	if i >= len(tree) {
		t.tree = append(tree, token32{pegRule: rule, begin: begin, end: end})
		return
	}
	tree[i] = token32{pegRule: rule, begin: begin, end: end}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type Types struct {
	Buffer string
	buffer []rune
	rules  [8]func() bool
	parse  func(rule ...int) error
	reset  func()
	Pretty bool
	tokens32
}

func (p *Types) Parse(rule ...int) error {
	return p.parse(rule...)
}

func (p *Types) Reset() {
	p.reset()
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *Types
	max token32
}

func (e *parseError) Error() string {
	tokens, err := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		err += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return err
}

func (p *Types) PrintSyntaxTree() {
	if p.Pretty {
		p.tokens32.PrettyPrintSyntaxTree(p.Buffer)
	} else {
		p.tokens32.PrintSyntaxTree(p.Buffer)
	}
}

func (p *Types) WriteSyntaxTree(w io.Writer) {
	p.tokens32.WriteSyntaxTree(w, p.Buffer)
}

func (p *Types) SprintSyntaxTree() string {
	var bldr strings.Builder
	p.WriteSyntaxTree(&bldr)
	return bldr.String()
}

func Pretty(pretty bool) func(*Types) error {
	return func(p *Types) error {
		p.Pretty = pretty
		return nil
	}
}

func Size(size int) func(*Types) error {
	return func(p *Types) error {
		p.tokens32 = tokens32{tree: make([]token32, 0, size)}
		return nil
	}
}
func (p *Types) Init(options ...func(*Types) error) error {
	var (
		max                  token32
		position, tokenIndex uint32
		buffer               []rune
	)
	for _, option := range options {
		err := option(p)
		if err != nil {
			return err
		}
	}
	p.reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.reset()

	_rules := p.rules
	tree := p.tokens32
	p.parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 MainRule <- <(Types EOF)> */
		func() bool {
			position0, tokenIndex0 := position, tokenIndex
			{
				position1 := position
				if !_rules[ruleTypes]() {
					goto l0
				}
				if !_rules[ruleEOF]() {
					goto l0
				}
				add(ruleMainRule, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 Types <- <((GenericType / TupleType / Type) (_ '|' _ Types)?)> */
		func() bool {
			position2, tokenIndex2 := position, tokenIndex
			{
				position3 := position
				{
					position4, tokenIndex4 := position, tokenIndex
					if !_rules[ruleGenericType]() {
						goto l5
					}
					goto l4
				l5:
					position, tokenIndex = position4, tokenIndex4
					if !_rules[ruleTupleType]() {
						goto l6
					}
					goto l4
				l6:
					position, tokenIndex = position4, tokenIndex4
					if !_rules[ruleType]() {
						goto l2
					}
				}
			l4:
				{
					position7, tokenIndex7 := position, tokenIndex
					if !_rules[rule_]() {
						goto l7
					}
					if buffer[position] != rune('|') {
						goto l7
					}
					position++
					if !_rules[rule_]() {
						goto l7
					}
					if !_rules[ruleTypes]() {
						goto l7
					}
					goto l8
				l7:
					position, tokenIndex = position7, tokenIndex7
				}
			l8:
				add(ruleTypes, position3)
			}
			return true
		l2:
			position, tokenIndex = position2, tokenIndex2
			return false
		},
		/* 2 _ <- <(' ' / '\t')*> */
		func() bool {
			{
				position10 := position
			l11:
				{
					position12, tokenIndex12 := position, tokenIndex
					{
						position13, tokenIndex13 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l14
						}
						position++
						goto l13
					l14:
						position, tokenIndex = position13, tokenIndex13
						if buffer[position] != rune('\t') {
							goto l12
						}
						position++
					}
				l13:
					goto l11
				l12:
					position, tokenIndex = position12, tokenIndex12
				}
				add(rule_, position10)
			}
			return true
		},
		/* 3 EOF <- <!.> */
		func() bool {
			position15, tokenIndex15 := position, tokenIndex
			{
				position16 := position
				{
					position17, tokenIndex17 := position, tokenIndex
					if !matchDot() {
						goto l17
					}
					goto l15
				l17:
					position, tokenIndex = position17, tokenIndex17
				}
				add(ruleEOF, position16)
			}
			return true
		l15:
			position, tokenIndex = position15, tokenIndex15
			return false
		},
		/* 4 Type <- <(([A-Z] / [a-z]) ([A-Z] / [a-z] / [0-9])*)> */
		func() bool {
			position18, tokenIndex18 := position, tokenIndex
			{
				position19 := position
				{
					position20, tokenIndex20 := position, tokenIndex
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l21
					}
					position++
					goto l20
				l21:
					position, tokenIndex = position20, tokenIndex20
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l18
					}
					position++
				}
			l20:
			l22:
				{
					position23, tokenIndex23 := position, tokenIndex
					{
						position24, tokenIndex24 := position, tokenIndex
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l25
						}
						position++
						goto l24
					l25:
						position, tokenIndex = position24, tokenIndex24
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l26
						}
						position++
						goto l24
					l26:
						position, tokenIndex = position24, tokenIndex24
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l23
						}
						position++
					}
				l24:
					goto l22
				l23:
					position, tokenIndex = position23, tokenIndex23
				}
				add(ruleType, position19)
			}
			return true
		l18:
			position, tokenIndex = position18, tokenIndex18
			return false
		},
		/* 5 GenericType <- <(Type _ '[' _ Types? _ ']')> */
		func() bool {
			position27, tokenIndex27 := position, tokenIndex
			{
				position28 := position
				if !_rules[ruleType]() {
					goto l27
				}
				if !_rules[rule_]() {
					goto l27
				}
				if buffer[position] != rune('[') {
					goto l27
				}
				position++
				if !_rules[rule_]() {
					goto l27
				}
				{
					position29, tokenIndex29 := position, tokenIndex
					if !_rules[ruleTypes]() {
						goto l29
					}
					goto l30
				l29:
					position, tokenIndex = position29, tokenIndex29
				}
			l30:
				if !_rules[rule_]() {
					goto l27
				}
				if buffer[position] != rune(']') {
					goto l27
				}
				position++
				add(ruleGenericType, position28)
			}
			return true
		l27:
			position, tokenIndex = position27, tokenIndex27
			return false
		},
		/* 6 TupleType <- <('(' _ Types _ (',' _ Types)+ _ ')')> */
		func() bool {
			position31, tokenIndex31 := position, tokenIndex
			{
				position32 := position
				if buffer[position] != rune('(') {
					goto l31
				}
				position++
				if !_rules[rule_]() {
					goto l31
				}
				if !_rules[ruleTypes]() {
					goto l31
				}
				if !_rules[rule_]() {
					goto l31
				}
				if buffer[position] != rune(',') {
					goto l31
				}
				position++
				if !_rules[rule_]() {
					goto l31
				}
				if !_rules[ruleTypes]() {
					goto l31
				}
			l33:
				{
					position34, tokenIndex34 := position, tokenIndex
					if buffer[position] != rune(',') {
						goto l34
					}
					position++
					if !_rules[rule_]() {
						goto l34
					}
					if !_rules[ruleTypes]() {
						goto l34
					}
					goto l33
				l34:
					position, tokenIndex = position34, tokenIndex34
				}
				if !_rules[rule_]() {
					goto l31
				}
				if buffer[position] != rune(')') {
					goto l31
				}
				position++
				add(ruleTupleType, position32)
			}
			return true
		l31:
			position, tokenIndex = position31, tokenIndex31
			return false
		},
	}
	p.rules = _rules
	return nil
}
