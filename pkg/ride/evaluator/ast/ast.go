package ast

import (
	"bytes"
	"fmt"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"io"
)

const InstanceFieldName = "$instance"

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
	return nil, errors.New("Exprs Evaluate")
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
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a Exprs) InstanceOf() string {
	return "Exprs"
}

func NewExprs(e ...Expr) Exprs {
	return e
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

type LetExpr struct {
	Name  string
	Value Expr
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

func (a *BooleanExpr) Evaluate(scope Scope) (Expr, error) {
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

type FuncCall struct {
	Func Expr
}

func (a *FuncCall) Write(w io.Writer) {
	a.Func.Write(w)
}

func (a *FuncCall) Evaluate(s Scope) (Expr, error) {
	return a.Func.Evaluate(s)
}

func (a *FuncCall) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a *FuncCall) InstanceOf() string {
	return "FuncCall"
}

func NewFuncCall(f Expr) *FuncCall {
	return &FuncCall{
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
	f, ok := s.FuncByShort(a.FunctionID)
	if !ok {
		return nil, errors.Errorf("evaluate native function: function id %d not found in scope", a.FunctionID)
	}

	return f(s.Clone(), a.Argv)
}

func (a *NativeFunction) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a *NativeFunction) InstanceOf() string {
	return "NativeFunction"
}

type UserFunction struct {
	Name string
	Argc int
	Argv Exprs
}

func NewUserFunction(name string, argc int, argv Exprs) *UserFunction {
	return &UserFunction{
		Name: name,
		Argc: argc,
		Argv: argv,
	}
}

func (a *UserFunction) Write(w io.Writer) {
	if a.Name == "!=" {
		infix(w, " != ", a.Argv)
		return
	}
	prefix(w, a.Name, a.Argv)
}

func (a *UserFunction) Evaluate(s Scope) (Expr, error) {
	f, ok := s.FuncByName(a.Name)
	if !ok {
		return nil, errors.Errorf("evaluate user function: function name %s not found in scope", a.Name)
	}

	return f(s.Clone(), a.Argv)
}

func (a *UserFunction) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a *UserFunction) InstanceOf() string {
	return "UserFunction"
}

type RefExpr struct {
	Name   string
	cached bool
	cache  RefCache
}

type RefCache struct {
	Expr Expr
	Err  error
}

func (a *RefExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, a.Name)
}

func (a *RefExpr) Evaluate(s Scope) (Expr, error) {

	if a.cached {
		return a.cache.Expr, a.cache.Err
	}

	expr, ok := s.Value(a.Name)
	if !ok {
		return nil, errors.Errorf("RefExpr evaluate: not found expr by name %s", a.Name)
	}

	rs, err := expr.Evaluate(s.Clone())

	a.cache = RefCache{
		Expr: rs,
		Err:  err,
	}
	a.cached = true

	return a.cache.Expr, a.cache.Err
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

func (a *BytesExpr) Evaluate(s Scope) (Expr, error) {
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

	if obj, ok := val.(*ObjectExpr); ok {
		e, err := obj.Get(a.Key)
		if err != nil {
			return nil, err
		}
		return e, nil
	}
	return nil, errors.Errorf("GetterExpr Evaluate: expected value be *ObjectExpr, got %T", val)
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

func (a *ObjectExpr) Evaluate(s Scope) (Expr, error) {
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

func (a *StringExpr) Evaluate(s Scope) (Expr, error) {
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

func (a AddressExpr) Evaluate(s Scope) (Expr, error) {
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

func (a Unit) Evaluate(s Scope) (Expr, error) {
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

func (a AliasExpr) Evaluate(s Scope) (Expr, error) {
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

type DataEntryListExpr struct {
	source []proto.DataEntry
	cached bool
	data   map[string]proto.DataEntry
}

func (a DataEntryListExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "Alias")
}

func (a *DataEntryListExpr) Evaluate(s Scope) (Expr, error) {
	return a, nil
}

func (a DataEntryListExpr) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a DataEntryListExpr) InstanceOf() string {
	return "DataEntryList"
}

func (a *DataEntryListExpr) Get(key string, valueType proto.DataValueType) Expr {
	if !a.cached {
		a.cache()
	}
	rs, ok := a.data[key]
	if ok {
		if rs.GetValueType() == valueType {
			switch valueType {
			case proto.DataInteger:
				return NewLong(rs.(*proto.IntegerDataEntry).Value)
			case proto.DataString:
				return NewString(rs.(*proto.StringDataEntry).Value)
			case proto.DataBoolean:
				return NewBoolean(rs.(*proto.BooleanDataEntry).Value)
			case proto.DataBinary:
				return NewBytes(rs.(*proto.BinaryDataEntry).Value)
			}
		}
	}
	return Unit{}
}

func (a *DataEntryListExpr) GetByIndex(index int, valueType proto.DataValueType) Expr {
	if index > len(a.source)-1 {
		return NewUnit()
	}

	rs := a.source[index]
	if rs.GetValueType() != valueType {
		return NewUnit()
	}

	switch valueType {
	case proto.DataInteger:
		return NewLong(rs.(*proto.IntegerDataEntry).Value)
	case proto.DataString:
		return NewString(rs.(*proto.StringDataEntry).Value)
	case proto.DataBoolean:
		return NewBoolean(rs.(*proto.BooleanDataEntry).Value)
	case proto.DataBinary:
		return NewBytes(rs.(*proto.BinaryDataEntry).Value)
	default:
		return NewUnit()
	}
}

func (a *DataEntryListExpr) cache() {
	a.data = make(map[string]proto.DataEntry)
	for _, row := range a.source {
		a.data[row.GetKey()] = row
	}
	a.cached = true
}

func NewDataEntryList(d []proto.DataEntry) *DataEntryListExpr {
	return &DataEntryListExpr{
		source: d,
	}
}

type RecipientExpr proto.Recipient

func NewRecipientFromProtoRecipient(a proto.Recipient) RecipientExpr {
	return RecipientExpr(a)
}

func (a RecipientExpr) Evaluate(s Scope) (Expr, error) {
	return a, nil
}

func (a RecipientExpr) Write(w io.Writer) {
	_, _ = fmt.Fprint(w, "RecipientExpr")
}

func (a RecipientExpr) Eq(other Expr) (bool, error) {
	return false, errors.Errorf("trying to compare %T with %T", a, other)
}

func (a RecipientExpr) InstanceOf() string {
	return "RecipientExpr"
}
