package ethabi

import "math/big"

type DataType interface{ ethABIDataTypeMarker() }

type (
	Int    int64
	BigInt struct {
		V *big.Int
	}
	Bool   bool
	Bytes  []byte
	String string
	List   []DataType
)

func (Int) ethABIDataTypeMarker()    {}
func (BigInt) ethABIDataTypeMarker() {}
func (Bool) ethABIDataTypeMarker()   {}
func (Bytes) ethABIDataTypeMarker()  {}
func (String) ethABIDataTypeMarker() {}
func (List) ethABIDataTypeMarker()   {}
