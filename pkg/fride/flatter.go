package fride

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func flatten(tree *Tree) (string, error) {
	f := newFlatter()
	sb := new(strings.Builder)
	err := f.flatten(sb, tree.Verifier)
	if err != nil {
		return "", err
	}
	sb.WriteString("RET")
	patches := make(map[string]string)
	for i, d := range f.tail {
		// Append label
		sb.WriteString(" ")
		sb.WriteString("[")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("]")
		sb.WriteString(" ")

		// Write code
		sb.WriteString(d.code())

		// Add return
		sb.WriteString("RET")

		// Add patch info
		for _, id := range d.ids() {
			patches[id] = strconv.Itoa(i)
		}
	}
	code := sb.String()
	for k, v := range patches {
		code = strings.ReplaceAll(code, k, v)
	}
	return code, nil
}

type declaration interface {
	code() string
	ids() []string
}

type variable struct {
	name   string
	sb     *strings.Builder
	usages []string
	local  bool
}

func newVariable(name string) *variable {
	return &variable{
		name:   name,
		sb:     new(strings.Builder),
		usages: make([]string, 0),
	}
}

func (v *variable) code() string {
	if v.sb != nil {
		return v.sb.String()
	}
	return ""
}

func (v *variable) ids() []string {
	return v.usages
}

type function struct {
	name  string
	sb    *strings.Builder
	calls []string
	args  []string
}

func newFunction(name string, args []string) *function {
	return &function{
		name:  name,
		sb:    new(strings.Builder),
		calls: make([]string, 0),
		args:  args,
	}
}

func (f *function) code() string {
	if f.sb != nil {
		return f.sb.String()
	}
	return ""
}

func (f *function) ids() []string {
	return f.calls
}

type flatter struct {
	variables []*variable
	functions []*function
	tail      []declaration
	count     int
}

func newFlatter() *flatter {
	return &flatter{
		variables: make([]*variable, 0),
		functions: make([]*function, 0),
		tail:      make([]declaration, 0),
	}
}

func (f *flatter) flatten(sb *strings.Builder, node Node) error {
	switch n := node.(type) {
	case *LongNode:
		sb.WriteString("LONG(")
		sb.WriteString(strconv.Itoa(int(n.Value)))
		sb.WriteString(")")
	case *BytesNode:
		sb.WriteString("BYTES(")
		sb.WriteString(base64.StdEncoding.EncodeToString(n.Value))
		sb.WriteString(")")
	case *StringNode:
		sb.WriteString("STRING(")
		sb.WriteString(n.Value)
		sb.WriteString(")")
	case *BooleanNode:
		if n.Value {
			sb.WriteString("TRUE")
		} else {
			sb.WriteString("FALSE")
		}
	case *ConditionalNode:
		err := f.flatten(sb, n.Condition)
		if err != nil {
			return err
		}

		sb.WriteString("? ")

		err = f.flatten(sb, n.TrueExpression)
		if err != nil {
			return err
		}

		sb.WriteString(": ")

		err = f.flatten(sb, n.FalseExpression)
		if err != nil {
			return err
		}

	case *AssignmentNode:
		err := f.pushv(n.Name, n.Expression)
		if err != nil {
			return err
		}

		err = f.flatten(sb, n.Block)
		if err != nil {
			return err
		}
		err = f.popv()
		if err != nil {
			return err
		}

	case *ReferenceNode:
		d, err := f.vlookup(n.Name)
		if err != nil {
			return err
		}
		if d.local {
			sb.WriteString("LREF(")
			sb.WriteString(n.Name)
			sb.WriteString(") ")
		} else {
			sb.WriteString("REF(")
			sb.WriteString(n.Name)
			sb.WriteString(", ")
			usage := f.use(d)
			sb.WriteString(usage)
			sb.WriteString(")")
		}

	case *FunctionDeclarationNode:
		err := f.pushf(n.Name, n.Arguments, n.Body)
		if err != nil {
			return err
		}
		err = f.flatten(sb, n.Block)
		if err != nil {
			return err
		}
		err = f.popf()
		if err != nil {
			return err
		}

	case *FunctionCallNode:
		for _, a := range n.Arguments {
			err := f.flatten(sb, a)
			if err != nil {
				return err
			}
		}
		d, err := f.flookup(n.Name)
		if err != nil {
			sb.WriteString("CALL(")
			sb.WriteString(n.Name)
			sb.WriteString(")")
		} else {
			sb.WriteString("LCALL(")
			sb.WriteString(n.Name)
			sb.WriteString(", ")
			call := f.call(d)
			sb.WriteString(call)
			sb.WriteString(")")
		}
	case *PropertyNode:
		return errors.New("not implemented")
	default:
		return errors.Errorf("unexpected node type '%T'", node)
	}
	if str := sb.String(); str[len(str)-1] != ' ' {
		sb.WriteRune(' ')
	}
	return nil
}

func (f *flatter) pushv(name string, node Node) error {
	d := newVariable(name)
	f.variables = append(f.variables, d)
	err := f.flatten(d.sb, node)
	if err != nil {
		return err
	}
	return nil
}

func (f *flatter) popv() error {
	var d *variable
	l := len(f.variables)
	d, f.variables = f.variables[l-1], f.variables[:l-1]
	f.tail = append([]declaration{d}, f.tail...) //Argh!
	return nil
}

func (f *flatter) use(d *variable) string {
	id := "@" + d.name + "#" + strconv.Itoa(f.count)
	d.usages = append(d.usages, id)
	f.count++
	return id
}

func (f *flatter) vlookup(name string) (*variable, error) {
	fl := len(f.functions)
	if fl > 0 {
		fd := f.functions[len(f.functions)-1]
		for _, a := range fd.args {
			if name == a {
				return &variable{
					name:   name,
					sb:     nil,
					usages: nil,
					local:  true,
				}, nil
			}
		}
	}
	for i := len(f.variables) - 1; i >= 0; i-- {
		if f.variables[i].name == name {
			return f.variables[i], nil
		}
	}
	return nil, errors.Errorf("variable '%s' is not declared", name)
}

func (f *flatter) pushf(name string, args []string, node Node) error {
	d := newFunction(name, args)
	f.functions = append(f.functions, d)
	for _, a := range args {
		d.sb.WriteString("LSTORE(")
		d.sb.WriteString(a)
		d.sb.WriteString(") ")
	}
	err := f.flatten(d.sb, node)
	if err != nil {
		return err
	}
	return nil
}

func (f *flatter) popf() error {
	var d *function
	l := len(f.functions)
	d, f.functions = f.functions[l-1], f.functions[:l-1]
	f.tail = append([]declaration{d}, f.tail...) //Argh!
	return nil
}

func (f *flatter) call(d *function) string {
	id := "@" + d.name + "#" + strconv.Itoa(f.count)
	d.calls = append(d.calls, id)
	f.count++
	return id
}

func (f *flatter) flookup(name string) (*function, error) {
	for i := len(f.functions) - 1; i >= 0; i-- {
		if f.functions[i].name == name {
			return f.functions[i], nil
		}
	}
	return nil, errors.Errorf("function '%s' is not declared", name)
}
