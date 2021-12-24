package proto

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

type InvokeExpressionTransactionWithProofs struct {
	ID         *crypto.Digest   `json:"id,omitempty"`
	Type       TransactionType  `json:"type"`
	Version    byte             `json:"version,omitempty"`
	ChainID    byte             `json:"-"`
	SenderPK   crypto.PublicKey `json:"senderPublicKey"`
	Fee        uint64           `json:"fee"`
	FeeAsset   OptionalAsset    `json:"feeAssetId"`
	Timestamp  uint64           `json:"timestamp,omitempty"`
	Proofs     *ProofsV1        `json:"proofs,omitempty"`
	Expression []byte           `json:"expression,omitempty"`
}

func (tx InvokeExpressionTransactionWithProofs) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{tx.Type, Proof}
}

func (tx InvokeExpressionTransactionWithProofs) GetVersion() byte {
	return tx.Version
}

func (tx *InvokeExpressionTransactionWithProofs) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx InvokeExpressionTransactionWithProofs) GetSender(scheme Scheme) (Address, error) {
	return NewAddressFromPublicKey(scheme, tx.SenderPK)
}

func (tx InvokeExpressionTransactionWithProofs) GetFee() uint64 {
	return tx.Fee
}

func (tx InvokeExpressionTransactionWithProofs) GetTimestamp() uint64 {
	return tx.Timestamp
}

func (tx *InvokeExpressionTransactionWithProofs) Validate(scheme Scheme) (Transaction, error) {
	if tx.Version < 1 || tx.Version > MaxInvokeScriptTransactionVersion {
		return tx, errors.Errorf("unexpected version %d for InvokeExpression", tx.Version)
	}
	if tx.Fee == 0 {
		return tx, errors.New("fee should be positive")
	}
	if !validJVMLong(tx.Fee) {
		return tx, errors.New("fee is too big")
	}

	if tx.ChainID != scheme {
		return tx, errors.New("the chain id of InvokeExpression is not equal to network byte")
	}
	return tx, nil
}

func (tx *InvokeExpressionTransactionWithProofs) GenerateID(scheme Scheme) error {
	if tx.ID == nil {
		body, err := MarshalTxBody(scheme, tx)
		if err != nil {
			return err
		}
		id := crypto.MustFastHash(body)
		tx.ID = &id
	}
	return nil
}

func (tx *InvokeExpressionTransactionWithProofs) Sign(scheme Scheme, sk crypto.SecretKey) error {
	b, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return errors.Wrap(err, "failed to sign InvokeExpression transaction")
	}
	if tx.Proofs == nil {
		tx.Proofs = &ProofsV1{proofsVersion, make([]B58Bytes, 0)}
	}
	err = tx.Proofs.Sign(0, sk, b)
	if err != nil {
		return errors.Wrap(err, "failed to sign InvokeExpression transaction")
	}
	d, err := crypto.FastHash(b)
	if err != nil {
		return errors.Wrap(err, "failed to sign InvokeExpression transaction")
	}
	tx.ID = &d
	return nil
}

func (tx *InvokeExpressionTransactionWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *InvokeExpressionTransactionWithProofs) MarshalBinary() ([]byte, error) {
	return nil, errors.New("MarshalBinary is not implemented")
}

func (tx *InvokeExpressionTransactionWithProofs) UnmarshalBinary([]byte, Scheme) error {
	return errors.New("UnmarshalBinary is not implemented")
}

func (tx *InvokeExpressionTransactionWithProofs) BodyMarshalBinary() ([]byte, error) {
	return nil, errors.New("BodyMarshalBinary is not implemented")
}

// TODO check on correctness
func (tx *InvokeExpressionTransactionWithProofs) BinarySize() int {
	return 4 + tx.Proofs.BinarySize() + crypto.PublicKeySize + tx.FeeAsset.BinarySize() + 16 + len(tx.Expression)
}

func (tx *InvokeExpressionTransactionWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalTxDeterministic(tx, scheme)
}

func (tx *InvokeExpressionTransactionWithProofs) UnmarshalFromProtobuf(data []byte) error {
	t, err := TxFromProtobuf(data)
	if err != nil {
		return err
	}
	invokeExpressionTx, ok := t.(*InvokeExpressionTransactionWithProofs)
	if !ok {
		return errors.New("failed to convert result to InvokeScripV1")
	}
	*tx = *invokeExpressionTx
	return nil
}

func (tx *InvokeExpressionTransactionWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *InvokeExpressionTransactionWithProofs) UnmarshalSignedFromProtobuf(data []byte) error {
	t, err := SignedTxFromProtobuf(data)
	if err != nil {
		return err
	}
	invokeExpressionTx, ok := t.(*InvokeExpressionTransactionWithProofs)
	if !ok {
		return errors.New("failed to convert result to InvokeScriptWithProofs")
	}
	*tx = *invokeExpressionTx
	return nil
}
func (tx *InvokeExpressionTransactionWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {

	txData := &g.Transaction_InvokeExpression{InvokeExpression: &g.InvokeExpressionTransactionData{
		Expression: tx.Expression,
	}}
	fee := &g.Amount{AssetId: tx.FeeAsset.ToID(), Amount: int64(tx.Fee)}
	res := TransactionToProtobufCommon(scheme, tx.SenderPK.Bytes(), tx)
	res.Fee = fee
	res.Data = txData
	return res, nil
}
func (tx *InvokeExpressionTransactionWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	unsigned, err := tx.ToProtobuf(scheme)
	if err != nil {
		return nil, err
	}
	if tx.Proofs == nil {
		return nil, errors.New("no proofs provided")
	}
	return &g.SignedTransaction{
		Transaction: &g.SignedTransaction_WavesTransaction{WavesTransaction: unsigned},
		Proofs:      tx.Proofs.Bytes(),
	}, nil
}
