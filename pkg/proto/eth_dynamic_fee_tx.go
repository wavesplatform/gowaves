package proto

import (
	"io"
	"math/big"

	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
)

type EthereumDynamicFeeTx struct {
	ChainID    *big.Int           // destination chain ID
	Nonce      uint64             // nonce of sender account
	GasTipCap  *big.Int           // EIP-1559 value
	GasFeeCap  *big.Int           // EIP-1559 value
	Gas        uint64             // gas limit
	To         *EthereumAddress   // nil means contract creation
	Value      *big.Int           // wei amount
	Data       []byte             // contract invocation input data
	AccessList EthereumAccessList // EIP-2930 access list
	V, R, S    *big.Int           // signature values
}

func (tx *EthereumDynamicFeeTx) unmarshalFromFastRLP(value *fastrlp.Value) error {
	const dynamicFeeTxFieldsCount = 12

	elems, err := value.GetElems()
	if err != nil {
		return errors.Wrap(err, "failed to get elements array")
	}

	if len(elems) != dynamicFeeTxFieldsCount {
		return errors.Errorf("expected %d elements, but recieved %d", dynamicFeeTxFieldsCount, len(elems))
	}

	var chainID big.Int
	if err := elems[0].GetBigInt(&chainID); err != nil {
		return errors.Wrap(err, "failed to parse ChainID")
	}

	nonce, err := elems[1].GetUint64()
	if err != nil {
		return errors.Wrap(err, "failed to parse nonce")
	}

	var gasTipCap big.Int
	if err := elems[2].GetBigInt(&gasTipCap); err != nil {
		return errors.Wrap(err, "failed to parse gasTipCap")
	}

	var gasFeeCap big.Int
	if err := elems[3].GetBigInt(&gasFeeCap); err != nil {
		return errors.Wrap(err, "failed to parse gasFeeCap")
	}

	gasLimit, err := elems[4].GetUint64()
	if err != nil {
		return errors.Wrap(err, "failed to parse Gas")
	}

	addrTo, err := unmarshalTransactionToFieldFastRLP(elems[5])
	if err != nil {
		return errors.Wrap(err, "failed to parse TO field")
	}

	var weiAmount big.Int
	if err := elems[6].GetBigInt(&weiAmount); err != nil {
		return errors.Wrap(err, "failed to parse wei amount")
	}

	contractData, err := elems[7].Bytes()
	if err != nil {
		return errors.Wrap(err, "failed to parse contract invocation input data")
	}

	accessList, err := unmarshalAccessListFastRLP(elems[8])
	if err != nil {
		return errors.Wrap(err, "failed to parse accessList")
	}

	v, r, s, err := unmarshalSignatureValuesFastRLP(elems[9], elems[10], elems[11])
	if err != nil {
		return errors.Wrap(err, "failed to parse signature value")
	}

	*tx = EthereumDynamicFeeTx{
		ChainID:    &chainID,
		Nonce:      nonce,
		GasTipCap:  &gasTipCap,
		GasFeeCap:  &gasFeeCap,
		Gas:        gasLimit,
		To:         addrTo,
		Value:      &weiAmount,
		Data:       contractData,
		AccessList: accessList,
		V:          &v,
		R:          &r,
		S:          &s,
	}
	return nil
}

func (tx *EthereumDynamicFeeTx) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewBigInt(tx.ChainID),
		arena.NewUint(tx.Nonce),
		arena.NewBigInt(tx.GasTipCap),
		arena.NewBigInt(tx.GasFeeCap),
		arena.NewUint(tx.Gas),
		arena.NewBytes(tx.To.tryToBytes()),
		arena.NewBigInt(tx.Value),
		arena.NewBytes(tx.Data),
		marshalAccessListToFastRLP(arena, tx.AccessList),
		arena.NewBigInt(tx.V),
		arena.NewBigInt(tx.R),
		arena.NewBigInt(tx.S),
	}

	array := arena.NewArray()
	for _, value := range values {
		array.Set(value)
	}
	return array
}

func (tx *EthereumDynamicFeeTx) DecodeRLP(rlpData []byte) error {
	parser := fastrlp.Parser{}
	rlpVal, err := parser.Parse(rlpData)
	if err != nil {
		return err
	}
	if err := tx.unmarshalFromFastRLP(rlpVal); err != nil {
		return errors.Wrap(err, "failed to parse EthereumDynamicFeeTx from RLP encoded data")
	}
	return nil
}

func (tx *EthereumDynamicFeeTx) EncodeRLP(w io.Writer) error {
	arena := fastrlp.Arena{}
	rlpVal := tx.marshalToFastRLP(&arena)
	rlpData := rlpVal.MarshalTo(nil)
	if _, err := w.Write(rlpData); err != nil {
		return err
	}
	return nil
}

func (tx *EthereumDynamicFeeTx) copy() EthereumTxData {
	return &EthereumDynamicFeeTx{
		ChainID:    copyBigInt(tx.ChainID),
		Nonce:      tx.Nonce,
		GasTipCap:  copyBigInt(tx.GasTipCap),
		GasFeeCap:  copyBigInt(tx.GasFeeCap),
		Gas:        tx.Gas,
		To:         tx.To.copy(),
		Value:      copyBigInt(tx.Value),
		Data:       copyBytes(tx.Data),
		AccessList: tx.AccessList.copy(),
		V:          copyBigInt(tx.V),
		R:          copyBigInt(tx.R),
		S:          copyBigInt(tx.S),
	}
}

// accessors for innerTx.
func (tx *EthereumDynamicFeeTx) ethereumTxType() EthereumTxType { return EthereumDynamicFeeTxType }
func (tx *EthereumDynamicFeeTx) chainID() *big.Int              { return tx.ChainID }
func (tx *EthereumDynamicFeeTx) accessList() EthereumAccessList { return tx.AccessList }
func (tx *EthereumDynamicFeeTx) data() []byte                   { return tx.Data }
func (tx *EthereumDynamicFeeTx) gas() uint64                    { return tx.Gas }
func (tx *EthereumDynamicFeeTx) gasFeeCap() *big.Int            { return tx.GasFeeCap }
func (tx *EthereumDynamicFeeTx) gasTipCap() *big.Int            { return tx.GasTipCap }
func (tx *EthereumDynamicFeeTx) gasPrice() *big.Int             { return tx.GasFeeCap }
func (tx *EthereumDynamicFeeTx) value() *big.Int                { return tx.Value }
func (tx *EthereumDynamicFeeTx) nonce() uint64                  { return tx.Nonce }
func (tx *EthereumDynamicFeeTx) to() *EthereumAddress           { return tx.To }

func (tx *EthereumDynamicFeeTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *EthereumDynamicFeeTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}

func (tx *EthereumDynamicFeeTx) signerHashFastRLP(chainID *big.Int, arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewBigInt(chainID),
		arena.NewUint(tx.Nonce),
		arena.NewBigInt(tx.GasTipCap),
		arena.NewBigInt(tx.GasFeeCap),
		arena.NewUint(tx.Gas),
		arena.NewBytes(tx.To.tryToBytes()),
		arena.NewBigInt(tx.Value),
		arena.NewBytes(tx.Data),
		marshalAccessListToFastRLP(arena, tx.AccessList),
	}

	array := arena.NewArray()
	for _, value := range values {
		array.Set(value)
	}
	return array
}
