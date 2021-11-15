package metamask

import (
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"io"
	"math/big"
)

// LegacyTx is the transaction data of regular Ethereum transactions.
type LegacyTx struct {
	Nonce    uint64   // nonce of sender account
	GasPrice *big.Int // wei per gas
	Gas      uint64   // gas limit
	To       *Address // nil value means contract creation
	Value    *big.Int // wei amount
	Data     []byte   // contract invocation input data
	V, R, S  *big.Int // signature values
}

func (ltx *LegacyTx) UnmarshalFromFastRLP(value *fastrlp.Value) error {
	const legacyTxFieldsCount = 9

	elems, err := value.GetElems()
	if err != nil {
		return errors.Wrap(err, "failed to get elements array")
	}

	if len(elems) != legacyTxFieldsCount {
		return errors.Errorf("expected %d elements, but recieved %d", legacyTxFieldsCount, len(elems))
	}

	nonce, err := elems[0].GetUint64()
	if err != nil {
		return errors.Wrap(err, "failed to parse nonce")
	}

	var gasPrice big.Int
	if err := elems[1].GetBigInt(&gasPrice); err != nil {
		return errors.Wrap(err, "failed to parse GasPrice")
	}

	gasLimit, err := elems[2].GetUint64()
	if err != nil {
		return errors.Wrap(err, "failed to parse Gas")
	}

	addrTo, err := unmarshalTransactionToFieldFastRLP(elems[3])
	if err != nil {
		return errors.Wrap(err, "failed to parse TO field")
	}

	var weiAmount big.Int
	if err := elems[4].GetBigInt(&weiAmount); err != nil {
		return errors.Wrap(err, "failed to parse wei amount")
	}

	contractData, err := elems[5].Bytes()
	if err != nil {
		return errors.Wrap(err, "failed to parse contract invocation input data")
	}

	V, R, S, err := unmarshalSignatureValuesFastRLP(elems[6], elems[7], elems[8])
	if err != nil {
		return errors.Wrap(err, "failed to parse signature value")
	}

	*ltx = LegacyTx{
		Nonce:    nonce,
		GasPrice: &gasPrice,
		Gas:      gasLimit,
		To:       addrTo,
		Value:    &weiAmount,
		Data:     contractData,
		V:        &V,
		R:        &R,
		S:        &S,
	}
	return nil
}

func (ltx *LegacyTx) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewUint(ltx.Nonce),
		arena.NewBigInt(ltx.GasPrice),
		arena.NewUint(ltx.Gas),
		arena.NewBytes(ltx.To.Bytes()),
		arena.NewBigInt(ltx.Value),
		arena.NewBytes(ltx.Data),
		arena.NewBigInt(ltx.V),
		arena.NewBigInt(ltx.R),
		arena.NewBigInt(ltx.S),
	}

	array := arena.NewArray()
	for _, value := range values {
		array.Set(value)
	}
	return array
}

func (ltx *LegacyTx) DecodeRLP(rlpData []byte) error {
	parser := fastrlp.Parser{}
	rlpVal, err := parser.Parse(rlpData)
	if err != nil {
		return err
	}
	if err := ltx.UnmarshalFromFastRLP(rlpVal); err != nil {
		return errors.Wrap(err, "failed to parse LegacyTx from RLP encoded data")
	}
	return nil
}

func (ltx *LegacyTx) EncodeRLP(w io.Writer) error {
	arena := fastrlp.Arena{}
	rlpVal := ltx.marshalToFastRLP(&arena)
	rlpData := rlpVal.MarshalTo(nil)
	if _, err := w.Write(rlpData); err != nil {
		return err
	}
	return nil
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (ltx *LegacyTx) copy() TxData {
	return &LegacyTx{
		Nonce:    ltx.Nonce,
		GasPrice: copyBigInt(ltx.GasPrice),
		Gas:      ltx.Gas,
		To:       ltx.To.copy(),
		Value:    copyBigInt(ltx.Value),
		Data:     copyBytes(ltx.Data),
		V:        copyBigInt(ltx.V),
		R:        copyBigInt(ltx.R),
		S:        copyBigInt(ltx.S),
	}
}

// accessors for innerTx.
func (ltx *LegacyTx) txType() byte           { return LegacyTxType }
func (ltx *LegacyTx) chainID() *big.Int      { return deriveChainId(ltx.V) }
func (ltx *LegacyTx) accessList() AccessList { return nil }
func (ltx *LegacyTx) data() []byte           { return ltx.Data }
func (ltx *LegacyTx) gas() uint64            { return ltx.Gas }
func (ltx *LegacyTx) gasPrice() *big.Int     { return ltx.GasPrice }
func (ltx *LegacyTx) gasTipCap() *big.Int    { return ltx.GasPrice }
func (ltx *LegacyTx) gasFeeCap() *big.Int    { return ltx.GasPrice }
func (ltx *LegacyTx) value() *big.Int        { return ltx.Value }
func (ltx *LegacyTx) nonce() uint64          { return ltx.Nonce }
func (ltx *LegacyTx) to() *Address           { return ltx.To }

func (ltx *LegacyTx) rawSignatureValues() (v, r, s *big.Int) {
	return ltx.V, ltx.R, ltx.S
}

func (ltx *LegacyTx) setSignatureValues(chainID, v, r, s *big.Int) {
	ltx.V, ltx.R, ltx.S = v, r, s
}

func (ltx *LegacyTx) signerHashFastRLP(chainID *big.Int, arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewUint(ltx.Nonce),
		arena.NewBigInt(ltx.GasPrice),
		arena.NewUint(ltx.Gas),
		arena.NewBytes(ltx.To.Bytes()),
		arena.NewBigInt(ltx.Value),
		arena.NewBytes(ltx.Data),
		arena.NewBigInt(chainID),
		arena.NewUint(0),
		arena.NewUint(0),
	}

	array := arena.NewArray()
	for _, value := range values {
		array.Set(value)
	}
	return array
}
