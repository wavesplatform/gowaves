package ast

import (
	"bytes"
	"fmt"
	"io"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const InstanceFieldName = "$instance"

type Callable func(Scope, Exprs) (Expr, error)

type Script struct {
	Version    int
	HasBlockV2 bool
	Verifier   Expr
	DApp       DApp
	dApp       bool
}

func (a *Script) HasVerifier() bool {
	if a.IsDapp() {
		return a.DApp.Verifier != nil
	}
	return a.Verifier != nil
}

func (a *Script) IsDapp() bool {
	return a.dApp
}

func protoArgToArgExpr(arg proto.Argument) (Expr, error) {
	switch a := arg.(type) {
	case *proto.IntegerArgument:
		return &LongExpr{a.Value}, nil
	case *proto.BooleanArgument:
		return &BooleanExpr{a.Value}, nil
	case *proto.StringArgument:
		return &StringExpr{a.Value}, nil
	case *proto.BinaryArgument:
		return &BytesExpr{a.Value}, nil
	default:
		return nil, errors.New("unknown argument type")
	}
}

func (a *Script) CallFunction(scheme proto.Scheme, state types.SmartState, tx *proto.InvokeScriptWithProofs, this, lastBlock Expr) (*proto.ScriptResult, error) {
	if !a.IsDapp() {
		return nil, errors.New("can't call Script.CallFunction on non DApp")
	}
	name := tx.FunctionCall.Name
	if name == "" && tx.FunctionCall.Default {
		name = "default"
	}
	fn, ok := a.DApp.CallableFuncs[name]
	if !ok {
		return nil, errors.Errorf("Callable function named '%s' not found", name)
	}
	invoke, err := BuildInvocation(scheme, tx)
	if err != nil {
		return nil, err
	}
	height, err := state.AddingBlockHeight()
	if err != nil {
		return nil, err
	}
	scope := NewScope(3, scheme, state)
	scope.SetThis(this)
	scope.SetLastBlockInfo(lastBlock)
	scope.SetHeight(height)

	// assign of global vars and function
	for _, expr := range a.DApp.Declarations {
		_, err = expr.Evaluate(scope)
		if err != nil {
			return nil, errors.Wrap(err, "Script.CallFunction")
		}
	}

	if len(fn.FuncDecl.Args) != len(tx.FunctionCall.Arguments) {
		return nil, errors.Errorf("invalid func '%s' args count, expected %d, got %d", fn.FuncDecl.Name, len(fn.FuncDecl.Args), len(tx.FunctionCall.Arguments))
	}
	// pass function arguments
	curScope := scope.Clone()
	for i, arg := range tx.FunctionCall.Arguments {
		argExpr, err := protoArgToArgExpr(arg)
		if err != nil {
			return nil, errors.Wrap(err, "Script.CallFunction")
		}
		curScope.AddValue(fn.FuncDecl.Args[i], argExpr)
	}
	// invocation type
	curScope.AddValue(fn.AnnotationInvokeName, invoke)

	rs, err := fn.FuncDecl.Body.Evaluate(curScope)
	if err != nil {
		return nil, errors.Wrap(err, "Script.CallFunction")
	}

	var resExpr *ScriptResultExpr
	switch t := rs.(type) {
	case *WriteSetExpr:
		resExpr = &ScriptResultExpr{WriteSet: t}
	case *TransferSetExpr:
		resExpr = &ScriptResultExpr{TransferSet: t}
	case *ScriptResultExpr:
		resExpr = t
	default:
		return nil, errors.Errorf("Script.CallFunction: unexpected result type '%T'", t)
	}
	return resExpr.ConvertToProto()
}

func (a *Script) Verify(scheme byte, state types.SmartState, object map[string]Expr, this, lastBlock Expr) (bool, error) {
	height, err := state.AddingBlockHeight()
	if err != nil {
		return false, err
	}
	if a.IsDapp() {
		if a.DApp.Verifier == nil {
			return false, errors.New("verify function not defined")
		}
		scope := NewScope(3, scheme, state)
		scope.SetThis(this)
		scope.SetLastBlockInfo(lastBlock)
		scope.SetHeight(height)

		fn := a.DApp.Verifier
		// pass function arguments
		curScope := scope //.Clone()
		// annotated tx type
		curScope.AddValue(fn.AnnotationInvokeName, NewObject(object))
		// here should be only assign of vars and function
		for _, expr := range a.DApp.Declarations {
			_, err = expr.Evaluate(curScope)
			if err != nil {
				return false, errors.Wrap(err, "Script.Verify")
			}
		}
		return evalAsBool(fn.FuncDecl.Body, curScope)
	} else {
		scope := NewScope(a.Version, scheme, state)
		scope.SetTransaction(object)
		scope.SetThis(this)
		scope.SetLastBlockInfo(lastBlock)
		scope.SetHeight(height)
		return evalAsBool(a.Verifier, scope)
	}
}

func evalAsBool(e Expr, s Scope) (bool, error) {
	rs, err := e.Evaluate(s)
	if err != nil {
		if _, ok := err.(Throw); ok {
			// maybe log error
			return false, nil
		}
		return false, err
	}
	b, ok := rs.(*BooleanExpr)
	if !ok {
		return false, errors.Errorf("expected evaluate return *BooleanExpr, but found %T", b)
	}
	return b.Value, nil
}

func (a *Script) Eval(s Scope) (bool, error) {
	return evalAsBool(a.Verifier, s)
}

type Expr interface {
	Write(io.Writer)
	Evaluate(Scope) (Expr, error)
	Eq(Expr) bool
	InstanceOf() string
}

type Exprs []Expr

func (a Exprs) Write(w io.Writer) {
	for _, expr := range a {
		expr.Write(w)
	}
}

func (a Exprs) Evaluate(s Scope) (Expr, error) {
	return a.EvaluateAll(s)
}

func (a Exprs) EvaluateAll(s Scope) (Exprs, error) {
	out := make(Exprs, len(a))
	for i, row := range a {
		rs, err := row.Evaluate(s)
		if err != nil {
			return nil, err
		}
		out[i] = rs
	}
	return out, nil
}

func (a Exprs) Eq(other Expr) bool {
	o, ok := other.(Exprs)
	if !ok {
		return false
	}
	if len(a) != len(o) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if !a[i].Eq(o[i]) {
			return false
		}
	}
	return true
}

func (a Exprs) InstanceOf() string {
	return "Exprs"
}

func NewExprs(e ...Expr) Exprs {
	return e
}

// will be calculated in future, with known Scope
type LazyValueExpr struct {
	Expr  Expr
	Scope Scope
}

func NewLazyValue(Expr Expr, Scope Scope) *LazyValueExpr {
	return &LazyValueExpr{
		Expr:  Expr,
		Scope: Scope,
	}
}

func (a *LazyValueExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "LazyValueExpr")
}

func (a *LazyValueExpr) Evaluate(Scope) (Expr, error) {
	return a.Expr.Evaluate(a.Scope)
}

func (a *LazyValueExpr) Eq(other Expr) bool {
	return false
}

func (a *LazyValueExpr) InstanceOf() string {
	return "LazyValue"
}

// get property from object
type Getable interface {
	Get(string) (Expr, error)
}

type Block struct {
	Let  *LetExpr
	Body Expr
}

func (a *Block) Write(w io.Writer) {
	a.Let.Write(w)
	_, _ = fmt.Fprintf(w, "\n")
	a.Body.Write(w)
}

func (a *Block) Evaluate(s Scope) (Expr, error) {
	s.AddValue(a.Let.Name, NewLazyValue(a.Let.Value, s))
	return a.Body.Evaluate(s.Clone())
}

func (a *Block) Eq(other Expr) bool {
	return false
}

func (a *Block) InstanceOf() string {
	return "Block"
}

type Declaration interface {
	Eval(s Scope)
}

type LetDeclaration struct {
	name string
	body Expr
}

func (a *LetDeclaration) Eval(s Scope) {
	s.AddValue(a.name, a.body)
}

type FuncDeclaration struct {
	Name string
	Args []string
	Body Expr
}

func (a *FuncDeclaration) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "FuncDeclaration")
}

func (a *FuncDeclaration) Evaluate(s Scope) (Expr, error) {
	s.AddValue(a.Name, NewFunctionWithScope(a.Args, a.Body, s.Clone()))
	return a, nil
}

func (a *FuncDeclaration) Eq(other Expr) bool {
	return false
}

func (a *FuncDeclaration) InstanceOf() string {
	return "FuncDeclaration"
}

type BlockV2 struct {
	Decl Expr
	Body Expr
}

func (a *BlockV2) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "BlockV2")
}

func (a *BlockV2) Evaluate(s Scope) (Expr, error) {
	_, err := a.Decl.Evaluate(s)
	if err != nil {
		return nil, err
	}
	return a.Body.Evaluate(s.Clone())
}

func (a *BlockV2) Eq(other Expr) bool {
	return false
}

func (a *BlockV2) InstanceOf() string {
	return "BlockV2"
}

type LetExpr struct {
	Name  string
	Value Expr
}

func (a *LetExpr) Evaluate(s Scope) (Expr, error) {
	s.AddValue(a.Name, a.Value)
	return a, nil
}

func (a *LetExpr) Eq(other Expr) bool {
	return false
}

func (a *LetExpr) InstanceOf() string {
	return "Let"
}

func (a *LetExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "let %s = ", a.Name)
	a.Value.Write(w)
}

func NewLet(name string, value Expr) *LetExpr {
	return &LetExpr{
		Name:  name,
		Value: value,
	}
}

type LongExpr struct {
	Value int64
}

func NewLong(value int64) *LongExpr {
	return &LongExpr{
		Value: value,
	}
}

func (a *LongExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "%d", a.Value)
}

func (a *LongExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *LongExpr) Eq(other Expr) bool {
	b, ok := other.(*LongExpr)
	if !ok {
		return false
	}
	return a.Value == b.Value
}

func (a *LongExpr) InstanceOf() string {
	return "Int"
}

type BooleanExpr struct {
	Value bool
}

func NewBoolean(value bool) *BooleanExpr {
	return &BooleanExpr{
		Value: value,
	}
}

func (a *BooleanExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *BooleanExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, a.Value)
}

func (a *BooleanExpr) Eq(other Expr) bool {
	b, ok := other.(*BooleanExpr)
	if !ok {
		return false
	}

	return a.Value == b.Value
}

func (a *BooleanExpr) InstanceOf() string {
	return "Boolean"
}

type FuncCallExpr struct {
	Func Expr
}

func (a *FuncCallExpr) Write(w io.Writer) {
	a.Func.Write(w)
}

func (a *FuncCallExpr) Evaluate(s Scope) (Expr, error) {
	return a.Func.Evaluate(s)
}

func (a *FuncCallExpr) Eq(other Expr) bool {
	return false
}

func (a *FuncCallExpr) InstanceOf() string {
	return "FuncCallExpr"
}

func NewFuncCall(f Expr) *FuncCallExpr {
	return &FuncCallExpr{
		Func: f,
	}
}

type FunctionCall struct {
	Name string
	Argc int
	Argv Exprs
}

func NewFunctionCall(name string, argv Exprs) *FunctionCall {
	return &FunctionCall{
		Name: name,
		Argc: len(argv),
		Argv: argv,
	}
}

func (a *FunctionCall) Write(w io.Writer) {
	if a.Name == "!=" {
		infix(w, " != ", a.Argv)
		return
	}
	writeFunction(w, a.Name, a.Argv)
}

func (a *FunctionCall) Evaluate(s Scope) (Expr, error) {
	e, ok := s.Value(a.Name)
	if !ok {
		return nil, errors.Errorf("evaluate user function: function named '%s' not found in scope", a.Name)
	}
	fn, ok := e.(*Function)
	if !ok {
		return nil, errors.Errorf("evaluate user function: expected value 'fn' to be *Function, found %T", e)
	}
	if fn.Argc != a.Argc {
		return nil, errors.Errorf("evaluate user function: function %s expects %d arguments, passed %d", a.Name, fn.Argc, a.Argc)
	}
	functionScope := s.Initial()
	if fn.Scope != nil {
		functionScope = fn.Scope.Clone()
	}
	for i := 0; i < a.Argc; i++ {
		evaluatedParam, err := a.Argv[i].Evaluate(s)
		if err != nil {
			return nil, errors.Wrapf(err, "evaluate user function: %s", a.Name)
		}
		functionScope.AddValue(fn.Argv[i], evaluatedParam)
		functionScope.setEvaluation(fn.Argv[i], evaluation{evaluatedParam, nil})
	}
	return fn.Evaluate(functionScope)
}

func (a *FunctionCall) Eq(other Expr) bool {
	return false
}

func (a *FunctionCall) InstanceOf() string {
	return "FunctionCall"
}

type Function struct {
	Argc  int
	Argv  []string
	Body  Expr
	Scope Scope
}

func (a *Function) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "Function")
}

func (a *Function) Evaluate(s Scope) (Expr, error) {
	return a.Body.Evaluate(s)
}

func (a *Function) Eq(other Expr) bool {
	return false
}

func (a *Function) InstanceOf() string {
	return "Function"
}

func NewFunctionWithScope(Argv []string, Body Expr, s Scope) *Function {
	return &Function{
		Argc:  len(Argv),
		Argv:  Argv,
		Body:  Body,
		Scope: s,
	}
}

func FunctionFromPredefined(c Callable, argc uint32) *Function {
	return &Function{
		Argc: int(argc),
		Argv: buildParams(argc),
		Body: &PredefFunction{
			argv: buildParams(argc),
			fn:   c,
		},
	}
}

func buildParams(argc uint32) []string {
	var out []string
	for i := uint32(0); i < argc; i++ {
		out = append(out, fmt.Sprintf("param%d", i))
	}
	return out
}

type PredefFunction struct {
	argv []string
	fn   Callable
}

func (a *PredefFunction) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "PredefFunction")
}

func (a *PredefFunction) Evaluate(s Scope) (Expr, error) {
	params := Params()
	for i := 0; i < len(a.argv); i++ {
		e, ok := s.Value(a.argv[i])
		if !ok {
			return nil, errors.Errorf("PredefFunction: param %s not found in scope", a.argv[i])
		}
		params = append(params, e)
	}
	return a.fn(s, params)
}

func (a *PredefFunction) Eq(other Expr) bool {
	return false
}

func (a *PredefFunction) InstanceOf() string {
	return "PredefFunction"
}

type RefExpr struct {
	Name string
}

func (a *RefExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, a.Name)
}

func (a *RefExpr) Evaluate(s Scope) (Expr, error) {
	c, ok := s.evaluation(a.Name)
	if ok {
		return c.expr, c.err
	}
	expr, ok := s.Value(a.Name)
	if !ok {
		return nil, errors.Errorf("RefExpr evaluate: not found expr by name '%s'", a.Name)
	}
	rs, err := expr.Evaluate(s)
	s.setEvaluation(a.Name, evaluation{rs, err})
	return rs, err
}

func (a *RefExpr) Eq(other Expr) bool {
	return false
}

func (a *RefExpr) InstanceOf() string {
	return "Ref"
}

type IfExpr struct {
	Condition Expr
	True      Expr
	False     Expr
}

func NewIf(cond, trueExpr, falseExpr Expr) *IfExpr {
	return &IfExpr{
		Condition: cond,
		True:      trueExpr,
		False:     falseExpr,
	}
}

func (a *IfExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "if ( ")
	a.Condition.Write(w)
	_, _ = fmt.Fprint(w, " ) { ")
	a.True.Write(w)
	_, _ = fmt.Fprint(w, " } else { ")
	a.False.Write(w)
	_, _ = fmt.Fprint(w, " }  ")
}

func (a *IfExpr) Evaluate(s Scope) (Expr, error) {
	cond, err := a.Condition.Evaluate(s)
	if err != nil {
		return nil, err
	}
	b, ok := cond.(*BooleanExpr)
	if !ok {
		return nil, errors.Errorf("IfExpr evaluate: expected bool in condition found %T", cond)
	}
	if b.Value {
		return a.True.Evaluate(s.Clone())
	} else {
		return a.False.Evaluate(s.Clone())
	}
}

func (a *IfExpr) Eq(other Expr) bool {
	return false
}

func (a *IfExpr) InstanceOf() string {
	return "If"
}

type BytesExpr struct {
	Value []byte
}

func NewBytes(b []byte) *BytesExpr {
	return &BytesExpr{
		Value: b,
	}
}

func (a *BytesExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "base58'", base58.Encode(a.Value), "'")
}

func (a *BytesExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *BytesExpr) Eq(other Expr) bool {
	switch o := other.(type) {
	case *Unit:
		return false
	case *BytesExpr:
		return bytes.Equal(a.Value, o.Value)
	default:
		return false
	}
}

func (a *BytesExpr) InstanceOf() string {
	return "ByteVector"
}

type InvalidAddressExpr BytesExpr

func (a *InvalidAddressExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "base58'", base58.Encode(a.Value), "'")
}

func (a *InvalidAddressExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *InvalidAddressExpr) Eq(other Expr) bool {
	switch o := other.(type) {
	case *Unit:
		return false
	case *BytesExpr:
		return bytes.Equal(a.Value, o.Value)
	case *AddressExpr:
		return bytes.Equal(a.Value, o[:])
	default:
		return false
	}
}

func (a *InvalidAddressExpr) InstanceOf() string {
	return "Address"
}

func (a *InvalidAddressExpr) Get(name string) (Expr, error) {
	switch name {
	case "bytes":
		return NewBytes(util.Dup(a.Value)), nil
	default:
		return nil, errors.Errorf("unknown fields '%s' on InvalidAddressExpr", name)
	}
}

type GetterExpr struct {
	Object Expr
	Key    string
}

func NewGetterExpr(object Expr, key string) *GetterExpr {
	return &GetterExpr{
		Object: object,
		Key:    key,
	}
}

func (a *GetterExpr) Write(w io.Writer) {
	a.Object.Write(w)
	_, _ = fmt.Fprint(w, ".", a.Key)
}

func (a *GetterExpr) Evaluate(s Scope) (Expr, error) {
	val, err := a.Object.Evaluate(s)
	if err != nil {
		return nil, errors.Wrapf(err, "GetterExpr Evaluate by key %s", a.Key)
	}
	switch obj := val.(type) {
	case Getable:
		e, err := obj.Get(a.Key)
		if err != nil {
			return nil, err
		}
		return e, nil
	case *Unit:
		return NewUnit(), nil
	default:
		return nil, errors.Errorf("GetterExpr Evaluate: expected value be Getable, got %T", val)
	}
}

func (a *GetterExpr) Eq(other Expr) bool {
	return false
}

func (a *GetterExpr) InstanceOf() string {
	return "Getter"
}

type ObjectExpr struct {
	fields map[string]Expr
}

func NewObject(fields map[string]Expr) *ObjectExpr {
	return &ObjectExpr{
		fields: fields,
	}
}

func (a *ObjectExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "object")
}

func (a *ObjectExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *ObjectExpr) Eq(other Expr) bool {
	b, ok := other.(*ObjectExpr)
	if !ok {
		return false
	}
	if len(a.fields) != len(b.fields) {
		return false
	}
	for k1, v1 := range a.fields {
		v2, ok := b.fields[k1]
		if !ok {
			return false
		}
		if !v1.Eq(v2) {
			return false
		}
	}
	return true
}

func (a *ObjectExpr) Get(name string) (Expr, error) {
	out, ok := a.fields[name]
	if !ok {
		return nil, errors.Errorf("ObjectExpr no such field %s", name)
	}
	return out, nil
}

func (a *ObjectExpr) InstanceOf() string {
	if s, ok := a.fields[InstanceFieldName].(*StringExpr); ok {
		return s.Value
	}
	return ""
}

type DataEntryExpr struct {
	fields object
}

func NewDataEntry(key string, value Expr) *DataEntryExpr {
	return &DataEntryExpr{fields: object{"key": NewString(key), "value": value}}
}

func (a *DataEntryExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "DataEntryExpr")
}

func (a *DataEntryExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *DataEntryExpr) Eq(other Expr) bool {
	return false
}

func (a *DataEntryExpr) InstanceOf() string {
	return "DataEntry"
}

func (a *DataEntryExpr) Get(name string) (Expr, error) {
	out, ok := a.fields[name]
	if !ok {
		return nil, errors.Errorf("ObjectExpr no such field %s", name)
	}
	return out, nil
}

func (a *DataEntryExpr) toProto() (proto.DataEntry, error) {
	keyExpr, err := a.Get("key")
	if err != nil {
		return nil, err
	}
	keyStrExpr, ok := keyExpr.(*StringExpr)
	if !ok {
		return nil, errors.New("invalid key type")
	}
	valExpr, err := a.Get("value")
	if err != nil {
		return nil, err
	}
	switch v := valExpr.(type) {
	case *LongExpr:
		return &proto.IntegerDataEntry{Key: keyStrExpr.Value, Value: v.Value}, nil
	case *BooleanExpr:
		return &proto.BooleanDataEntry{Key: keyStrExpr.Value, Value: v.Value}, nil
	case *BytesExpr:
		return &proto.BinaryDataEntry{Key: keyStrExpr.Value, Value: v.Value}, nil
	case *StringExpr:
		return &proto.StringDataEntry{Key: keyStrExpr.Value, Value: v.Value}, nil
	}
	return nil, errors.New("unknown value type")
}

type StringExpr struct {
	Value string
}

func NewString(value string) *StringExpr {
	return &StringExpr{
		Value: value,
	}
}

func (a *StringExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, `"`, a.Value, `"`)
}

func (a *StringExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *StringExpr) Eq(other Expr) bool {
	b, ok := other.(*StringExpr)
	if !ok {
		return false
	}

	return a.Value == b.Value
}

func (a *StringExpr) InstanceOf() string {
	return "String"
}

type AddressExpr proto.Address

func (a *AddressExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, proto.Address(*a).String())
}

func (a *AddressExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *AddressExpr) Eq(other Expr) bool {
	switch o := other.(type) {
	case *RecipientExpr:
		return o.Address != nil && bytes.Equal(a[:], o.Address.Bytes())
	case *AddressExpr:
		return bytes.Equal(a[:], o[:])
	case *BytesExpr:
		return bytes.Equal(a[:], o.Value)
	default:
		return false
	}
}

func (a *AddressExpr) InstanceOf() string {
	return "Address"
}

func (a *AddressExpr) Get(name string) (Expr, error) {
	switch name {
	case "bytes":
		return NewBytes(util.Dup(proto.Address(*a).Bytes())), nil
	default:
		return nil, errors.Errorf("unknown fields '%s' on AddressExpr", name)
	}
}

func (a *AddressExpr) Recipient() proto.Recipient {
	return proto.NewRecipientFromAddress(proto.Address(*a))
}

func NewAddressFromString(s string) (*AddressExpr, error) {
	addr, err := proto.NewAddressFromString(s)
	a := AddressExpr(addr)
	return &a, err
}

func NewAddressFromProtoAddress(a proto.Address) *AddressExpr {
	addr := AddressExpr(a)
	return &addr
}

type Unit struct {
}

func (a *Unit) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "Unit")
}

func (a *Unit) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *Unit) Eq(other Expr) bool {
	return other.InstanceOf() == a.InstanceOf()
}

func (a *Unit) InstanceOf() string {
	return "Unit"
}

func NewUnit() *Unit {
	return &Unit{}
}

type AliasExpr proto.Alias

func (a *AliasExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "Alias")
}

func (a *AliasExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *AliasExpr) Eq(other Expr) bool {
	switch o := other.(type) {
	case *RecipientExpr:
		return o.Alias != nil && proto.Alias(*a).String() == o.Alias.String()
	case *AliasExpr:
		return proto.Alias(*a).String() == proto.Alias(*o).String()
	default:
		return false
	}
}

func (a *AliasExpr) InstanceOf() string {
	return "Alias"
}

// Recipient interface
func (a *AliasExpr) Recipient() proto.Recipient {
	return proto.NewRecipientFromAlias(proto.Alias(*a))
}

func NewAliasFromProtoAlias(a proto.Alias) *AliasExpr {
	al := AliasExpr(a)
	return &al
}

func newObjectExprFromDataEntry(entry proto.DataEntry) (*ObjectExpr, error) {
	fields := map[string]Expr{"key": NewString(entry.GetKey())}
	switch e := entry.(type) {
	case *proto.IntegerDataEntry:
		fields["value"] = NewLong(e.Value)
	case *proto.BooleanDataEntry:
		fields["value"] = NewBoolean(e.Value)
	case *proto.BinaryDataEntry:
		fields["value"] = NewBytes(e.Value)
	case *proto.StringDataEntry:
		fields["value"] = NewString(e.Value)
	default:
		return nil, errors.Errorf("unsupported data entry type '%T'", entry)
	}
	return NewObject(fields), nil
}

func NewDataEntryList(entries []proto.DataEntry) Exprs {
	r := make([]Expr, len(entries))
	for i, entry := range entries {
		v, err := newObjectExprFromDataEntry(entry)
		if err != nil {
			r[i] = NewUnit()
		}
		r[i] = v
	}
	return r
}

type Recipient interface {
	Expr
	Recipient() proto.Recipient
}

type RecipientExpr proto.Recipient

func NewRecipientFromProtoRecipient(a proto.Recipient) *RecipientExpr {
	r := RecipientExpr(a)
	return &r
}

func (a *RecipientExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *RecipientExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "RecipientExpr")
}

func (a *RecipientExpr) Eq(other Expr) bool {
	switch o := other.(type) {
	case *RecipientExpr:
		return a.Alias == o.Alias || a.Address == o.Address
	case *AddressExpr:
		return *a.Address == proto.Address(*o)
	case *AliasExpr:
		return *a.Alias == proto.Alias(*o)
	default:
		return false
	}
}

func (a *RecipientExpr) InstanceOf() string {
	return "Recipient"
}

func (a *RecipientExpr) Recipient() proto.Recipient {
	return proto.Recipient(*a)
}

type AssetPairExpr struct {
	fields object
}

func NewAssetPair(amountAsset Expr, priceAsset Expr) *AssetPairExpr {
	m := newObject()
	m["amountAsset"] = amountAsset
	m["priceAsset"] = priceAsset
	return &AssetPairExpr{fields: m}
}

func (a *AssetPairExpr) InstanceOf() string {
	return "AssetPair"
}

func (a *AssetPairExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *AssetPairExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "AssetPairExpr")
}

func (a *AssetPairExpr) Eq(other Expr) bool {
	if a.InstanceOf() != other.InstanceOf() {
		return false
	}
	o, ok := other.(*AssetPairExpr)
	if !ok {
		return false
	}
	return a.fields.Eq(o.fields)
}

func (a *AssetPairExpr) Get(name string) (Expr, error) {
	return a.fields.Get(name)
}

type object map[string]Expr

func newObject() object {
	return make(object)
}

func (a object) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "object")
}

func (a object) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a object) Eq(other Expr) bool {
	b, ok := other.(object)
	if !ok {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for k1, v1 := range a {
		v2, ok := b[k1]
		if !ok {
			return false
		}
		if !v1.Eq(v2) {
			return false
		}
	}
	return true
}

func (a object) Get(name string) (Expr, error) {
	out, ok := a[name]
	if !ok {
		return nil, errors.Errorf("ObjectExpr no such field %s", name)
	}
	return out, nil
}

func (a object) InstanceOf() string {
	return "object"
}

type BuyExpr struct{}

func NewBuy() *BuyExpr {
	return &BuyExpr{}
}

func (a *BuyExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *BuyExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "BuyExpr")
}

func (a *BuyExpr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *BuyExpr) InstanceOf() string {
	return "Buy"
}

type SellExpr struct{}

func NewSell() *SellExpr {
	return &SellExpr{}
}

func (a *SellExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *SellExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SellExpr")
}

func (a *SellExpr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *SellExpr) InstanceOf() string {
	return "Sell"
}

type CeilingExpr struct{}

func (a *CeilingExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *CeilingExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "CeilingExpr")
}

func (a *CeilingExpr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *CeilingExpr) InstanceOf() string {
	return "Ceiling"
}

type FloorExpr struct{}

func (a *FloorExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *FloorExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "FloorExpr")
}

func (a *FloorExpr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *FloorExpr) InstanceOf() string {
	return "Floor"
}

type HalfEvenExpr struct{}

func (a *HalfEvenExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *HalfEvenExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "HalfEvenExpr")
}

func (a *HalfEvenExpr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *HalfEvenExpr) InstanceOf() string {
	return "HalfEven"
}

type DownExpr struct{}

func (a *DownExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *DownExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "DownExpr")
}

func (a *DownExpr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *DownExpr) InstanceOf() string {
	return "Down"
}

type UpExpr struct{}

func (a *UpExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *UpExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "UpExpr")
}

func (a *UpExpr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *UpExpr) InstanceOf() string {
	return "Up"
}

type HalfUpExpr struct{}

func (a *HalfUpExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *HalfUpExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "HalfUpExpr")
}

func (a *HalfUpExpr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *HalfUpExpr) InstanceOf() string {
	return "HalfUp"
}

type HalfDownExpr struct{}

func (a *HalfDownExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *HalfDownExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "HalfDownExpr")
}

func (a *HalfDownExpr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *HalfDownExpr) InstanceOf() string {
	return "HalfDown"
}

type NoAlgExpr struct{}

func (a *NoAlgExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *NoAlgExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "NoAlgExpr")
}

func (a *NoAlgExpr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *NoAlgExpr) InstanceOf() string {
	return "NoAlg"
}

type MD5Expr struct{}

func (a *MD5Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *MD5Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "MD5Expr")
}

func (a *MD5Expr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *MD5Expr) InstanceOf() string {
	return "Md5"
}

type SHA1Expr struct{}

func (a *SHA1Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *SHA1Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA1Expr")
}

func (a *SHA1Expr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *SHA1Expr) InstanceOf() string {
	return "Sha1"
}

type SHA224Expr struct{}

func (a *SHA224Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *SHA224Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA224Expr")
}

func (a *SHA224Expr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *SHA224Expr) InstanceOf() string {
	return "Sha224"
}

type SHA256Expr struct{}

func (a *SHA256Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *SHA256Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA256Expr")
}

func (a *SHA256Expr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *SHA256Expr) InstanceOf() string {
	return "Sha256"
}

type SHA384Expr struct{}

func (a *SHA384Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *SHA384Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA384Expr")
}

func (a *SHA384Expr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *SHA384Expr) InstanceOf() string {
	return "Sha384"
}

type SHA512Expr struct{}

func (a *SHA512Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *SHA512Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA512Expr")
}

func (a SHA512Expr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *SHA512Expr) InstanceOf() string {
	return "Sha512"
}

type SHA3224Expr struct{}

func (a *SHA3224Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *SHA3224Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA3224Expr")
}

func (a *SHA3224Expr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *SHA3224Expr) InstanceOf() string {
	return "Sha3224"
}

type SHA3256Expr struct{}

func (a *SHA3256Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *SHA3256Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA3256Expr")
}

func (a *SHA3256Expr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *SHA3256Expr) InstanceOf() string {
	return "Sha3256"
}

type SHA3384Expr struct{}

func (a *SHA3384Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *SHA3384Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA3384Expr")
}

func (a *SHA3384Expr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *SHA3384Expr) InstanceOf() string {
	return "Sha3384"
}

type SHA3512Expr struct{}

func (a *SHA3512Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *SHA3512Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA3512Expr")
}

func (a *SHA3512Expr) Eq(other Expr) bool {
	return a.InstanceOf() == other.InstanceOf()
}

func (a *SHA3512Expr) InstanceOf() string {
	return "Sha3512"
}

//assetId ByteVector|Unit
//amount Int
type AttachedPaymentExpr struct {
	fields object
}

func NewAttachedPaymentExpr(assetId Expr, amount Expr) *AttachedPaymentExpr {
	fields := newObject()
	fields["assetId"] = assetId
	fields["amount"] = amount
	return &AttachedPaymentExpr{
		fields: fields,
	}
}

func (a *AttachedPaymentExpr) Write(w io.Writer) {
	_, _ = w.Write([]byte("AttachedPaymentExpr"))
}

func (a *AttachedPaymentExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *AttachedPaymentExpr) Eq(other Expr) bool {
	if a.InstanceOf() != other.InstanceOf() {
		return false
	}
	o := other.(*AttachedPaymentExpr)
	return a.fields.Eq(o.fields)
}

func (a *AttachedPaymentExpr) InstanceOf() string {
	return "AttachedPayment"
}

func (a *AttachedPaymentExpr) Get(key string) (Expr, error) {
	return a.fields.Get(key)
}

type BlockHeaderExpr struct {
	fields object
}

func (a *BlockHeaderExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "BlockHeaderExpr")
}

func (a *BlockHeaderExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *BlockHeaderExpr) Eq(other Expr) bool {
	return false
}

func (a *BlockHeaderExpr) InstanceOf() string {
	return "BlockHeader"
}

func (a *BlockHeaderExpr) Get(name string) (Expr, error) {
	return a.fields.Get(name)
}

func NewBlockHeader(fields object) *BlockHeaderExpr {
	return &BlockHeaderExpr{
		fields: fields,
	}
}

func makeFeatures(features []int16) Exprs {
	out := Exprs{}
	for _, f := range features {
		out = append(out, NewLong(int64(f)))
	}
	return out
}

type AssetInfoExpr struct {
	fields object
}

func (a *AssetInfoExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "AssetInfoExpr")
}

func (a *AssetInfoExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *AssetInfoExpr) Eq(other Expr) bool {
	return false
}

func (a *AssetInfoExpr) InstanceOf() string {
	return "Asset"
}

func (a *AssetInfoExpr) Get(name string) (Expr, error) {
	return a.fields.Get(name)
}

func NewAssetInfo(obj object) *AssetInfoExpr {
	return &AssetInfoExpr{fields: obj}
}

type BlockInfoExpr struct {
	fields object
}

func (a *BlockInfoExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "BlockInfoExpr")
}

func (a *BlockInfoExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *BlockInfoExpr) Get(name string) (Expr, error) {
	return a.fields.Get(name)
}

func (a *BlockInfoExpr) Eq(other Expr) bool {
	return false
}

func (a *BlockInfoExpr) InstanceOf() string {
	return "BlockInfo"
}

func NewBlockInfo(obj object, height proto.Height) *BlockInfoExpr {
	fields := newObject()
	fields["timestamp"] = obj["timestamp"]
	fields["height"] = NewLong(int64(height))
	fields["baseTarget"] = obj["baseTarget"]
	fields["generationSignature"] = obj["generationSignature"]
	fields["generator"] = obj["generator"]
	fields["generatorPublicKey"] = obj["generatorPublicKey"]
	return &BlockInfoExpr{
		fields: fields,
	}
}

type WriteSetExpr struct {
	body []*DataEntryExpr
}

func (a *WriteSetExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "WriteSetExpr")
}

func (a *WriteSetExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *WriteSetExpr) Eq(other Expr) bool {
	return false
}

func (a *WriteSetExpr) InstanceOf() string {
	return "WriteSet"
}

func (a *WriteSetExpr) toProto() ([]proto.DataEntry, error) {
	res := make([]proto.DataEntry, len(a.body))
	for i, entryExpr := range a.body {
		entry, err := entryExpr.toProto()
		if err != nil {
			return nil, err
		}
		res[i] = entry
	}
	return res, nil
}

func (a *WriteSetExpr) Get(name string) (Expr, error) {
	if name == "data" {
		r := make(Exprs, len(a.body))
		for i, item := range a.body {
			r[i] = item
		}
		return r, nil
	}
	return NewUnit(), nil
}

func NewWriteSet(e ...*DataEntryExpr) *WriteSetExpr {
	return &WriteSetExpr{
		body: e,
	}
}

type TransferSetExpr struct {
	body []*ScriptTransferExpr
}

func (a *TransferSetExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "TransferSetExpr")
}

func (a *TransferSetExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *TransferSetExpr) Eq(other Expr) bool {
	return false
}

func (a *TransferSetExpr) InstanceOf() string {
	return "TransferSet"
}

func (a *TransferSetExpr) toProto() ([]proto.ScriptResultTransfer, error) {
	res := make([]proto.ScriptResultTransfer, 0, len(a.body))
	for _, transferExpr := range a.body {
		transfer, err := transferExpr.toProto()
		if err != nil {
			return nil, err
		}
		if transfer.Amount == 0 { // Skip empty
			continue
		}
		res = append(res, *transfer)
	}
	return res, nil
}

func NewTransferSet(e ...*ScriptTransferExpr) *TransferSetExpr {
	return &TransferSetExpr{body: e}
}

type InvocationExpr struct {
	fields object
}

func (a *InvocationExpr) Get(name string) (Expr, error) {
	return a.fields.Get(name)
}

func (a *InvocationExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "InvocationExpr")
}

func (a *InvocationExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *InvocationExpr) Eq(other Expr) bool {
	return false
}

func (a *InvocationExpr) InstanceOf() string {
	return "Invocation"
}

func BuildInvocation(scheme proto.Scheme, tx *proto.InvokeScriptWithProofs) (*InvocationExpr, error) {
	fields := object{}
	addr, err := proto.NewAddressFromPublicKey(scheme, tx.SenderPK)
	if err != nil {
		return nil, err
	}
	fields["caller"] = NewAddressFromProtoAddress(addr)
	fields["callerPublicKey"] = NewBytes(tx.SenderPK.Bytes())
	fields["payment"] = NewUnit()
	if len(tx.Payments) > 0 {
		fields["payment"] = NewAttachedPaymentExpr(
			makeOptionalAsset(tx.Payments[0].Asset),
			NewLong(int64(tx.Payments[0].Amount)),
		)
	}
	fields["transactionId"] = NewBytes(tx.ID.Bytes())
	fields["fee"] = NewLong(int64(tx.Fee))
	fields["feeAssetId"] = makeOptionalAsset(tx.FeeAsset)

	return &InvocationExpr{
		fields: fields,
	}, nil
}

type ScriptTransferExpr struct {
	fields object
}

func (a *ScriptTransferExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "ScriptTransferExpr")
}

func (a *ScriptTransferExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *ScriptTransferExpr) Eq(other Expr) bool {
	return false
}

func (a *ScriptTransferExpr) InstanceOf() string {
	return "ScriptTransfer"
}

func (a *ScriptTransferExpr) toProto() (*proto.ScriptResultTransfer, error) {
	recipientExpr, ok := a.fields["recipient"]
	if !ok {
		return nil, errors.New("recipient is not found")
	}
	recipientExprDed, ok := recipientExpr.(Recipient)
	if !ok {
		return nil, errors.New("invalid recipient type")
	}
	amountExpr, ok := a.fields["amount"]
	if !ok {
		return nil, errors.New("amount is not found")
	}
	amountExprDed, ok := amountExpr.(*LongExpr)
	if !ok {
		return nil, errors.New("invalid amount type")
	}
	assetExpr, ok := a.fields["asset"]
	if !ok {
		return nil, errors.New("asset is not found")
	}
	var oa *proto.OptionalAsset
	var err error
	switch asset := assetExpr.(type) {
	case *Unit:
		oa = &proto.OptionalAsset{Present: false}
	case *BytesExpr:
		oa, err = proto.NewOptionalAssetFromBytes(asset.Value)
		if err != nil {
			return nil, errors.Wrap(err, "invalid asset id bytes")
		}
	default:
		return nil, errors.New("invalid type for asset expr")
	}
	return &proto.ScriptResultTransfer{
		Recipient: recipientExprDed.Recipient(),
		Amount:    amountExprDed.Value,
		Asset:     *oa,
	}, nil
}

func NewScriptTransfer(recipient Recipient, amount *LongExpr, asset Expr) (*ScriptTransferExpr, error) {
	switch asset.(type) {
	case *Unit, *BytesExpr:
	default:
		return nil, errors.Errorf("expected 'Unit' or '*BytesExpr' as asset, found %T", asset)
	}
	fields := object{}
	fields["recipient"] = recipient
	fields["amount"] = amount
	fields["asset"] = asset
	return &ScriptTransferExpr{
		fields: fields,
	}, nil
}

type ScriptResultExpr struct {
	WriteSet    *WriteSetExpr
	TransferSet *TransferSetExpr
}

func (a *ScriptResultExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "ScriptResultExpr")
}

func (a *ScriptResultExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a *ScriptResultExpr) Eq(other Expr) bool {
	return false
}

func (a *ScriptResultExpr) InstanceOf() string {
	return "ScriptResult"
}

func (a *ScriptResultExpr) ConvertToProto() (*proto.ScriptResult, error) {
	res := &proto.ScriptResult{}
	if a.TransferSet != nil {
		transfers, err := a.TransferSet.toProto()
		if err != nil {
			return nil, err
		}
		res.Transfers = transfers
	}
	if a.WriteSet != nil {
		writes, err := a.WriteSet.toProto()
		if err != nil {
			return nil, err
		}
		res.Writes = writes
	}
	return res, nil
}

func NewScriptResult(writeSet *WriteSetExpr, transferSet *TransferSetExpr) *ScriptResultExpr {
	return &ScriptResultExpr{
		WriteSet:    writeSet,
		TransferSet: transferSet,
	}
}
