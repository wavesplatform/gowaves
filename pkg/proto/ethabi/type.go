package ethabi

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
)

type ArgT byte

// Type enumerator
const (
	IntTy ArgT = iota
	UintTy
	BytesTy
	BoolTy
	StringTy

	SliceTy
	TupleTy

	AddressTy    // we use this type only for erc20 transfers
	FixedBytesTy // we use this type only for payment asset
)

// Type is the reflection of the supported argument type.
type Type struct {
	Elem *Type // nested types for SliceTy
	Size int
	T    ArgT // Our own type checking

	stringKind string // holds the unparsed string for deriving signatures

	// Tuple relative fields
	TupleRawName string    // Raw struct name defined in source code, may be empty.
	TupleFields  Arguments // Type and name information of all tuple fields
}

func (t *Type) String() string {
	return t.stringKind
}

// requiresLengthPrefix returns whether the type requires any sort of length prefixing.
func requiresLengthPrefix(t Type) bool {
	return t.T == StringTy || t.T == BytesTy || t.T == SliceTy
}

// getTypeSize returns the size that this type needs to occupy.
// We distinguish static and dynamic types. Static types are encoded in-place
// and dynamic types are encoded at a separately allocated location after the
// current block.
// So for a static variable, the size returned represents the size that the
// variable actually occupies.
// For a dynamic variable, the returned size is fixed 32 bytes, which is used
// to store the location reference for actual value storage.
func getTypeSize(t Type) int {
	if t.T == TupleTy && !isDynamicType(t) {
		// Recursively calculate type size if it is a nested tuple
		total := 0
		for _, elem := range t.TupleFields {
			total += getTypeSize(elem.Type)
		}
		return total
	}
	return 32
}

// isDynamicType returns true if the type is dynamic.
// The following types are called “dynamic”:
// * bytes
// * string
// * T[] for any T
// * T[k] for any dynamic T and any k >= 0
// * (T1,...,Tk) if Ti is dynamic for some 1 <= i <= k
func isDynamicType(t Type) bool {
	if t.T == TupleTy {
		for _, elem := range t.TupleFields {
			if isDynamicType(elem.Type) {
				return true
			}
		}
		return false
	}
	return t.T == StringTy || t.T == BytesTy || t.T == SliceTy
}

func AbiTypeFromRideTypeMeta(metaT meta.Type) (abiT Type, err error) {
	switch t := metaT.(type) {
	case meta.SimpleType:
		switch t {
		case meta.Int:
			abiT = Type{T: IntTy, Size: 64}
		case meta.Bytes:
			abiT = Type{T: BytesTy}
		case meta.Boolean:
			abiT = Type{T: BoolTy}
		case meta.String:
			abiT = Type{T: StringTy}
		default:
			return Type{}, errors.Errorf("invalid ride simple type (%d)", t)
		}
	case meta.ListType:
		inner, err := AbiTypeFromRideTypeMeta(t.Inner)
		if err != nil {
			return Type{}, errors.Wrapf(err,
				"failed to create abi type for ride meta list type, inner type %T", t.Inner,
			)
		}
		abiT = Type{Elem: &inner, T: SliceTy}
	case meta.UnionType:
		indexElemStrKindMarshaler := intTextBuilder{
			size:     8,
			unsigned: true,
		}
		indexElemStringKind, err := indexElemStrKindMarshaler.MarshalText()
		if err != nil {
			return Type{}, errors.Wrap(err, "failed to marshal index elem stringKind")
		}
		tupleFields := append(make(Arguments, 0, len(t)+1),
			Argument{
				Name: "union_index",
				Type: Type{
					Size:       indexElemStrKindMarshaler.size,
					T:          UintTy,
					stringKind: string(indexElemStringKind),
				},
			},
		)
		for _, fieldT := range t {
			field, err := AbiTypeFromRideTypeMeta(fieldT)
			if err != nil {
				return Type{}, errors.Wrapf(err,
					"failed to create abi type for ride meta union type, field type %T", fieldT,
				)
			}
			tupleFields = append(tupleFields, Argument{
				Name: "",
				Type: field,
			})
		}
		abiT = Type{
			T:           TupleTy,
			TupleFields: tupleFields,
		}
	default:
		return Type{}, errors.Errorf("unsupported ride metadata type, type %T", t)
	}
	// TODO(nickeskov): Do we really need this? In result we have recursion inside recursion.
	stringKindMarshaler, err := rideMetaTypeToTextMarshaler(metaT)
	if err != nil {
		return Type{}, errors.Wrapf(err, "failed to create stringKind marshaler for ride meta type %T", metaT)
	}
	stringKind, err := stringKindMarshaler.MarshalText()
	if err != nil {
		return Type{}, errors.Wrapf(err, "failed to create stringKind for ride meta type %T", metaT)
	}
	abiT.stringKind = string(stringKind)

	return abiT, nil
}
