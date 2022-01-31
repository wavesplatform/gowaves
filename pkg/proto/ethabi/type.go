package ethabi

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
)

var UnsupportedType = errors.New("unsupported type")

type ArgType byte

// Type enumerator
const (
	IntType ArgType = iota
	UintType
	BytesType
	BoolType
	StringType

	SliceType
	TupleType

	AddressType    // we use this type only for erc20 transfers
	FixedBytesType // we use this type only for payment asset
)

func (t ArgType) String() string {
	switch t {
	case IntType:
		return "IntType"
	case UintType:
		return "UintType"
	case BytesType:
		return "BytesType"
	case BoolType:
		return "BoolType"
	case StringType:
		return "StringType"
	case SliceType:
		return "SliceType"
	case TupleType:
		return "TupleType"
	case AddressType:
		return "AddressType"
	case FixedBytesType:
		return "FixedBytesType"
	default:
		return fmt.Sprintf("unknown ArgType (%d)", t)
	}
}

// Type is the reflection of the supported argument type.
type Type struct {
	Elem *Type // nested types for SliceType
	Size int
	T    ArgType // Our own type checking

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
	return t.T == StringType || t.T == BytesType || t.T == SliceType
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
	if t.T == TupleType && !isDynamicType(t) {
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
	if t.T == TupleType {
		for _, elem := range t.TupleFields {
			if isDynamicType(elem.Type) {
				return true
			}
		}
		return false
	}
	return t.T == StringType || t.T == BytesType || t.T == SliceType
}

func AbiTypeFromRideTypeMeta(metaT meta.Type) (abiT Type, err error) {
	switch t := metaT.(type) {
	case meta.SimpleType:
		switch t {
		case meta.Int:
			abiT = Type{T: IntType, Size: 64}
		case meta.Bytes:
			abiT = Type{T: BytesType}
		case meta.Boolean:
			abiT = Type{T: BoolType}
		case meta.String:
			abiT = Type{T: StringType}
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
		abiT = Type{Elem: &inner, T: SliceType}
	case meta.UnionType:
		return Type{}, errors.Wrap(UnsupportedType, "UnionType")
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
