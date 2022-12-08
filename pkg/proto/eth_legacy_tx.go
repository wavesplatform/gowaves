package proto

import (
	"io"
	"math/big"

	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
)

// EthereumLegacyTx is the transaction data of regular Ethereum transactions.
type EthereumLegacyTx struct {
	Nonce    uint64           // nonce of sender account
	GasPrice *big.Int         // wei per gas
	Gas      uint64           // gas limit
	To       *EthereumAddress // nil value means contract creation
	Value    *big.Int         // wei amount
	Data     []byte           // contract invocation input data
	V, R, S  *big.Int         // signature values
}

func (tx *EthereumLegacyTx) unmarshalFromFastRLP(value *fastrlp.Value) error {
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

	v, r, s, err := unmarshalSignatureValuesFastRLP(elems[6], elems[7], elems[8])
	if err != nil {
		return errors.Wrap(err, "failed to parse signature value")
	}

	*tx = EthereumLegacyTx{
		Nonce:    nonce,
		GasPrice: &gasPrice,
		Gas:      gasLimit,
		To:       addrTo,
		Value:    &weiAmount,
		Data:     contractData,
		V:        &v,
		R:        &r,
		S:        &s,
	}
	return nil
}

func (tx *EthereumLegacyTx) marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewUint(tx.Nonce),
		arena.NewBigInt(tx.GasPrice),
		arena.NewUint(tx.Gas),
		arena.NewBytes(tx.To.tryToBytes()),
		arena.NewBigInt(tx.Value),
		arena.NewBytes(tx.Data),
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

func (tx *EthereumLegacyTx) DecodeRLP(rlpData []byte) error {
	parser := fastrlp.Parser{}
	rlpVal, err := parser.Parse(rlpData)
	if err != nil {
		return err
	}
	if err := tx.unmarshalFromFastRLP(rlpVal); err != nil {
		return errors.Wrap(err, "failed to parse EthereumLegacyTx from RLP encoded data")
	}
	return nil
}

func (tx *EthereumLegacyTx) EncodeRLP(w io.Writer) error {
	arena := fastrlp.Arena{}
	rlpVal := tx.marshalToFastRLP(&arena)
	rlpData := rlpVal.MarshalTo(nil)
	if _, err := w.Write(rlpData); err != nil {
		return err
	}
	return nil
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *EthereumLegacyTx) copy() EthereumTxData {
	return &EthereumLegacyTx{
		Nonce:    tx.Nonce,
		GasPrice: copyBigInt(tx.GasPrice),
		Gas:      tx.Gas,
		To:       tx.To.copy(),
		Value:    copyBigInt(tx.Value),
		Data:     copyBytes(tx.Data),
		V:        copyBigInt(tx.V),
		R:        copyBigInt(tx.R),
		S:        copyBigInt(tx.S),
	}
}

// accessors for innerTx.
func (tx *EthereumLegacyTx) ethereumTxType() EthereumTxType { return EthereumLegacyTxType }
func (tx *EthereumLegacyTx) chainID() *big.Int              { return deriveChainId(tx.V) }
func (tx *EthereumLegacyTx) accessList() EthereumAccessList { return nil }
func (tx *EthereumLegacyTx) data() []byte                   { return tx.Data }
func (tx *EthereumLegacyTx) gas() uint64                    { return tx.Gas }
func (tx *EthereumLegacyTx) gasPrice() *big.Int             { return tx.GasPrice }
func (tx *EthereumLegacyTx) gasTipCap() *big.Int            { return tx.GasPrice }
func (tx *EthereumLegacyTx) gasFeeCap() *big.Int            { return tx.GasPrice }
func (tx *EthereumLegacyTx) value() *big.Int                { return tx.Value }
func (tx *EthereumLegacyTx) nonce() uint64                  { return tx.Nonce }
func (tx *EthereumLegacyTx) to() *EthereumAddress           { return tx.To }

func (tx *EthereumLegacyTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *EthereumLegacyTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.V, tx.R, tx.S = v, r, s
}

func (tx *EthereumLegacyTx) signerHashFastRLP(chainID *big.Int, arena *fastrlp.Arena) *fastrlp.Value {
	values := [...]*fastrlp.Value{
		arena.NewUint(tx.Nonce),
		arena.NewBigInt(tx.GasPrice),
		arena.NewUint(tx.Gas),
		arena.NewBytes(tx.To.tryToBytes()),
		arena.NewBigInt(tx.Value),
		arena.NewBytes(tx.Data),
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
