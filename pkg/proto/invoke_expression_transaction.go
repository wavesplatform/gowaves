package proto

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

type InvokeExpressionTransaction struct {
}

func (tx InvokeExpressionTransaction) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{}
}

func (tx InvokeExpressionTransaction) GetVersion() byte {
	return 1
}

func (tx *InvokeExpressionTransaction) GetID(scheme Scheme) ([]byte, error) {
	return nil, nil
}

func (tx InvokeExpressionTransaction) GetSender(scheme Scheme) (Address, error) {
	return WavesAddress{}, nil
}

func (tx InvokeExpressionTransaction) GetFee() uint64 {
	return 1
}

func (tx InvokeExpressionTransaction) GetTimestamp() uint64 {
	return 1
}

func (tx *InvokeExpressionTransaction) Validate(scheme Scheme) (Transaction, error) {
	return nil, nil
}

func (tx *InvokeExpressionTransaction) GenerateID(scheme Scheme) error {
	return nil
}

func (tx *InvokeExpressionTransaction) Sign(scheme Scheme, sk crypto.SecretKey) error {
	return nil
}

func (tx *InvokeExpressionTransaction) MerkleBytes(scheme Scheme) ([]byte, error) {
	return nil, nil
}

func (tx *InvokeExpressionTransaction) MarshalBinary() ([]byte, error) {
	return nil, nil
}

func (tx *InvokeExpressionTransaction) UnmarshalBinary([]byte, Scheme) error {
	return nil
}

func (tx *InvokeExpressionTransaction) BodyMarshalBinary() ([]byte, error) {
	return nil, nil
}

func (tx *InvokeExpressionTransaction) BinarySize() int {
	return 1
}

func (tx *InvokeExpressionTransaction) MarshalToProtobuf(scheme Scheme) ([]byte, error) {
	return nil, nil
}

func (tx *InvokeExpressionTransaction) UnmarshalFromProtobuf([]byte) error {
	return nil
}

func (tx *InvokeExpressionTransaction) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return nil, nil
}

func (tx *InvokeExpressionTransaction) UnmarshalSignedFromProtobuf([]byte) error {
	return nil
}
func (tx *InvokeExpressionTransaction) ToProtobuf(scheme Scheme) (*g.Transaction, error) {
	return nil, nil
}
func (tx *InvokeExpressionTransaction) ToProtobufSigned(scheme Scheme) (*g.SignedTransaction, error) {
	return nil, nil
}
