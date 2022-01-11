package meta

import (
	"github.com/pkg/errors"
	g "github.com/wavesplatform/gowaves/pkg/ride/meta/generated"
)

// Type interface represents all type descriptors that can be encoded in Meta.
type Type interface {
	metaTypeMarker()
}

// SimpleType is one of four basic types of Meta.
type SimpleType byte

func (SimpleType) metaTypeMarker() {}

const (
	Int SimpleType = 1 << iota
	Bytes
	Boolean
	String
	list
)

const (
	basicTypesCount    = 4
	combinedBasicTypes = byte(Int | Bytes | Boolean | String)
	combinedListTypes  = byte(Int | Bytes | Boolean | String | list)
)

// UnionType represents a composition of basic types.
type UnionType []SimpleType

func (UnionType) metaTypeMarker() {}

// ListType is a list of items of Inner type.
type ListType struct {
	Inner Type
}

func (ListType) metaTypeMarker() {}

// Function is a function signature descriptor. Function has a Name and list of argument's types.
type Function struct {
	Name      string
	Arguments []Type
}

// DApp is a collection of callable functions' descriptions. As additional it has Version and Abbreviations map.
type DApp struct {
	Version       int
	Functions     []Function
	Abbreviations Abbreviations
}

type pair struct {
	compact  string
	original string
}

type Abbreviations struct {
	pairs            []pair
	names            []string
	compact2original map[string]string
}

func (a *Abbreviations) CompactToOriginal(compact string) (string, error) {
	if n, ok := a.compact2original[compact]; ok {
		return n, nil
	}
	return "", errors.Errorf("short name '%s' not found", compact)
}

func Convert(meta *g.DAppMeta) (DApp, error) {
	v := int(meta.GetVersion())
	abbreviations := convertAbbreviations(meta.GetCompactNameAndOriginalNamePairList(), meta.GetOriginalNames())
	switch v {
	case 0:
		return DApp{}, nil
	case 1:
		functions, err := convertFunctions(convertTypesV1, meta.GetFuncs())
		if err != nil {
			return DApp{}, err
		}
		m := DApp{
			Version:       v,
			Functions:     functions,
			Abbreviations: abbreviations,
		}
		return m, nil
	case 2:
		functions, err := convertFunctions(convertTypesV2, meta.GetFuncs())
		if err != nil {
			return DApp{}, err
		}
		m := DApp{
			Version:       v,
			Functions:     functions,
			Abbreviations: abbreviations,
		}
		return m, nil
	default:
		return DApp{}, errors.Errorf("unsupported meta version %d", v)
	}
}

func buildAbbreviations(abbreviations Abbreviations) ([]*g.DAppMeta_CompactNameAndOriginalNamePair, []string) {
	r := make([]*g.DAppMeta_CompactNameAndOriginalNamePair, len(abbreviations.compact2original))
	for i, p := range abbreviations.pairs {
		r[i] = &g.DAppMeta_CompactNameAndOriginalNamePair{
			CompactName:  p.compact,
			OriginalName: p.original,
		}
	}
	return r, abbreviations.names
}

func buildFunctions(typeBuilder func([]Type) ([]byte, error), functions []Function) ([]*g.DAppMeta_CallableFuncSignature, error) {
	r := make([]*g.DAppMeta_CallableFuncSignature, len(functions))
	for i, f := range functions {
		ts, err := typeBuilder(f.Arguments)
		if err != nil {
			return nil, err
		}
		r[i] = &g.DAppMeta_CallableFuncSignature{
			Types: ts,
		}
	}
	return r, nil
}

func typeBuilderV1(types []Type) ([]byte, error) {
	r := make([]byte, 0, len(types))
	for _, t := range types {
		if _, ok := t.(ListType); ok {
			return nil, errors.New("unsupported type 'ListType'")
		}
		b, err := encodeType(t)
		if err != nil {
			return nil, err
		}
		r = append(r, b)
	}
	return r, nil
}

func typeBuilderV2(types []Type) ([]byte, error) {
	r := make([]byte, 0, len(types))
	for _, t := range types {
		b, err := encodeType(t)
		if err != nil {
			return nil, err
		}
		r = append(r, b)
	}
	return r, nil
}

func Build(m DApp) (*g.DAppMeta, error) {
	switch m.Version {
	case 0:
		return new(g.DAppMeta), nil
	case 1:
		r := new(g.DAppMeta)
		r.Version = int32(m.Version)
		fns, err := buildFunctions(typeBuilderV1, m.Functions)
		if err != nil {
			return nil, err
		}
		r.Funcs = fns
		r.CompactNameAndOriginalNamePairList, r.OriginalNames = buildAbbreviations(m.Abbreviations)
		return r, nil
	case 2:
		r := new(g.DAppMeta)
		r.Version = int32(m.Version)
		fns, err := buildFunctions(typeBuilderV2, m.Functions)
		if err != nil {
			return nil, err
		}
		r.Funcs = fns
		r.CompactNameAndOriginalNamePairList, r.OriginalNames = buildAbbreviations(m.Abbreviations)
		return r, nil
	default:
		return nil, errors.Errorf("unsupported meta version %d", m.Version)
	}
}

func convertAbbreviations(pairs []*g.DAppMeta_CompactNameAndOriginalNamePair, names []string) Abbreviations {
	l := len(pairs)
	r := Abbreviations{pairs: make([]pair, l), names: names, compact2original: make(map[string]string, l)}
	for i, p := range pairs {
		r.compact2original[p.CompactName] = p.OriginalName
		r.pairs[i] = pair{compact: p.CompactName, original: p.OriginalName}
	}
	return r
}

func convertFunctions(typeConverter func([]byte) ([]Type, error), functions []*g.DAppMeta_CallableFuncSignature) ([]Function, error) {
	r := make([]Function, len(functions))
	for i, f := range functions {
		types, err := typeConverter(f.GetTypes())
		if err != nil {
			return nil, err
		}
		r[i] = Function{
			Name:      "",
			Arguments: types,
		}
	}
	return r, nil
}

func parseUnion(t byte) Type {
	r := make([]SimpleType, 0, basicTypesCount)
	for i := 0; i < basicTypesCount; i++ {
		m := byte(1 << i)
		if t&m == m {
			r = append(r, SimpleType(m))
		}
	}
	if len(r) == 1 {
		return r[0]
	}
	return UnionType(r)
}

func encodeType(t Type) (byte, error) {
	switch tt := t.(type) {
	case SimpleType:
		return byte(tt), nil
	case UnionType:
		r := byte(0)
		for _, st := range tt {
			r |= byte(st)
		}
		return r, nil
	case ListType:
		inner, err := encodeType(tt.Inner)
		if err != nil {
			return 0, err
		}
		return byte(list) | inner, nil
	default:
		return 0, errors.Errorf("")
	}
}

func convertTypesV1(types []byte) ([]Type, error) {
	r := make([]Type, 0, len(types))
	for _, t := range types {
		if t < byte(Int) || t > combinedBasicTypes {
			return nil, errors.Errorf("unsupproted type '%d' for meta V1", t)
		}
		r = append(r, parseUnion(t))
	}
	return r, nil
}

func parseList(t byte) Type {
	if t < byte(list) {
		return parseUnion(t)
	}
	t = t ^ byte(list)
	return ListType{Inner: parseUnion(t)}
}

func convertTypesV2(types []byte) ([]Type, error) {
	r := make([]Type, 0, len(types))
	for _, t := range types {
		if t < byte(Int) || t > combinedListTypes {
			return nil, errors.Errorf("unsupported type '%d' for meta V2", t)
		}
		r = append(r, parseList(t))
	}
	return r, nil
}
