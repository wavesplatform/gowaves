package ethabi

import (
	"encoding/binary"
	"math/big"
)

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

const abiSlotSize = 32

func (i Int) encodeToABISlot() (slot [abiSlotSize]byte) {
	binary.BigEndian.PutUint64(slot[abiSlotSize-8:], uint64(i))
	return slot
}

func (i Int) EncodeToABI() []byte {
	s := i.encodeToABISlot()
	return s[:]
}

func (b Bool) EncodeToABI() []byte {
	var slot [abiSlotSize]byte
	if b {
		slot[abiSlotSize-1] = 1
	}
	return slot[:]
}

func (s String) EncodeToABI() []byte {
	l := len(s)
	strSlots := l / abiSlotSize
	if l-strSlots*abiSlotSize > 0 { // division rem
		strSlots += 1 // add slot
	}
	var (
		offset = Int(abiSlotSize).encodeToABISlot()
		size   = Int(l).encodeToABISlot()
	)
	outSize := (2 + strSlots) * abiSlotSize // offset slot + size slot + string slots
	out := make([]byte, 0, outSize)
	out = append(out, offset[:]...)
	out = append(out, size[:]...)
	out = append(out, s[:]...)
	return out[:outSize]
}
