package metamask

import (
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"io"
	"math/big"
)

// AccessList is an EIP-2930 access list.
type AccessList []AccessTuple

func (al AccessList) copy() AccessList {
	if al == nil {
		return nil
	}
	cpy := make(AccessList, len(al))
	copy(cpy, al)
	return cpy
}

// AccessTuple is the element type of an access list.
type AccessTuple struct {
	Address     Address `json:"address"`
	StorageKeys []Hash  `json:"storageKeys"`
}

func (at *AccessTuple) unmarshalFromFastRLP(value *fastrlp.Value) error {
	const accessTupleFieldsCount = 2

	elems, err := value.GetElems()
	if err != nil {
		return errors.Wrapf(err, "expected RLP Array, but received %q", value.Type().String())
	}
	if len(elems) != accessTupleFieldsCount {
		return errors.Errorf("expected %d elements, but recieved %d", accessTupleFieldsCount, len(elems))
	}

	var address Address
	if err := address.unmarshalFromFastRLP(elems[0]); err != nil {
		return errors.Wrap(err, "failed to unmarshal Address to fastRLP value for AccessTuple")
	}

	storageKeys, err := unmarshalHashesFromFastRLP(elems[1])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal StorageKeys to fastRLP value for AccessTuple")
	}

	*at = AccessTuple{
		Address:     address,
		StorageKeys: storageKeys,
	}
	return nil
}

func (at *AccessTuple) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	array := arena.NewArray()

	rlpAddr := at.Address.marshalToFastRLP(arena)
	array.Set(rlpAddr)

	storageKeys := marshalHashesToFastRLP(arena, at.StorageKeys)
	array.Set(storageKeys)

	return array
}

func unmarshalAccessListFastRLP(value *fastrlp.Value) (AccessList, error) {
	elems, err := value.GetElems()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get elements array")
	}
	hashes := make(AccessList, 0, len(elems))
	for _, elem := range elems {
		var accessTuple AccessTuple
		if err := accessTuple.unmarshalFromFastRLP(elem); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal AccessTuple from fastRLP value")
		}
		hashes = append(hashes, accessTuple)
	}
	return hashes, nil
}

func marshalAccessListToFastRLP(arena *fastrlp.Arena, accessList AccessList) *fastrlp.Value {
	array := arena.NewArray()
	for _, accessTuple := range accessList {
		val := accessTuple.marshalToFastRLP(arena)
		array.Set(val)
	}
	return array
}

// AccessListTx is the data of EIP-2930 access list transactions.
type AccessListTx struct {
	ChainID    *big.Int   // destination chain ID
	Nonce      uint64     // nonce of sender account
	GasPrice   *big.Int   // wei per gas
	Gas        uint64     // gas limit
	To         *Address   // nil value means contract creation
	Value      *big.Int   // wei amount
	Data       []byte     // contract invocation input data
	AccessList AccessList // EIP-2930 access list
	V, R, S    *big.Int   // signature values
}

func (altx *AccessListTx) unmarshalFromFastRLP(value *fastrlp.Value) error {
	const accessListTxFieldsCount = 11

	elems, err := value.GetElems()
	if err != nil {
		return errors.Wrap(err, "failed to get elements array")
	}

	if len(elems) != accessListTxFieldsCount {
		return errors.Errorf("expected %d elements, but recieved %d", accessListTxFieldsCount, len(elems))
	}

	var chainID big.Int
	if err := elems[0].GetBigInt(&chainID); err != nil {
		return errors.Wrap(err, "failed to parse ChainID")
	}

	nonce, err := elems[1].GetUint64()
	if err != nil {
		return errors.Wrap(err, "failed to parse nonce")
	}

	var gasPrice big.Int
	if err := elems[2].GetBigInt(&gasPrice); err != nil {
		return errors.Wrap(err, "failed to parse GasPrice")
	}

	gasLimit, err := elems[3].GetUint64()
	if err != nil {
		return errors.Wrap(err, "failed to parse Gas")
	}

	addrTo, err := unmarshalTransactionToFieldFastRLP(elems[4])
	if err != nil {
		return errors.Wrap(err, "failed to parse TO field")
	}

	var weiAmount big.Int
	if err := elems[5].GetBigInt(&weiAmount); err != nil {
		return errors.Wrap(err, "failed to parse wei amount")
	}

	contractData, err := elems[6].Bytes()
	if err != nil {
		return errors.Wrap(err, "failed to parse contract invocation input data")
	}

	accessList, err := unmarshalAccessListFastRLP(elems[7])
	if err != nil {
		return errors.Wrap(err, "failed to parse accessList")
	}

	V, R, S, err := unmarshalSignatureValuesFastRLP(elems[8], elems[9], elems[10])
	if err != nil {
		return errors.Wrap(err, "failed to parse signature value")
	}

	*altx = AccessListTx{
		ChainID:    &chainID,
		Nonce:      nonce,
		GasPrice:   &gasPrice,
		Gas:        gasLimit,
		To:         addrTo,
		Value:      &weiAmount,
		Data:       contractData,
		AccessList: accessList,
		V:          &V,
		R:          &R,
		S:          &S,
	}
	return nil
}

func (altx *AccessListTx) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewBigInt(altx.ChainID),
		arena.NewUint(altx.Nonce),
		arena.NewBigInt(altx.GasPrice),
		arena.NewUint(altx.Gas),
		arena.NewBytes(altx.To.Bytes()),
		arena.NewBigInt(altx.Value),
		arena.NewBytes(altx.Data),
		marshalAccessListToFastRLP(arena, altx.AccessList),
		arena.NewBigInt(altx.V),
		arena.NewBigInt(altx.R),
		arena.NewBigInt(altx.S),
	}

	array := arena.NewArray()
	for _, value := range values {
		array.Set(value)
	}
	return array
}

func (altx *AccessListTx) DecodeRLP(rlpData []byte) error {
	parser := fastrlp.Parser{}
	rlpVal, err := parser.Parse(rlpData)
	if err != nil {
		return err
	}
	if err := altx.unmarshalFromFastRLP(rlpVal); err != nil {
		return errors.Wrap(err, "failed to parse AccessListTx from RLP encoded data")
	}
	return nil
}

func (altx *AccessListTx) EncodeRLP(w io.Writer) error {
	arena := fastrlp.Arena{}
	rlpVal := altx.marshalToFastRLP(&arena)
	rlpData := rlpVal.MarshalTo(nil)
	if _, err := w.Write(rlpData); err != nil {
		return err
	}
	return nil
}

func (altx *AccessListTx) copy() TxData {
	return &AccessListTx{
		ChainID:    copyBigInt(altx.ChainID),
		Nonce:      altx.Nonce,
		GasPrice:   copyBigInt(altx.GasPrice),
		Gas:        altx.Gas,
		To:         altx.To.copy(),
		Value:      copyBigInt(altx.Value),
		Data:       copyBytes(altx.Data),
		AccessList: altx.AccessList.copy(),
		V:          copyBigInt(altx.V),
		R:          copyBigInt(altx.R),
		S:          copyBigInt(altx.S),
	}
}

// accessors for innerTx.
func (altx *AccessListTx) txType() byte           { return AccessListTxType }
func (altx *AccessListTx) chainID() *big.Int      { return altx.ChainID }
func (altx *AccessListTx) accessList() AccessList { return altx.AccessList }
func (altx *AccessListTx) data() []byte           { return altx.Data }
func (altx *AccessListTx) gas() uint64            { return altx.Gas }
func (altx *AccessListTx) gasPrice() *big.Int     { return altx.GasPrice }
func (altx *AccessListTx) gasTipCap() *big.Int    { return altx.GasPrice }
func (altx *AccessListTx) gasFeeCap() *big.Int    { return altx.GasPrice }
func (altx *AccessListTx) value() *big.Int        { return altx.Value }
func (altx *AccessListTx) nonce() uint64          { return altx.Nonce }
func (altx *AccessListTx) to() *Address           { return altx.To }

func (altx *AccessListTx) rawSignatureValues() (v, r, s *big.Int) {
	return altx.V, altx.R, altx.S
}

func (altx *AccessListTx) setSignatureValues(chainID, v, r, s *big.Int) {
	altx.ChainID, altx.V, altx.R, altx.S = chainID, v, r, s
}

func (altx *AccessListTx) signerHashFastRLP(chainID *big.Int, arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewBigInt(chainID),
		arena.NewUint(altx.Nonce),
		arena.NewBigInt(altx.GasPrice),
		arena.NewUint(altx.Gas),
		arena.NewBytes(altx.To.Bytes()),
		arena.NewBigInt(altx.Value),
		arena.NewBytes(altx.Data),
		marshalAccessListToFastRLP(arena, altx.AccessList),
	}

	array := arena.NewArray()
	for _, value := range values {
		array.Set(value)
	}
	return array
}
