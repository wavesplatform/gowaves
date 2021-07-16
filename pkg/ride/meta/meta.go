package meta

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride"
	g "github.com/wavesplatform/gowaves/pkg/ride/meta/generated"
	protobuf "google.golang.org/protobuf/proto"
)

type Type byte

const (
	TypeInt = iota + 1
)

type Function struct {
	Name      string
	Arguments []Type
}

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

func FromProtobuf(meta ride.ScriptMeta) (DApp, error) {
	switch meta.Version {
	case 0:
		pbMeta := new(g.DAppMeta)
		if err := protobuf.Unmarshal(meta.Bytes, pbMeta); err != nil {
			return DApp{}, err
		}
		m, err := convert(pbMeta)
		if err != nil {
			return DApp{}, err
		}
		return m, nil
	default:
		return DApp{}, errors.Errorf("unsupported script meta version %d", meta.Version)
	}
}

func convert(meta *g.DAppMeta) (DApp, error) {
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
	var typeConverter func([]byte) []Type
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
		types := typeConverter(f.GetTypes())
		r[i] = Function{
			Name:      "",
			Arguments: types,
		}
	}
	return r, nil
}

func convertTypesV1(dtypes []byte) []Type {
	return nil
}

func convertTypesV2(types []byte) []Type {
	return nil
}
