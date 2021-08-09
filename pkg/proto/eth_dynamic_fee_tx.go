package proto

import (
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"io"
	"math/big"
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

func (dftx *EthereumDynamicFeeTx) unmarshalFromFastRLP(value *fastrlp.Value) error {
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

	V, R, S, err := unmarshalSignatureValuesFastRLP(elems[9], elems[10], elems[11])
	if err != nil {
		return errors.Wrap(err, "failed to parse signature value")
	}

	*dftx = EthereumDynamicFeeTx{
		ChainID:    &chainID,
		Nonce:      nonce,
		GasTipCap:  &gasTipCap,
		GasFeeCap:  &gasFeeCap,
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

func (dftx *EthereumDynamicFeeTx) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewBigInt(dftx.ChainID),
		arena.NewUint(dftx.Nonce),
		arena.NewBigInt(dftx.GasTipCap),
		arena.NewBigInt(dftx.GasFeeCap),
		arena.NewUint(dftx.Gas),
		arena.NewBytes(dftx.To.tryToBytes()),
		arena.NewBigInt(dftx.Value),
		arena.NewBytes(dftx.Data),
		marshalAccessListToFastRLP(arena, dftx.AccessList),
		arena.NewBigInt(dftx.V),
		arena.NewBigInt(dftx.R),
		arena.NewBigInt(dftx.S),
	}

	array := arena.NewArray()
	for _, value := range values {
		array.Set(value)
	}
	return array
}

func (dftx *EthereumDynamicFeeTx) DecodeRLP(rlpData []byte) error {
	parser := fastrlp.Parser{}
	rlpVal, err := parser.Parse(rlpData)
	if err != nil {
		return err
	}
	if err := dftx.unmarshalFromFastRLP(rlpVal); err != nil {
		return errors.Wrap(err, "failed to parse EthereumDynamicFeeTx from RLP encoded data")
	}
	return nil
}

func (dftx *EthereumDynamicFeeTx) EncodeRLP(w io.Writer) error {
	arena := fastrlp.Arena{}
	rlpVal := dftx.marshalToFastRLP(&arena)
	rlpData := rlpVal.MarshalTo(nil)
	if _, err := w.Write(rlpData); err != nil {
		return err
	}
	return nil
}

func (dftx *EthereumDynamicFeeTx) copy() EthereumTxData {
	return &EthereumDynamicFeeTx{
		ChainID:    copyBigInt(dftx.ChainID),
		Nonce:      dftx.Nonce,
		GasTipCap:  copyBigInt(dftx.GasTipCap),
		GasFeeCap:  copyBigInt(dftx.GasFeeCap),
		Gas:        dftx.Gas,
		To:         dftx.To.copy(),
		Value:      copyBigInt(dftx.Value),
		Data:       copyBytes(dftx.Data),
		AccessList: dftx.AccessList.copy(),
		V:          copyBigInt(dftx.V),
		R:          copyBigInt(dftx.R),
		S:          copyBigInt(dftx.S),
	}
}

// accessors for innerTx.
func (dftx *EthereumDynamicFeeTx) ethereumTxType() EthereumTxType { return DynamicFeeTxType }
func (dftx *EthereumDynamicFeeTx) chainID() *big.Int              { return dftx.ChainID }
func (dftx *EthereumDynamicFeeTx) accessList() EthereumAccessList { return dftx.AccessList }
func (dftx *EthereumDynamicFeeTx) data() []byte                   { return dftx.Data }
func (dftx *EthereumDynamicFeeTx) gas() uint64                    { return dftx.Gas }
func (dftx *EthereumDynamicFeeTx) gasFeeCap() *big.Int            { return dftx.GasFeeCap }
func (dftx *EthereumDynamicFeeTx) gasTipCap() *big.Int            { return dftx.GasTipCap }
func (dftx *EthereumDynamicFeeTx) gasPrice() *big.Int             { return dftx.GasFeeCap }
func (dftx *EthereumDynamicFeeTx) value() *big.Int                { return dftx.Value }
func (dftx *EthereumDynamicFeeTx) nonce() uint64                  { return dftx.Nonce }
func (dftx *EthereumDynamicFeeTx) to() *EthereumAddress           { return dftx.To }

func (dftx *EthereumDynamicFeeTx) rawSignatureValues() (v, r, s *big.Int) {
	return dftx.V, dftx.R, dftx.S
}

func (dftx *EthereumDynamicFeeTx) setSignatureValues(chainID, v, r, s *big.Int) {
	dftx.ChainID, dftx.V, dftx.R, dftx.S = chainID, v, r, s
}

func (dftx *EthereumDynamicFeeTx) signerHashFastRLP(chainID *big.Int, arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewBigInt(chainID),
		arena.NewUint(dftx.Nonce),
		arena.NewBigInt(dftx.GasTipCap),
		arena.NewBigInt(dftx.GasFeeCap),
		arena.NewUint(dftx.Gas),
		arena.NewBytes(dftx.To.tryToBytes()),
		arena.NewBigInt(dftx.Value),
		arena.NewBytes(dftx.Data),
		marshalAccessListToFastRLP(arena, dftx.AccessList),
	}

	array := arena.NewArray()
	for _, value := range values {
		array.Set(value)
	}
	return array
}
