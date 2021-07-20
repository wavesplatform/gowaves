package fourbyte

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
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
	//TupleTy

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

	//// Tuple relative fields
	//TupleRawName  string       // Raw struct name defined in source code, may be empty.
	//TupleElems    []*Type      // Type information of all tuple fields
	//TupleRawNames []string     // Raw field name of all tuple fields
	//TupleType     reflect.Type // Underlying struct of the tuple
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
// TODO(nickeskov): remove this
func getTypeSize(t Type) int {
	//if t.T == ArrayTy && !isDynamicType(*t.Elem) {
	//	// Recursively calculate type size if it is a nested array
	//	if t.Elem.T == ArrayTy || t.Elem.T == TupleTy {
	//		return t.Size * getTypeSize(*t.Elem)
	//	}
	//	return t.Size * 32
	//} else if t.T == TupleTy && !isDynamicType(t) {
	//	total := 0
	//	for _, elem := range t.TupleElems {
	//		total += getTypeSize(*elem)
	//	}
	//	return total
	//}
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
	case AddressTy:
		// TODO(nickeskov): use our address
		return reflect.TypeOf(common.Address{})
	case BytesTy:
		return reflect.SliceOf(reflect.TypeOf(byte(0)))
	default:
		panic(fmt.Errorf("invalid ABI type (T=%d)", t.T))
	}
}
