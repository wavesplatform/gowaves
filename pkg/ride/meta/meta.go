package meta

import (
	"github.com/pkg/errors"
	g "github.com/wavesplatform/gowaves/pkg/ride/meta/generated"
)

// Type interface represents all type descriptors that can be encoded in Meta.
type Type interface {
	_type()
}

// SimpleType is one of a four basic types of Meta.
type SimpleType byte

func (t SimpleType) _type() {}

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

func (t UnionType) _type() {}

// ListType is a list of items of Inner type.
type ListType struct {
	Inner Type
}

func (t ListType) _type() {}

// Function is a function signature descriptor. Functions has Name and list of argument's types.
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

type Abbreviations struct {
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
	abbreviations := convertAbbreviations(meta.GetCompactNameAndOriginalNamePairList())
	switch v {
	case 0:
		return DApp{}, nil
	case 1:
		functions, err := convertFunctions(v, meta.GetFuncs())
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
		functions, err := convertFunctions(v, meta.GetFuncs())
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

func convertAbbreviations(pairs []*g.DAppMeta_CompactNameAndOriginalNamePair) Abbreviations {
	r := Abbreviations{compact2original: make(map[string]string)}
	for _, p := range pairs {
		r.compact2original[p.CompactName] = p.OriginalName
	}
	return r
}

func convertFunctions(version int, functions []*g.DAppMeta_CallableFuncSignature) ([]Function, error) {
	var typeConverter func([]byte) ([]Type, error)
	switch version {
	case 1:
		typeConverter = convertTypesV1
	case 2:
		typeConverter = convertTypesV2
	default:
		return nil, errors.Errorf("unsupported version of meta %d", version)
	}
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
