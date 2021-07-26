package fourbyte

import (
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/metamask"
	"math/big"
	"reflect"
)

type ArgT byte

// Type enumerator
const (
	IntTy ArgT = iota
	UintTy
	BoolTy

	StringTy
	SliceTy
	BytesTy
	//ArrayTy
	TupleTy

	AddressTy // nickeskov: we use this type only for erc20 transfers

	//FixedBytesTy
	//HashTy
	//FixedPointTy
	//FunctionTy
)

// Type is the reflection of the supported argument type.
type Type struct {
	Elem *Type // nested types for SliceTy
	Size int
	T    ArgT // Our own type checking

	stringKind string // holds the unparsed string for deriving signatures

	// Tuple relative fields
	TupleRawName  string       // Raw struct name defined in source code, may be empty.
	TupleElems    []*Type      // Type information of all tuple fields
	TupleRawNames []string     // Raw field name of all tuple fields
	TupleType     reflect.Type // Underlying struct of the tuple
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
		for _, elem := range t.TupleElems {
			total += getTypeSize(*elem)
		}
		return total
	}
	return 32
}

// GetType returns the reflection type of the ABI type.
func (t Type) GetType() reflect.Type {
	switch t.T {
	case IntTy:
		return reflectIntType(false, t.Size)
	case UintTy:
		return reflectIntType(true, t.Size)
	case BoolTy:
		return reflect.TypeOf(false)
	case StringTy:
		return reflect.TypeOf("")
	case SliceTy:
		return reflect.SliceOf(t.Elem.GetType())
	case TupleTy:
		return t.TupleType
	case AddressTy:
		// TODO(nickeskov): use our address
		return reflect.TypeOf(metamask.Address{})
	case BytesTy:
		return reflect.SliceOf(reflect.TypeOf(byte(0)))
	default:
		panic(fmt.Errorf("invalid ABI type (T=%d)", t.T))
	}
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
		for _, elem := range t.TupleElems {
			if isDynamicType(*elem) {
				return true
			}
		}
		return false
	}
	return t.T == StringTy || t.T == BytesTy || t.T == SliceTy
}

// reflectIntType returns the reflect using the given size and
// unsignedness.
func reflectIntType(unsigned bool, size int) reflect.Type {
	if unsigned {
		switch size {
		case 8:
			return reflect.TypeOf(uint8(0))
		case 16:
			return reflect.TypeOf(uint16(0))
		case 32:
			return reflect.TypeOf(uint32(0))
		case 64:
			return reflect.TypeOf(uint64(0))
		}
	}
	switch size {
	case 8:
		return reflect.TypeOf(int8(0))
	case 16:
		return reflect.TypeOf(int16(0))
	case 32:
		return reflect.TypeOf(int32(0))
	case 64:
		return reflect.TypeOf(int64(0))
	}
	return reflect.TypeOf(&big.Int{})
}
