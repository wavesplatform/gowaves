package ethabi

import "math/big"

type DataType interface{ _dataTypeMarker() }

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

func (Int) _dataTypeMarker()    {}
func (BigInt) _dataTypeMarker() {}
func (Bool) _dataTypeMarker()   {}
func (Bytes) _dataTypeMarker()  {}
func (String) _dataTypeMarker() {}
func (List) _dataTypeMarker()   {}
