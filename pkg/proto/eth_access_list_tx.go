package proto

import (
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"io"
	"math/big"
)

// EthereumAccessList is an EIP-2930 access list.
type EthereumAccessList []EthereumAccessTuple

func (al EthereumAccessList) copy() EthereumAccessList {
	if al == nil {
		return nil
	}
	cpy := make(EthereumAccessList, len(al))
	copy(cpy, al)
	return cpy
}

// EthereumAccessTuple is the element type of an access list.
type EthereumAccessTuple struct {
	Address     EthereumAddress `json:"address"`
	StorageKeys []EthereumHash  `json:"storageKeys"`
}

func (at *EthereumAccessTuple) unmarshalFromFastRLP(value *fastrlp.Value) error {
	const accessTupleFieldsCount = 2

	elems, err := value.GetElems()
	if err != nil {
		return errors.Wrapf(err, "expected RLP Array, but received %q", value.Type().String())
	}
	if len(elems) != accessTupleFieldsCount {
		return errors.Errorf("expected %d elements, but recieved %d", accessTupleFieldsCount, len(elems))
	}

	var address EthereumAddress
	if err := address.unmarshalFromFastRLP(elems[0]); err != nil {
		return errors.Wrap(err, "failed to unmarshal EthereumAddress to fastRLP value for EthereumAccessTuple")
	}

	storageKeys, err := unmarshalHashesFromFastRLP(elems[1])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal StorageKeys to fastRLP value for EthereumAccessTuple")
	}

	*at = EthereumAccessTuple{
		Address:     address,
		StorageKeys: storageKeys,
	}
	return nil
}

func (at *EthereumAccessTuple) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	array := arena.NewArray()

	rlpAddr := at.Address.marshalToFastRLP(arena)
	array.Set(rlpAddr)

	storageKeys := marshalHashesToFastRLP(arena, at.StorageKeys)
	array.Set(storageKeys)

	return array
}

func unmarshalAccessListFastRLP(value *fastrlp.Value) (EthereumAccessList, error) {
	elems, err := value.GetElems()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get elements array")
	}
	hashes := make(EthereumAccessList, 0, len(elems))
	for _, elem := range elems {
		var accessTuple EthereumAccessTuple
		if err := accessTuple.unmarshalFromFastRLP(elem); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal EthereumAccessTuple from fastRLP value")
		}
		hashes = append(hashes, accessTuple)
	}
	return hashes, nil
}

func marshalAccessListToFastRLP(arena *fastrlp.Arena, accessList EthereumAccessList) *fastrlp.Value {
	array := arena.NewArray()
	for _, accessTuple := range accessList {
		val := accessTuple.marshalToFastRLP(arena)
		array.Set(val)
	}
	return array
}

// EthereumAccessListTx is the data of EIP-2930 access list transactions.
type EthereumAccessListTx struct {
	ChainID    *big.Int           // destination chain ID
	Nonce      uint64             // nonce of sender account
	GasPrice   *big.Int           // wei per gas
	Gas        uint64             // gas limit
	To         *EthereumAddress   // nil value means contract creation
	Value      *big.Int           // wei amount
	Data       []byte             // contract invocation input data
	AccessList EthereumAccessList // EIP-2930 access list
	V, R, S    *big.Int           // signature values
}

func (altx *EthereumAccessListTx) unmarshalFromFastRLP(value *fastrlp.Value) error {
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

	*altx = EthereumAccessListTx{
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

func (altx *EthereumAccessListTx) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewBigInt(altx.ChainID),
		arena.NewUint(altx.Nonce),
		arena.NewBigInt(altx.GasPrice),
		arena.NewUint(altx.Gas),
		arena.NewBytes(altx.To.tryToBytes()),
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

func (altx *EthereumAccessListTx) DecodeRLP(rlpData []byte) error {
	parser := fastrlp.Parser{}
	rlpVal, err := parser.Parse(rlpData)
	if err != nil {
		return err
	}
	if err := altx.unmarshalFromFastRLP(rlpVal); err != nil {
		return errors.Wrap(err, "failed to parse EthereumAccessListTx from RLP encoded data")
	}
	return nil
}

func (altx *EthereumAccessListTx) EncodeRLP(w io.Writer) error {
	arena := fastrlp.Arena{}
	rlpVal := altx.marshalToFastRLP(&arena)
	rlpData := rlpVal.MarshalTo(nil)
	if _, err := w.Write(rlpData); err != nil {
		return err
	}
	return nil
}

func (altx *EthereumAccessListTx) copy() EthereumTxData {
	return &EthereumAccessListTx{
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
func (altx *EthereumAccessListTx) txType() TxType                 { return AccessListTxType }
func (altx *EthereumAccessListTx) chainID() *big.Int              { return altx.ChainID }
func (altx *EthereumAccessListTx) accessList() EthereumAccessList { return altx.AccessList }
func (altx *EthereumAccessListTx) data() []byte                   { return altx.Data }
func (altx *EthereumAccessListTx) gas() uint64                    { return altx.Gas }
func (altx *EthereumAccessListTx) gasPrice() *big.Int             { return altx.GasPrice }
func (altx *EthereumAccessListTx) gasTipCap() *big.Int            { return altx.GasPrice }
func (altx *EthereumAccessListTx) gasFeeCap() *big.Int            { return altx.GasPrice }
func (altx *EthereumAccessListTx) value() *big.Int                { return altx.Value }
func (altx *EthereumAccessListTx) nonce() uint64                  { return altx.Nonce }
func (altx *EthereumAccessListTx) to() *EthereumAddress           { return altx.To }

func (altx *EthereumAccessListTx) rawSignatureValues() (v, r, s *big.Int) {
	return altx.V, altx.R, altx.S
}

func (altx *EthereumAccessListTx) setSignatureValues(chainID, v, r, s *big.Int) {
	altx.ChainID, altx.V, altx.R, altx.S = chainID, v, r, s
}

func (altx *EthereumAccessListTx) signerHashFastRLP(chainID *big.Int, arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewBigInt(chainID),
		arena.NewUint(altx.Nonce),
		arena.NewBigInt(altx.GasPrice),
		arena.NewUint(altx.Gas),
		arena.NewBytes(altx.To.tryToBytes()),
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
