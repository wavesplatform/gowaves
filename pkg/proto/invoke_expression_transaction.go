package proto

import (
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
	Expression string           `json:"expression,omitempty"`
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
	return nil, nil
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
	return nil
}

func (tx *InvokeExpressionTransactionWithProofs) MerkleBytes(scheme Scheme) ([]byte, error) {
	return tx.MarshalSignedToProtobuf(scheme)
}

func (tx *InvokeExpressionTransactionWithProofs) MarshalBinary() ([]byte, error) {
	return nil, nil
}

func (tx *InvokeExpressionTransactionWithProofs) UnmarshalBinary([]byte, Scheme) error {
	return nil
}

func (tx *InvokeExpressionTransactionWithProofs) BodyMarshalBinary() ([]byte, error) {
	return nil, nil
}

func (tx *InvokeExpressionTransactionWithProofs) BinarySize() int {
	return 1
}

func (tx *InvokeExpressionTransactionWithProofs) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return nil, nil
}

func (tx *InvokeExpressionTransactionWithProofs) UnmarshalFromProtobuf([]byte) error {
	return nil
}

func (tx *InvokeExpressionTransactionWithProofs) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return nil, nil
}

func (tx *InvokeExpressionTransactionWithProofs) UnmarshalSignedFromProtobuf([]byte) error {
	return nil
}
func (tx *InvokeExpressionTransactionWithProofs) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	return nil, nil
}
func (tx *InvokeExpressionTransactionWithProofs) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	return nil, nil
}
