package ast

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const InstanceFieldName = "$instance"

type Script struct {
	//TODO: update for DApps support
	Version    int
	HasBlockV2 bool
	Verifier   Expr
}

type Expr interface {
	Write(io.Writer)
	Evaluate(Scope) (Expr, error)
	Eq(Expr) (bool, error)
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
		rs, err := row.Evaluate(s.Clone())
		if err != nil {
			return nil, err
		}
		out[i] = rs
	}
	return out, nil
}

func (a Exprs) Eq(other Expr) (bool, error) {
	o, ok := other.(Exprs)
	if !ok {
		return false, errors.Errorf("trying to compare %T with %T", a, other)
	}
	if len(a) != len(o) {
		return false, nil
	}
	for i := 0; i < len(a); i++ {
		eq, err := a[i].Eq(o[i])
		if err != nil {
			return false, errors.Wrapf(err, "compare Exprs")
		}
		if !eq {
			return false, nil
		}
	}
	return true, nil
}

func (a Exprs) InstanceOf() string {
	return "Exprs"
}

func NewExprs(e ...Expr) Exprs {
	return e
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
	s.AddValue(a.Let.Name, a.Let.Value)
	return a.Body.Evaluate(s.Clone())
}

func (a *Block) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
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
	s.AddValue(a.Name, NewFunction(a.Args, a.Body))
	return a, nil
}

func (a *FuncDeclaration) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
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
	_, _ = a.Decl.Evaluate(s)
	return a.Body.Evaluate(s.Clone())
}

func (a *BlockV2) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
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

func (a *LetExpr) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
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

func (a *LongExpr) Eq(other Expr) (bool, error) {
	b, ok := other.(*LongExpr)
	if !ok {
		return false, errors.Errorf("trying to compare %T with %T", a, other)
	}
	return a.Value == b.Value, nil
}

func (a *LongExpr) InstanceOf() string {
	return "Long"
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

func (a *BooleanExpr) Eq(other Expr) (bool, error) {
	b, ok := other.(*BooleanExpr)
	if !ok {
		return false, errors.Errorf("trying to compare %T with %T", a, other)
	}

	return a.Value == b.Value, nil
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

func (a *FuncCallExpr) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a *FuncCallExpr) InstanceOf() string {
	return "FuncCallExpr"
}

func NewFuncCall(f Expr) *FuncCallExpr {
	return &FuncCallExpr{
		Func: f,
	}
}

type NativeFunction struct {
	FunctionID int16
	Argc       int
	Argv       Exprs
}

func NewNativeFunction(id int16, argc int, argv Exprs) *NativeFunction {
	return &NativeFunction{
		FunctionID: id,
		Argc:       argc,
		Argv:       argv,
	}
}

func (a *NativeFunction) Write(w io.Writer) {
	writeNativeFunction(w, a.FunctionID, a.Argv)
}

func (a *NativeFunction) Evaluate(s Scope) (Expr, error) {
	name := strconv.Itoa(int(a.FunctionID))
	e, ok := s.Value(name)
	if !ok {
		return nil, errors.Errorf("evaluate native function: function named '%s' not found in scope", name)
	}
	fn, ok := e.(*Function)
	if !ok {
		return nil, errors.Errorf("evaluate native function: expected value 'fn' to be *Function, found %T", e)
	}
	if fn.Argc != a.Argc {
		return nil, errors.Errorf("evaluate native function: function %s expects %d arguments, passed %d", name, fn.Argc, a.Argc)
	}
	initial := s.Initial()
	for i := 0; i < a.Argc; i++ {
		evaluatedParam, err := a.Argv[i].Evaluate(s.Clone())
		if err != nil {
			return nil, errors.Wrapf(err, "evaluate native function: %s", name)
		}
		initial.AddValue(fn.Argv[i], evaluatedParam)
	}
	return fn.Evaluate(initial)
}

func (a *NativeFunction) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a *NativeFunction) InstanceOf() string {
	return "NativeFunction"
}

type UserFunctionCall struct {
	Name string
	Argc int
	Argv Exprs
}

func NewUserFunctionCall(name string, argc int, argv Exprs) *UserFunctionCall {
	return &UserFunctionCall{
		Name: name,
		Argc: argc,
		Argv: argv,
	}
}

func (a *UserFunctionCall) Write(w io.Writer) {
	if a.Name == "!=" {
		infix(w, " != ", a.Argv)
		return
	}
	prefix(w, a.Name, a.Argv)
}

func (a *UserFunctionCall) Evaluate(s Scope) (Expr, error) {
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
	initial := s.Initial()
	for i := 0; i < a.Argc; i++ {
		evaluatedParam, err := a.Argv[i].Evaluate(s.Clone())
		if err != nil {
			return nil, errors.Wrapf(err, "evaluate user function: %s", a.Name)
		}
		initial.AddValue(fn.Argv[i], evaluatedParam)
	}
	return fn.Evaluate(initial)
}

func (a *UserFunctionCall) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a *UserFunctionCall) InstanceOf() string {
	return "UserFunctionCall"
}

type Function struct {
	Argc int
	Argv []string
	Body Expr
}

func (a *Function) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "Function")
}

func (a *Function) Evaluate(s Scope) (Expr, error) {
	return a.Body.Evaluate(s)
}

func (a *Function) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a *Function) InstanceOf() string {
	return "Function"
}

func NewFunction(Argv []string, Body Expr) *Function {
	return &Function{
		Argc: len(Argv),
		Argv: Argv,
		Body: Body,
	}
}

func FunctionFromPredefined(c Callable, argc uint32) *Function {
	return &Function{
		Argc: int(argc),
		Argv: buildParams(argc),
		Body: &PredefinedUserFunction{
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

type PredefinedUserFunction struct {
	argv []string
	fn   Callable
}

func (a PredefinedUserFunction) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "PredefinedUserFunction")
}

func (a PredefinedUserFunction) Evaluate(s Scope) (Expr, error) {
	params := Params()
	for i := 0; i < len(a.argv); i++ {
		e, ok := s.Value(a.argv[i])
		if !ok {
			return nil, errors.Errorf("PredefinedUserFunction: param %s not found in scope", a.argv[i])
		}
		params = append(params, e)
	}
	return a.fn(s.Clone(), params)
}

func (a PredefinedUserFunction) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a PredefinedUserFunction) InstanceOf() string {
	return "PredefinedUserFunction"
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
		return nil, errors.Errorf("RefExpr evaluate: not found expr by name %s", a.Name)
	}
	rs, err := expr.Evaluate(s.Clone())
	s.setEvaluation(a.Name, evaluation{rs, err})
	return rs, err
}

func (a *RefExpr) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
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
	cond, err := a.Condition.Evaluate(s.Clone())
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

func (a *IfExpr) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
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

func (a *BytesExpr) Eq(other Expr) (bool, error) {
	b, ok := other.(*BytesExpr)
	if !ok {
		return false, errors.Errorf("trying to compare %T with %T", a, other)
	}

	return bytes.Equal(a.Value, b.Value), nil
}

func (a *BytesExpr) InstanceOf() string {
	return "Bytes"
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
	val, err := a.Object.Evaluate(s.Clone())
	if err != nil {
		return nil, errors.Wrapf(err, "GetterExpr Evaluate by key %s", a.Key)
	}

	if obj, ok := val.(Getable); ok {
		e, err := obj.Get(a.Key)
		if err != nil {
			return nil, err
		}
		return e, nil
	}
	return nil, errors.Errorf("GetterExpr Evaluate: expected value be Getable, got %T", val)
}

func (a *GetterExpr) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
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

func (a *ObjectExpr) Eq(other Expr) (bool, error) {
	b, ok := other.(*ObjectExpr)
	if !ok {
		return false, errors.Errorf("trying to compare %T with %T", a, other)
	}

	if len(a.fields) != len(b.fields) {
		return false, nil
	}

	for k1, v1 := range a.fields {
		v2, ok := b.fields[k1]
		if !ok {
			return false, nil
		}
		rs, err := v1.Eq(v2)
		if err != nil {
			return false, err
		}
		if !rs {
			return false, nil
		}
	}

	return true, nil
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

func (a *StringExpr) Eq(other Expr) (bool, error) {
	b, ok := other.(*StringExpr)
	if !ok {
		return false, errors.Errorf("trying to compare %T with %T", a, other)
	}

	return a.Value == b.Value, nil
}

func (a *StringExpr) InstanceOf() string {
	return "String"
}

type AddressExpr proto.Address

func (a AddressExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, proto.Address(a).String())
}

func (a AddressExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a AddressExpr) Eq(other Expr) (bool, error) {
	b, ok := other.(AddressExpr)
	if !ok {
		return false, errors.Errorf("trying to compare AddressExpr with %T", other)
	}

	return bytes.Equal(a[:], b[:]), nil
}

func (a AddressExpr) InstanceOf() string {
	return "AddressExpr"
}

func NewAddressFromString(s string) (AddressExpr, error) {
	addr, err := proto.NewAddressFromString(s)
	return AddressExpr(addr), err
}

func NewAddressFromProtoAddress(a proto.Address) AddressExpr {
	return AddressExpr(a)
}

type Unit struct {
}

func (a Unit) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "Unit")
}

func (a Unit) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a Unit) Eq(other Expr) (bool, error) {
	if other.InstanceOf() == a.InstanceOf() {
		return true, nil
	}
	return false, nil
}

func (a Unit) InstanceOf() string {
	return "Unit"
}

func NewUnit() Unit {
	return Unit{}
}

type AliasExpr proto.Alias

func (a AliasExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "Alias")
}

func (a AliasExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a AliasExpr) Eq(other Expr) (bool, error) {
	if b, ok := other.(AliasExpr); ok {
		return proto.Alias(a).String() == proto.Alias(b).String(), nil
	}

	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a AliasExpr) InstanceOf() string {
	return "Alias"
}

func NewAliasFromProtoAlias(a proto.Alias) AliasExpr {
	return AliasExpr(a)
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

type RecipientExpr proto.Recipient

func NewRecipientFromProtoRecipient(a proto.Recipient) RecipientExpr {
	return RecipientExpr(a)
}

func (a RecipientExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a RecipientExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "RecipientExpr")
}

func (a RecipientExpr) Eq(other Expr) (bool, error) {
	switch o := other.(type) {
	case RecipientExpr:
		return a.Alias == o.Alias && a.Address == o.Address, nil
	case AddressExpr:
		return *a.Address == proto.Address(o), nil
	case AliasExpr:
		return *a.Alias == proto.Alias(o), nil
	default:
		return false, errors.Errorf("trying to compare %T with %T", a, other)
	}
}

func (a RecipientExpr) InstanceOf() string {
	return "Recipient"
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

func (a AssetPairExpr) InstanceOf() string {
	return "AssetPair"
}

func (a AssetPairExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a AssetPairExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "AssetPairExpr")
}

func (a AssetPairExpr) Eq(other Expr) (bool, error) {
	if a.InstanceOf() != other.InstanceOf() {
		return false, errors.Errorf("trying to compare %T with %T", a, other)
	}
	o, ok := other.(*AssetPairExpr)
	if !ok {
		return false, errors.Errorf("can't cast %T as type *AssetPairExpr", other)
	}
	return a.fields.Eq(o.fields)
}

func (a AssetPairExpr) Get(name string) (Expr, error) {
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

func (a object) Eq(other Expr) (bool, error) {
	b, ok := other.(object)
	if !ok {
		return false, errors.Errorf("trying to compare %T with %T", a, other)
	}

	if len(a) != len(b) {
		return false, nil
	}

	for k1, v1 := range a {
		v2, ok := b[k1]
		if !ok {
			return false, nil
		}
		rs, err := v1.Eq(v2)
		if err != nil {
			return false, err
		}
		if !rs {
			return false, nil
		}
	}

	return true, nil
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

func (a BuyExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a BuyExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "BuyExpr")
}

func (a BuyExpr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a BuyExpr) InstanceOf() string {
	return "Buy"
}

type SellExpr struct{}

func NewSell() *SellExpr {
	return &SellExpr{}
}

func (a SellExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a SellExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SellExpr")
}

func (a SellExpr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a SellExpr) InstanceOf() string {
	return "Sell"
}

type CeilingExpr struct{}

func (a CeilingExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a CeilingExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "CeilingExpr")
}

func (a CeilingExpr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a CeilingExpr) InstanceOf() string {
	return "Ceiling"
}

type FloorExpr struct{}

func (a FloorExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a FloorExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "FloorExpr")
}

func (a FloorExpr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a FloorExpr) InstanceOf() string {
	return "Floor"
}

type HalfEvenExpr struct{}

func (a HalfEvenExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a HalfEvenExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "HalfEvenExpr")
}

func (a HalfEvenExpr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a HalfEvenExpr) InstanceOf() string {
	return "HalfEven"
}

type DownExpr struct{}

func (a DownExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a DownExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "DownExpr")
}

func (a DownExpr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a DownExpr) InstanceOf() string {
	return "Down"
}

type UpExpr struct{}

func (a UpExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a UpExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "UpExpr")
}

func (a UpExpr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a UpExpr) InstanceOf() string {
	return "Up"
}

type HalfUpExpr struct{}

func (a HalfUpExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a HalfUpExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "HalfUpExpr")
}

func (a HalfUpExpr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a HalfUpExpr) InstanceOf() string {
	return "HalfUp"
}

type HalfDownExpr struct{}

func (a HalfDownExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a HalfDownExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "HalfDownExpr")
}

func (a HalfDownExpr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a HalfDownExpr) InstanceOf() string {
	return "HalfDown"
}

type NoAlgExpr struct{}

func (a NoAlgExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a NoAlgExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "NoAlgExpr")
}

func (a NoAlgExpr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a NoAlgExpr) InstanceOf() string {
	return "NoAlg"
}

type MD5Expr struct{}

func (a MD5Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a MD5Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "MD5Expr")
}

func (a MD5Expr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a MD5Expr) InstanceOf() string {
	return "Md5"
}

type SHA1Expr struct{}

func (a SHA1Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a SHA1Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA1Expr")
}

func (a SHA1Expr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a SHA1Expr) InstanceOf() string {
	return "Sha1"
}

type SHA224Expr struct{}

func (a SHA224Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a SHA224Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA224Expr")
}

func (a SHA224Expr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a SHA224Expr) InstanceOf() string {
	return "Sha224"
}

type SHA256Expr struct{}

func (a SHA256Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a SHA256Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA256Expr")
}

func (a SHA256Expr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a SHA256Expr) InstanceOf() string {
	return "Sha256"
}

type SHA384Expr struct{}

func (a SHA384Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a SHA384Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA384Expr")
}

func (a SHA384Expr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a SHA384Expr) InstanceOf() string {
	return "Sha384"
}

type SHA512Expr struct{}

func (a SHA512Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a SHA512Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA512Expr")
}

func (a SHA512Expr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a SHA512Expr) InstanceOf() string {
	return "Sha512"
}

type SHA3224Expr struct{}

func (a SHA3224Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a SHA3224Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA3224Expr")
}

func (a SHA3224Expr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a SHA3224Expr) InstanceOf() string {
	return "Sha3224"
}

type SHA3256Expr struct{}

func (a SHA3256Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a SHA3256Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA3256Expr")
}

func (a SHA3256Expr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a SHA3256Expr) InstanceOf() string {
	return "Sha3256"
}

type SHA3384Expr struct{}

func (a SHA3384Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a SHA3384Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA3384Expr")
}

func (a SHA3384Expr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a SHA3384Expr) InstanceOf() string {
	return "Sha3384"
}

type SHA3512Expr struct{}

func (a SHA3512Expr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a SHA3512Expr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "SHA3512Expr")
}

func (a SHA3512Expr) Eq(other Expr) (bool, error) {
	return a.InstanceOf() == other.InstanceOf(), nil
}

func (a SHA3512Expr) InstanceOf() string {
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

func (a AttachedPaymentExpr) Write(w io.Writer) {
	_, _ = w.Write([]byte("AttachedPaymentExpr"))
}

func (a AttachedPaymentExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a AttachedPaymentExpr) Eq(other Expr) (bool, error) {
	if a.InstanceOf() != other.InstanceOf() {
		return false, errors.Errorf("trying to compare %T with %T", a, other)
	}
	o := other.(*AttachedPaymentExpr)
	return a.fields.Eq(o.fields)
}

func (a AttachedPaymentExpr) InstanceOf() string {
	return "AttachedPayment"
}

func (a AttachedPaymentExpr) Get(key string) (Expr, error) {
	return a.fields.Get(key)
}

type BlockHeaderExpr struct {
	fields object
}

func (a BlockHeaderExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "BlockHeaderExpr")
}

func (a BlockHeaderExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a BlockHeaderExpr) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a BlockHeaderExpr) InstanceOf() string {
	return "BlockHeader"
}

func (a BlockHeaderExpr) Get(name string) (Expr, error) {
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

func (a AssetInfoExpr) Write(w io.Writer) {
	_, _ = fmt.Fprintf(w, "AssetInfoExpr")
}

func (a AssetInfoExpr) Evaluate(Scope) (Expr, error) {
	return a, nil
}

func (a AssetInfoExpr) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a AssetInfoExpr) InstanceOf() string {
	return "AssetInfo"
}

func (a AssetInfoExpr) Get(name string) (Expr, error) {
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

func (a BlockInfoExpr) Get(name string) (Expr, error) {
	return a.fields.Get(name)
}

func (a BlockInfoExpr) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a BlockInfoExpr) InstanceOf() string {
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

func Merge(x map[string]Expr, y map[string]Expr) map[string]Expr {
	out := make(map[string]Expr)
	for k, v := range x {
		out[k] = v
	}
	for k, v := range y {
		out[k] = v
	}
	return out
}
